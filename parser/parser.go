// Package parser provides implementations of xml parsers.
//
// It's main purpose is to provide xml parser which is suitable for
// generating xtree structure used in xml comparison with xdiff algorithm.
//
// There are two parser implementations available:
//
//  - The standard implementation using encoding/xml library,
//  - and the custom in-place parser which is much faster but less robust.
//
// Use standard parser if you need robust and well tested parser and you
// don't care about it's performance. Use custom in-place parser if you
// need fast performance but can sacrifice robustness. In each case test it
// first on your own xml data.
//
// There are three functionalities that require xdiff xtree parser:
//  - Assign signatures and hashes to the nodes,
//  - Parse directories as nodes,
//  - Parse attributes as nodes,
//  - Optimizing execution speed and resource usage,
//      for more info see https://github.com/golang/go/issues/21823
//
// Example API usage:
//
//   p := parser.New()
//   // to parse from reader
//	 xtree, err := p.ParseReader(openedFileReader)
//   // to parse from bytes
//	 xtree, err := p.ParseBytes(filebytes)
//   // to parse from filepath
//   xtree, err := p.ParseFile("filename.xml")
//   // to parse from dirpath
//	 xtree, err := p.ParseDir("dirname")
//
package parser

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/ajankovic/xdiff/xtree"
)

// XDiff is a custom XML parser able to parse xml files to generate structures
// needed for doing xtree comparison in XDiff algorithm.
//
// Besides standard xml parsing, parsed xtree is also able to include nodes
// formed by the directory structure in which xml files reside. This is useful
// for comparing xml data which is spread across different directories and files.
//
// Parser is doing only basic transformation on the xml document to preserve documents as is
// for better comparison.
type XDiff struct {
	// Whether or not to validate closing tags.
	ValidateClosingTag bool
	// Handler for non-xml files encountered during directory traversal.
	NonXMLHandler func(f *os.File, fi os.FileInfo) (*xtree.Node, error)
	// Set document node name to filename when parsing xml by filename.
	SetDocumentFilename bool

	position int
	len      int
	data     []byte
}

// New instantiates new parser.
func New() *XDiff {
	return &XDiff{
		ValidateClosingTag: true,
	}
}

// ParseReader returns reference to the document node got by parsing bytes from the provided reader.
func (p *XDiff) ParseReader(r io.Reader) (*xtree.Node, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return p.ParseBytes(b)
}

// ParseFile returns reference to the document node got by parsing provided filepath.
func (p *XDiff) ParseFile(filepath string) (*xtree.Node, error) {
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	return p.ParseBytes(b)
}

// ParseDir returns reference to the directory node got by parsing provided dirpath.
// Each directory is parent to the child xml documents and documents have xtree structure
// got by parsing xml.
//
// RootDir
// |- Child1.xml
// | `- rootElement
// | 	|- childElement
// |	`- childElement
//  `- Child2.xml
// 	 `- rootElement
//
// Use NonXMLHandler flag on the parser to configure behavior for handling
// non-xml files found in the directories.
func (p *XDiff) ParseDir(path string) (*xtree.Node, error) {
	d, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	root := xtree.NewDirectory([]byte(filepath.Base(d.Name())))
	list, err := d.Readdir(-1)
	d.Close()
	if err != nil {
		return nil, err
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name() < list[j].Name() })
	for _, fi := range list {
		var ch *xtree.Node
		if fi.IsDir() {
			ch, err = p.ParseDir(filepath.Join(path, fi.Name()))
		} else {
			ch, err = p.parseDirFile(filepath.Join(path, fi.Name()), fi)
		}
		if err != nil {
			return nil, err
		}
		if ch != nil {
			root.AppendChild(ch)
		}
	}

	return root, nil
}

// parseDirFile tires to detect if the file is xml or not. If it's not it then
// tries to efficiently load non-xml files as nodes making sure they are not
// too big.
func (p *XDiff) parseDirFile(filepath string, fi os.FileInfo) (*xtree.Node, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b := make([]byte, 4)
	_, err = f.Read(b)
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	b, i := ensureUTF8(b)
	// Reset reader.
	_, err = f.Seek(0, 0)
	if err != nil {
		return nil, err
	}
	size := fi.Size() + bytes.MinRead
	var n *xtree.Node
	if b[i] == '<' {
		b, err = readAll(f, size)
		n, err = p.ParseBytes(b)
	} else if p.NonXMLHandler != nil {
		n, err = p.NonXMLHandler(f, fi)
	}
	if n != nil && len(n.Name) == 0 {
		n.Name = []byte(fi.Name())
	}
	return n, err
}

