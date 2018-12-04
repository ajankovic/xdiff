package parser

import (
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ajankovic/xdiff/xtree"
)

// Standard is a standard parser using encoding/xml sax parser as engine.
type Standard struct {
	// Handler for non-xml files encountered during directory traversal.
	NonXMLHandler func(f *os.File, fi os.FileInfo) (*xtree.Node, error)
}

// NewStandard instantiates new standard parser.
func NewStandard() *Standard {
	return &Standard{}
}

// ParseReader returns reference to the document node got by parsing bytes from the provided
// reader.
func (p *Standard) ParseReader(r io.Reader) (*xtree.Node, error) {
	dec := xml.NewDecoder(r)
	doc := xtree.NewNode(xtree.Document)
	current := doc
	for {
		tok, err := dec.Token()
		if err != nil && err != io.EOF {
			return nil, err
		}
		if tok == nil {
			break
		}
		switch el := tok.(type) {
		case xml.StartElement:
			child := &xtree.Node{
				Type: xtree.Element,
				Name: []byte(el.Name.Local),
			}
			for _, a := range el.Attr {
				child.AppendChild(&xtree.Node{
					Type:  xtree.Attribute,
					Name:  []byte(a.Name.Local),
					Value: []byte(a.Value),
				})
			}
			current.AppendChild(child)
			current = child
		case xml.EndElement:
			current = current.Parent
		case xml.CharData:
			content := make([]byte, len(el))
			copy(content, el)
			if strings.TrimSpace(string(content)) == "" {
				continue
			}
			child := &xtree.Node{
				Type:  xtree.Data,
				Value: content,
			}
			current.AppendChild(child)
		case xml.Comment:
			content := make([]byte, len(el))
			copy(content, el)
			child := &xtree.Node{
				Type:  xtree.Comment,
				Value: content,
			}
			current.AppendChild(child)
		case xml.Directive:
			content := make([]byte, len(el))
			copy(content, el)
			child := &xtree.Node{
				Type:  xtree.Doctype,
				Value: content,
			}
			current.AppendChild(child)
		case xml.ProcInst:
			var child *xtree.Node
			if el.Target == "xml" {
				content := make([]byte, len(el.Inst))
				copy(content, el.Inst)
				child = &xtree.Node{
					Type: xtree.Declaration,
					Name: []byte("xml"),
				}
				attrs, err := p.parseAttributes(content)
				if err != nil {
					return nil, err
				}
				for _, attr := range attrs {
					child.AppendChild(attr)
				}
			} else {
				child = &xtree.Node{
					Type:  xtree.ProcInstr,
					Value: append([]byte(el.Target), el.Inst...),
				}
			}
			current.AppendChild(child)
		}
	}
	if err := xtree.Prepare(doc); err != nil {
		return doc, err
	}
	return doc, nil
}

// parseAttributes is  parsing declaration attributes since standard library
// parses it only as byte content.
//
// TODO(office@ajankovic.com) This is a happy path implementation so proper error handling
// is needed to cover issues with invalid attribute format.
func (p *Standard) parseAttributes(content []byte) ([]*xtree.Node, error) {
	var attrs []*xtree.Node
	buf := bytes.NewBuffer(bytes.TrimSpace(content))
	for buf.Len() > 0 {
		name, err := buf.ReadBytes('=')
		if err != nil {
			return nil, err
		}
		_, err = buf.ReadBytes('"')
		if err != nil {
			return nil, err
		}
		value, err := buf.ReadBytes('"')
		if err != nil {
			return nil, err
		}
		attrs = append(attrs, &xtree.Node{
			Type:  xtree.Attribute,
			Name:  bytes.TrimSpace(name[:len(name)-1]),
			Value: bytes.TrimSpace(value[:len(value)-1]),
		})
	}

	return attrs, nil
}

// ParseFile returns reference to the document node got by parsing provided filepath.
func (p *Standard) ParseFile(filepath string) (*xtree.Node, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	return p.ParseReader(f)
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
// Use ErrorOnNonXMLFiles flag on the parser to configure behavior for handling
// non xml files found in the directories.
func (p *Standard) ParseDir(dirpath string) (*xtree.Node, error) {
	d, err := os.Open(dirpath)
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
			ch, err = p.ParseDir(filepath.Join(dirpath, fi.Name()))
		} else {
			ch, err = p.parseDirFile(filepath.Join(dirpath, fi.Name()), fi)
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
func (p *Standard) parseDirFile(filepath string, fi os.FileInfo) (*xtree.Node, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b := make([]byte, 4)
	_, err = f.Read(b)
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

// ParseBytes returns reference to the document node parsed from provided bytes.
func (p *Standard) ParseBytes(b []byte) (*xtree.Node, error) {
	return p.ParseReader(bytes.NewBuffer(b))
}
