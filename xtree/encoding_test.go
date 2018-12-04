package xtree

import (
	"bytes"
	"testing"
)

func testTree() *Node {
	doc := NewDocument(nil)
	root := NewElement([]byte("root"))
	child1 := NewElement([]byte("child1"))
	child2 := NewElement([]byte("child2"))
	subchild := NewAttribute([]byte("subchild"), []byte("value"))
	doc.AppendChild(root)
	root.AppendChild(child1)
	root.AppendChild(child2)
	child1.AppendChild(subchild)
	Prepare(doc)
	return doc
}

func TestTextEncoding(t *testing.T) {
	doc := testTree()
	tests := []struct {
		name string
		n    *Node
		want string
		err  error
	}{
		{
			"basic",
			doc,
			`───┐(Document n: v: s:/ h:75a77b)
   └──┐(Element n:root v: s:/root/Element h:843463)
      ├──┐(Element n:child1 v: s:/root/child1/Element h:1ee593)
      │  └───(Attribute n:subchild v:value s:/root/child1/subchild/Attribute h:ea2033)
      └───(Element n:child2 v: s:/root/child2/Element h:597330)
`,
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			enc := NewTextEncoder(w)
			if err := enc.Encode(tt.n); err != tt.err {
				t.Errorf("TextEncoder.Encode() = error %v\nexpected: %v", err, tt.err)
			}
			if w.String() != tt.want {
				t.Errorf("TextEncoder.Encode() = \n%v\nexpected:\n%v", w.String(), tt.want)
			}
		})
	}
}

func TestXMLEncoding(t *testing.T) {
	doc := testTree()
	tests := []struct {
		name string
		n    *Node
		want string
		err  error
	}{
		{
			"basic",
			doc,
			`<root><child1 subchild="value"></child1><child2></child2></root>`,
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			enc := NewXMLEncoder(w)
			if err := enc.Encode(tt.n); err != tt.err {
				t.Errorf("TextEncoder.Encode() = error %v\nexpected: %v", err, tt.err)
			}
			if w.String() != tt.want {
				t.Errorf("TextEncoder.Encode() = \n%v\nexpected:\n%v", w.String(), tt.want)
			}
		})
	}
}