// readAll reads from r until an error or EOF and returns the data it read
// from the internal buffer allocated with a specified capacity.
func readAll(r io.Reader, capacity int64) (b []byte, err error) {
	var buf bytes.Buffer
	// If the buffer overflows, we will get bytes.ErrTooLarge.
	// Return that as an error. Any other panic remains.
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if panicErr, ok := e.(error); ok && panicErr == bytes.ErrTooLarge {
			err = panicErr
		} else {
			panic(e)
		}
	}()
	if int64(int(capacity)) == capacity {
		buf.Grow(int(capacity))
	}
	_, err = buf.ReadFrom(r)
	return buf.Bytes(), err
}

// ParseBytes returns reference to the document node parsed from provided bytes.
// For performance reasons returned structure will reuse byte array from the
// provided slice. Do not manipulate with it until you are done using the parsed structure.
// Pass the copy of the slice if you want to maintain ownership of the bytes.
func (p *XDiff) ParseBytes(b []byte) (*xtree.Node, error) {
	p.data, p.position = ensureUTF8(b)
	p.len = len(p.data)
	doc := xtree.NewDocument(nil)

	for p.position < p.len {
		p.skip(lookupWhitespace)
		if p.position == p.len-1 {
			// Clean exit.
			break
		}
		if p.currentByte() == '<' {
			p.position++
			parsed, err := p.parseNode()
			if err != nil {
				return doc, p.wrapError(err)
			}
			doc.AppendChild(parsed)
		} else {
			return doc, p.wrapError(fmt.Errorf("expected '<', but found %q", rune(p.data[p.position])))
		}
	}

	if err := xtree.Prepare(doc); err != nil {
		return doc, err
	}

	return doc, nil
}

// parseNode is the highest level parsing method; expects position to be after a '<'.
func (p *XDiff) parseNode() (*xtree.Node, error) {
	c := p.data[p.position]
	switch c {
	case '?':
		if err := p.skipBytes(4); err != nil {
			return nil, ErrUnexpectedEnd
		}
		x := p.sliceFrom(p.position - 3)
		if bytes.Compare([]byte("xml"), bytes.ToLower(x)) == 0 &&
			lookupWhitespace[p.currentByte()] == 1 {
			p.nextByte() // skip to next byte
			n, err := p.parseXMLDeclaration()
			if err != nil {
				return nil, err
			}
			return n, nil
		}
		p.position -= 3 // go back 4
		// not <?xml, so parse program instruction
		n, err := p.parseProcInstr()
		if err != nil {
			return nil, err
		}
		return n, nil
	case '!':
		// Parse proper subset
		switch c2 := p.nextByte(); c2 {
		// <!-
		case '-':
			if c2 := p.nextByte(); c2 == '-' {
				p.nextByte() // <!--
				n, err := p.parseComment()
				if err != nil {
					return nil, err
				}
				return n, nil
			}

		// <![
		case '[':
			err := p.skipBytes(1)
			if err != nil {
				return nil, err
			}
			// <![CDATA[]
			if !bytes.HasPrefix(p.sliceToEnd(), []byte("CDATA[")) {
				return nil, fmt.Errorf("unexpected data following <![")
			}
			p.skipBytes(6) // skip <![CDATA[
			n, err := p.parseCDATA()
			if err != nil {
				return nil, err
			}
			return n, nil
		// <!D
		case 'D':
			err := p.skipBytes(1)
			if err != nil {
				return nil, err
			}
			if bytes.HasPrefix(p.sliceToEnd(), []byte("OCTYPE")) && lookupWhitespace[p.data[p.position+6]] == 1 {
				// "<!DOCTYPE "
				p.skipBytes(6)
				n, err := p.parseDocType()
				if err != nil {
					return nil, err
				}
				return n, nil
			}
			fallthrough //? needed ?
		case 0: // zerobyte returned, not legal
			return nil, fmt.Errorf("unexpected end of data at %v", p.position)
		default: // Attempt to skip other, unrecognized node types starting with <!
			err := p.skipPastChar('>')
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("unrecognized node at %v", p.position)
		}
	default:
		return p.parseElement()
	}

	return nil, p.skipToChar('>')
}

