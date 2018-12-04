package xtree

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// TextEncoder knows how to encode node tree as text format suitable for command line output.
type TextEncoder struct {
	p textPrinter
}

// NewTextEncoder creates new textual encoder that writes text into the writer.
func NewTextEncoder(w io.Writer) *TextEncoder {
	return &TextEncoder{textPrinter{Writer: bufio.NewWriter(w)}}
}

// Encode writes node tree rooted at n to the stream.
func (enc *TextEncoder) Encode(n *Node) error {
	if err := enc.p.printText(n); err != nil {
		return err
	}
	return enc.p.Flush()
}

type textPrinter struct {
	*bufio.Writer
}

func (p *textPrinter) writeLine(current *Node) error {
	indent := func(n *Node) string {
		var out []string
		for parent := n.Parent; parent != nil; parent = parent.Parent {
			if parent.NextSibling != nil {
				out = append([]string{"│  "}, out...)
			} else {
				out = append([]string{"   "}, out...)
			}
		}
		return strings.Join(out, "")
	}
	branching := func(n *Node) string {
		out := ""
		if n.Parent == nil {
			out = "───"
		} else if n.NextSibling == nil {
			out = "└──"
		} else {
			out = "├──"
		}
		if n.FirstChild != nil {
			out += "┐"
		} else {
			out += "─"
		}
		return out
	}
	_, err := p.WriteString(fmt.Sprintf("%s%s%s\n",
		indent(current),
		branching(current),
		current.String()))
	return err
}

func (p *textPrinter) printText(n *Node) error {
	if n == nil {
		return nil
	}
	s := Stack{}
	s.Push(n)
	for !s.IsEmpty() {
		current, _ := s.Pop()
		if err := p.writeLine(current); err != nil {
			return err
		}
		if current.NextSibling != nil {
			if !s.Push(current.NextSibling) {
				return fmt.Errorf("tree: maximum tree depth of %d reached", maxStackSize)
			}
		}
		if current.FirstChild != nil {
			if !s.Push(current.FirstChild) {
				return fmt.Errorf("tree: maximum tree depth of %d reached", maxStackSize)
			}
		}
	}
	return nil
}

// XMLEncoder knows how to encode node tree to XML format.
type XMLEncoder struct {
	p xmlPrinter
}

// NewXMLEncoder creates new XMLEncoder that writes xml into the writer.
func NewXMLEncoder(w io.Writer) *XMLEncoder {
	return &XMLEncoder{xmlPrinter{Writer: bufio.NewWriter(w)}}
}

// Encode writes node tree rooted at n to the stream.
func (enc *XMLEncoder) Encode(n *Node) error {
	if err := enc.p.printXML(n); err != nil {
		return err
	}
	return enc.p.Flush()
}

// Indent sets the encoder to generate XML in which each element
// begins on a new indented line that starts with one or more copies
// of indent according to the nesting depth.
func (enc *XMLEncoder) Indent(indent string) {
	enc.p.indent = indent
}

type xmlPrinter struct {
	*bufio.Writer

	indent string
}