// parseAttributes parses the attribute of an element, returns an AttributeNode.
func (p *XDiff) parseAttributes(element *xtree.Node) error {
	for lookupAttributeName[p.currentByte()] == 1 {
		start := p.position
		p.position++
		p.skip(lookupAttributeName)
		attrNode := xtree.NewNode(xtree.Attribute)
		attrNode.Name = p.sliceFrom(start)

		// skip whitespace
		p.skip(lookupWhitespace)
		// skip "="
		if p.currentByte() != '=' {
			return fmt.Errorf("expected '=' but found %q at position %v", p.data[p.position], p.position)
		}
		p.position++

		// skip whitespace after =
		p.skip(lookupWhitespace)
		q := p.currentByte()
		if q != '\'' && q != '"' {
			return fmt.Errorf("expected ' or \" but found %q at position %v", q, p.position)
		}
		p.position++ // Skip quote
		// Extract attribute value, and expand char refs in it
		start = p.position
		var value []byte
		if q == '\'' {
			value = p.skipAndExpandCharacterRefs(lookupAttributeValueSingle, lookupAttributeValueSingleNoProc)
		} else if q == '"' {
			value = p.skipAndExpandCharacterRefs(lookupAttributeValueDouble, lookupAttributeValueDoubleNoProc)
		} else {
			panic("should never happen")
		}
		if value == nil {
			return fmt.Errorf("error parsing attribute value")
		}
		// Set attribute value.
		attrNode.Value = value
		// Make sure end quote is present.
		if p.currentByte() != q {
			return fmt.Errorf("expected %v as end quote", q)
		}
		element.AppendChild(attrNode)
		p.position++             // Skip quote.
		p.skip(lookupWhitespace) // Skip whitespace after attribute value.
	}
	return nil
}

// parseElement parses element node.
func (p *XDiff) parseElement() (*xtree.Node, error) {
	currentElement := xtree.NewNode(xtree.Element)
	// Extract element name.
	start := p.position
	p.skip(lookupNodeName)
	if start == p.position {
		return nil, fmt.Errorf("error parsing node name")
	}
	currentElement.Name = p.data[start:p.position]

	// Skip whitespace between element name and attributes or >.
	p.skip(lookupWhitespace)

	// Parse attributes.
	err := p.parseAttributes(currentElement)
	if err != nil {
		return nil, err
	}
	// Ending type.
	c := p.currentByte()
	if c == '>' {
		p.position++
		if err := p.parseNodeContents(currentElement); err != nil {
			return nil, err
		}
	} else if c == '/' {
		if p.nextByte() != '>' {
			return nil, fmt.Errorf("expected '>' after '/' at position %v", p.position)
		}
		p.position++
	} else {
		return nil, fmt.Errorf("unknown end type error")
	}
	return currentElement, nil
}

// parseNodeContents recursively parses contents of the node.
func (p *XDiff) parseNodeContents(cn *xtree.Node) error {
	for {
		p.skip(lookupWhitespace)
		if p.currentByte() == '<' {
			if p.nextByte() == '/' {
				p.position++
				if p.ValidateClosingTag {
					start := p.position
					p.skip(lookupNodeName)
					closeTag := p.sliceFrom(start)
					if bytes.Compare(closeTag, cn.Name) != 0 {
						return fmt.Errorf("unexpected closing tag got '%s', expected '%s'", closeTag, cn.Name)
					}
				} else {
					p.skip(lookupNodeName)
				}
				p.skip(lookupWhitespace)
				if p.currentByte() != '>' {
					return fmt.Errorf("expected '>'")
				}
				p.position++ // Skip '>'.
				return nil
			}
			// Recursive call.
			parsed, err := p.parseNode()
			if err != nil {
				return err
			}
			cn.AppendChild(parsed)
		} else {
			err := p.parseAndAppendData(cn)
			if err != nil {
				return err
			}
		}
	}
}

// parseDocType returns the Doctype Node
func (p *XDiff) parseDocType() (*xtree.Node, error) {
	start := p.position
	// skip to > , we haven't closed the tag yet
	// since doctype can contain other elements, it can be somewhat tricky to detect in an efficient
	// way when it ends
	for p.currentByte() != '>' {
		if p.currentByte() == '[' { // beginning of elements
			p.skipBytes(1) // skip the '['
			for depth, insideElement := 1, false; depth > 0; {
				switch p.currentByte() {
				case '[':
					if !insideElement { // only count if not in a quote
						depth++
					}
				case ']':
					if !insideElement {
						depth--
					}
				case '>':
					insideElement = false
				}
				if bytes.HasPrefix(p.sliceToEnd(), []byte("<!")) {
					insideElement = true
				}
				p.nextByte()
			}
		} else {
			err := p.skipBytes(1)
			if err != nil {
				return nil, err
			}
		}
	}
	dt := xtree.NewNode(xtree.Doctype)
	dt.Value = p.sliceFrom(start)
	p.skipBytes(1)
	return dt, nil
}

func (p *XDiff) parseXMLDeclaration() (*xtree.Node, error) {
	nd := xtree.NewNode(xtree.Declaration)
	nd.Name = []byte("xml")
	p.skip(lookupWhitespace)
	p.parseAttributes(nd)
	// expect closing tags after attributes
	if !bytes.HasPrefix(p.sliceToEnd(), []byte("?>")) {
		p.position += 2
		return nil, fmt.Errorf("unexpected end of xml declaration. Expected '?>'")
	}
	p.position += 2
	return nd, nil
}

// parseProcInstr returns a Processing Instruction node.
func (p *XDiff) parseProcInstr() (*xtree.Node, error) {
	start := p.position
	p.skip(lookupNodeName)
	if start == p.position {
		return nil, fmt.Errorf("expected PI target")
	}
	pin := xtree.NewNode(xtree.ProcInstr)
	pin.Name = p.sliceFrom(start)
	p.skip(lookupWhitespace)
	start = p.position
	if err := p.skipToChars([]byte("?>")); err != nil {
		return nil, err
	}
	pin.Value = p.sliceFrom(start)

	p.position += 2
	return pin, nil
}

// parseCDATA creates a CDATA node
func (p *XDiff) parseCDATA() (*xtree.Node, error) {
	start := p.position // expects after <![CDATA[
	err := p.skipToChars([]byte("]]"))
	if err != nil {
		return nil, err
	}
	cd := xtree.NewNode(xtree.CData)
	cd.Value = p.sliceFrom(start)
	return cd, nil
}

// parseComment creates the Comment node
func (p *XDiff) parseComment() (*xtree.Node, error) {
	start := p.position
	// Skip to end of comments
	for !bytes.HasPrefix(p.sliceToEnd(), []byte("--")) {
		if err := p.skipBytes(1); err != nil {
			return nil, ErrUnexpectedEnd
		}
	}
	if err := p.skipBytes(2); err != nil {
		return nil, ErrUnexpectedEnd
	}
	if p.currentByte() != '>' {
		// there is '--' inside comment; not allowed in specs.
		return nil, fmt.Errorf("invalid '--' inside comment")
	}
	comment := xtree.NewNode(xtree.Comment)
	comment.Value = p.data[start : p.position-2]

	p.skipBytes(1)
	return comment, nil
}

func (p *XDiff) currentByte() byte {
	return p.data[p.position]
}

func (p *XDiff) nextByte() byte {
	p.position++
	if p.position > p.len-1 {
		return 0
	}
	return p.data[p.position]
}

func (p *XDiff) skipBytes(n int) error {
	if p.len <= p.position+n {
		return ErrUnexpectedEnd
	}
	p.position += n
	return nil
}

func (p *XDiff) skipPastChar(b byte) error {
	for {
		p.position++
		if p.position >= p.len-1 { // if, then we are at last char and cant advance
			return ErrUnexpectedEnd
		}
		if p.data[p.position] == b {
			p.position++ // advance, we know there is enough room
			return nil
		}
	}
}

func (p *XDiff) skipToChar(b byte) error {
	for {
		p.position++
		if p.position >= p.len { // if, then we are at last char and cant advance
			return ErrUnexpectedEnd
		}
		if p.data[p.position] == b {
			return nil
		}
	}
}