func (p *xmlPrinter) writeIndent(n *Node) error {
	if len(p.indent) == 0 {
		return nil
	}
	for parent := n.Parent; parent != nil && parent.Type != Document; parent = parent.Parent {
		_, err := p.WriteString(p.indent)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *xmlPrinter) writeLE() error {
	if len(p.indent) == 0 {
		return nil
	}
	return p.WriteByte('\n')
}

func (p *xmlPrinter) writeNode(n *Node) error {
	switch n.Type {
	case Document:
	case Element:
		if err := p.writeIndent(n); err != nil {
			return err
		}
		err := p.WriteByte('<')
		if err != nil {
			return err
		}
		_, err = p.Write(n.Name)
		if err != nil {
			return err
		}
		if n.FirstChild == nil || (n.FirstChild != nil && n.FirstChild.Type != Attribute) {
			err := p.WriteByte('>')
			if err != nil {
				return err
			}
			if n.FirstChild != nil && n.FirstChild.Type != Data {
				if err := p.writeLE(); err != nil {
					return err
				}
			}
		}
	case Attribute:
		err := p.WriteByte(' ')
		if err != nil {
			return err
		}
		_, err = p.Write(n.Name)
		if err != nil {
			return err
		}
		_, err = p.WriteString("=\"")
		if err != nil {
			return err
		}
		_, err = p.Write(n.Value)
		if err != nil {
			return err
		}
		err = p.WriteByte('"')
		if err != nil {
			return err
		}
		if n.NextSibling == nil || n.NextSibling.Type != Attribute {
			if n.Parent.Type != Declaration {
				_, err = p.WriteString(">")
				if err != nil {
					return err
				}
				if n.NextSibling != nil && n.NextSibling.Type != Data {
					if err := p.writeLE(); err != nil {
						return err
					}
				}
			}
		}
	case Data:
		_, err := p.Write(n.Value)
		if err != nil {
			return err
		}
	case CData:
		_, err := p.WriteString("<![CDATA[")
		if err != nil {
			return err
		}
		_, err = p.Write(n.Value)
		if err != nil {
			return err
		}
		_, err = p.WriteString("]]")
		if err != nil {
			return err
		}
	case Comment:
		if err := p.writeIndent(n); err != nil {
			return err
		}
		_, err := p.WriteString("<!--")
		if err != nil {
			return err
		}
		_, err = p.Write(n.Value)
		if err != nil {
			return err
		}
		_, err = p.WriteString("-->")
		if err != nil {
			return err
		}
		if err := p.writeLE(); err != nil {
			return err
		}
	case Declaration:
		_, err := p.WriteString("<?xml")
		if err != nil {
			return err
		}
		if n.Value != nil {
			err := p.WriteByte(' ')
			if err != nil {
				return err
			}
			_, err = p.Write(n.Value)
			if err != nil {
				return err
			}
			_, err = p.WriteString("?>")
			if err != nil {
				return err
			}
			if err := p.writeLE(); err != nil {
				return err
			}
		}
	case Doctype:
		_, err := p.WriteString("<!DOCTYPE ")
		if err != nil {
			return err
		}
		_, err = p.Write(n.Value)
		if err != nil {
			return err
		}
		err = p.WriteByte('>')
		if err != nil {
			return err
		}
		if err := p.writeLE(); err != nil {
			return err
		}
	case ProcInstr:
		_, err := p.WriteString("<?")
		if err != nil {
			return err
		}
		_, err = p.Write(n.Value)
		if err != nil {
			return err
		}
		_, err = p.WriteString("?>")
		if err != nil {
			return err
		}
		if err := p.writeLE(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("tree: invalid xml node type %s", n.Type)
	}
	return nil
}

func (p *xmlPrinter) writeClosing(n *Node) error {
	switch n.Type {
	case Document, Attribute, Data, CData, Comment, Doctype, ProcInstr:
	case Element:
		if n.LastChild() != nil && n.LastChild().Type != Data {
			if err := p.writeIndent(n); err != nil {
				return err
			}
		}
		_, err := p.WriteString("</")
		if err != nil {
			return err
		}
		_, err = p.Write(n.Name)
		if err != nil {
			return err
		}
		err = p.WriteByte('>')
		if err != nil {
			return err
		}
		if err := p.writeLE(); err != nil {
			return err
		}
	case Declaration:
		_, err := p.WriteString("?>")
		if err != nil {
			return err
		}
		if err := p.writeLE(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("tree: closing invalid node type %s", n.Type)
	}
	return nil
}

func (p *xmlPrinter) printXML(n *Node) error {
	if n == nil {
		return nil
	}
	s := Stack{}
	s.Push(n)
	for !s.IsEmpty() {
		current, _ := s.Pop()
		if err := p.writeNode(current); err != nil {
			return err
		}
		if current.NextSibling != nil {
			if !s.Push(current.NextSibling) {
				return fmt.Errorf("tree: maximum tree depth of %d reached", maxStackSize)
			}
		}
		if current.FirstChild != nil {
			if !s.Push(current.FirstChild) {
				return fmt.Errorf("tree: maximum tree depth of %d reached", maxStackSize)
			}
		}
		if current.NextSibling == nil && current.FirstChild == nil {
			if err := p.writeClosing(current); err != nil {
				return err
			}
			for closing := current.Parent; closing != nil; closing = closing.Parent {
				if !s.IsEmpty() && closing == s.Peek().Parent {
					break
				}
				if err := p.writeClosing(closing); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// TextString is a helper function to quickly get string representation of the node tree
// in text format.
func TextString(n *Node) (string, error) {
	buf := bytes.NewBuffer(nil)
	enc := NewTextEncoder(buf)
	if err := enc.Encode(n); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// XMLString is a helper function to quickly get string representation of the node tree
// in xml format.
func XMLString(n *Node) (string, error) {
	buf := bytes.NewBuffer(nil)
	enc := NewXMLEncoder(buf)
	enc.Indent("  ")
	if err := enc.Encode(n); err != nil {
		return "", err
	}
	return buf.String(), nil
}