func (p *XDiff) skipToChars(b []byte) error {
	for ; p.position < p.len; p.position++ {
		if bytes.HasPrefix(p.data[p.position:], b) {
			return nil
		}
	}
	p.position-- // Avoid crash at the end of data.
	return ErrUnexpectedEnd
}

// skip characters until table evaluates to true, then return offset
func (p *XDiff) skip(table *[256]byte) {
	for ; p.position < p.len; p.position++ {
		if table[p.data[p.position]] != 1 {
			return
		}
	}
	p.position-- // Avoid crash at the end of data.
}

func (p *XDiff) sliceFrom(start int) []byte {
	return p.data[start:p.position]
}

func (p *XDiff) sliceForward(i int) []byte {
	if p.position+i <= p.len {
		return p.data[p.position : p.position+i]
	}
	return p.data[p.position:p.len]
}

func (p *XDiff) sliceToEnd() []byte {
	return p.data[p.position:]
}

// skipAndExpandCharacterRefs is used to parse both attribute values and node data while expanding entities
// since this function can overwrite the buffer, it returns a slice of the active area.
func (p *XDiff) skipAndExpandCharacterRefs(stopPred, stopPredPure *[256]byte) []byte {
	start := p.position
	p.skip(stopPredPure) // Fast path if no '&' is found.
	trail := p.position
	for c := p.currentByte(); stopPred[c] == 1; {
		if c == '&' {
			c = p.nextByte()
			switch c {
			case 'a': // &amp; &apos;
				if err := p.skipBytes(1); err == nil && bytes.HasPrefix(p.sliceToEnd(), []byte("mp;")) {
					p.position += 2
				} else if bytes.HasPrefix(p.sliceToEnd(), []byte("pos;")) {
					p.data[trail] = '\\' // overwrite
					p.position += 3
				}
			case 'q': // &quot;
				if err := p.skipBytes(1); err == nil && bytes.HasPrefix(p.sliceToEnd(), []byte("uot;")) {
					p.position += 3
				}
			case 'g': // &gt;
				if err := p.skipBytes(1); err == nil && bytes.HasPrefix(p.sliceToEnd(), []byte("t;")) {
					p.data[trail] = '>' // overwrite
					p.position++
				}
			case 'l': // &lt;
				if err := p.skipBytes(1); err == nil && bytes.HasPrefix(p.sliceToEnd(), []byte("t;")) {
					p.data[trail] = '<' // overwrite
					p.position++
				}
			default:
				trail++ // In case we can't find any entity move after to p.position
			case 0:
				panic("end of data")
			}
			// &#...; - assumes ASCII -- not implemented
		} else if trail < p.position {
			// If trail is lagging the position, we need to copy.
			p.data[trail] = p.data[p.position]
		}
		if c = p.nextByte(); c == 0 {
			return nil // error
		}
		trail++
	}
	return p.data[start:trail]
}

// parseAndAppendData adds a data node to the parent node.
func (p *XDiff) parseAndAppendData(parent *xtree.Node) error {
	value := p.skipAndExpandCharacterRefs(lookupText, lookupTextNoProc)
	if value == nil {
		return fmt.Errorf("unable to append data node")
	}
	n := xtree.NewNode(xtree.Data)
	n.Value = value
	parent.AppendChild(n)
	return nil
}

func (p *XDiff) wrapError(v error) error {
	const contextSize = 40
	start := max(p.position-contextSize, 0)
	stop := min(p.position, p.len)
	leftcontext := p.data[start:stop]
	start = min(p.position+1, p.len)
	stop = min(p.position+contextSize, p.len)
	rightcontext := p.data[start:stop]
	if p.position > p.len-1 {
		return fmt.Errorf("parser: %v\n%v", v, string(leftcontext))
	}
	return fmt.Errorf("parser: %v\n%v{%s}%v", v, string(leftcontext), string(p.currentByte()),
		string(rightcontext))
}

func max(x, y int) int {
	if x >= y {
		return x
	}
	return y
}

func min(x, y int) int {
	if x <= y {
		return x
	}
	return y
}

// ErrUnexpectedEnd represents attempted read beyond available data.
var ErrUnexpectedEnd = errors.New("parser: unexpected end of data")
