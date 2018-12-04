//go:generate stringer -type=NodeType

package xtree

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"
)

// NodeType is describing possible node types.
type NodeType int

// NodeType values.
const (
	// Node is not xml file.
	NotXML NodeType = iota
	// A directory node. Name is same as directory name. Value is empty.
	Directory
	// A document node. If result of directory parsing name is same as the filename otherwise it's empty. Value is empty.
	Document
	// An element node. Name contains element name. Value contains text of first data node.
	Element
	// An attribute node. Name contains attribute name. Value contains attribute value.
	Attribute
	// A data node. Name is empty. Value contains text data.
	Data
	// A CDATA node. Name is empty. Value contains text data.
	CData
	// A comment node. Name is empty. Value contains comment text.
	Comment
	// A declaration node. Name and value are empty. Declaration parameters (version, encoding and standalone) are in node attributes.
	Declaration
	// A DOCTYPE node. Name is empty. Value contains DOCTYPE text.
	Doctype
	// A ProcInstr node. Name contains target. Value contains instructions.
	ProcInstr
)

// Signature returns byte representation of the signature.
func (nt NodeType) Signature() []byte {
	return []byte(nt.String())
}

// Node represents node of the xtree.
type Node struct {
	Type              NodeType
	Parent            *Node
	FirstChild        *Node
	NextSibling       *Node
	PrevSiblingCyclic *Node
	Name              []byte
	Value             []byte
	Hash              []byte
	Signature         []byte
}

// NewNode creates new node of type t.
func NewNode(t NodeType) *Node {
	return &Node{
		Type: t,
	}
}

// NewDirectory creates new node with Directory type.
func NewDirectory(name []byte) *Node {
	return &Node{
		Type: Directory,
		Name: []byte(name),
	}
}

// NewNotXML creates new node with NotXML type.
func NewNotXML(name []byte, value []byte) *Node {
	return &Node{
		Type:  NotXML,
		Name:  name,
		Value: value,
	}
}

// NewDocument creates new node with Document type.
func NewDocument(name []byte) *Node {
	return &Node{
		Type: Document,
		Name: []byte(name),
	}
}

// NewElement creates new node with Element type.
func NewElement(name []byte) *Node {
	return &Node{
		Type: Element,
		Name: []byte(name),
	}
}

// NewAttribute creates new node with Attribute type.
func NewAttribute(name []byte, value []byte) *Node {
	return &Node{
		Type:  Attribute,
		Name:  name,
		Value: value,
	}
}

// NewData creates new node with Data type.
func NewData(value []byte) *Node {
	return &Node{
		Type:  Data,
		Value: value,
	}
}

// NewCData creates new node with CData type.
func NewCData(value []byte) *Node {
	return &Node{
		Type:  CData,
		Value: value,
	}
}

// NewComment creates new node with Comment type.
func NewComment(value []byte) *Node {
	return &Node{
		Type:  Comment,
		Value: value,
	}
}

// NewDeclaration creates new node with Declaration type.
func NewDeclaration() *Node {
	return &Node{
		Type: Declaration,
		Name: []byte("xml"),
	}
}

// NewDoctype creates new node with Doctype type.
func NewDoctype(value []byte) *Node {
	return &Node{
		Type:  Doctype,
		Value: value,
	}
}

// NewProcInstr creates new node with ProcInstr type.
func NewProcInstr(name []byte, value []byte) *Node {
	return &Node{
		Type:  ProcInstr,
		Name:  name,
		Value: value,
	}
}

// Implements stringer.
func (n Node) String() string {
	name := string(n.Name)
	value := string(n.Value)
	sig := string(n.Signature)
	hash := hex.EncodeToString(n.Hash)
	if hash != "" {
		hash = hash[:6]
	}
	return fmt.Sprintf("(%s n:%s v:%s s:%s h:%s)", n.Type, name, strings.Replace(value, "\n", "\\n", -1), sig, hash)
}

// Children returns list of immediate children of the node.
func (n *Node) Children() []*Node {
	var children []*Node
	for next := n.FirstChild; next != nil; next = next.NextSibling {
		children = append(children, next)
	}
	return children
}

// CalculateHash sets hash value of the node.
func (n *Node) CalculateHash(h hash.Hash) error {
	if h != nil {
		h.Reset()
	} else {
		h = sha1.New()
	}
	_, err := h.Write([]byte{byte(n.Type)})
	if err != nil {
		return err
	}
	_, err = h.Write(n.Name)
	if err != nil {
		return err
	}
	_, err = h.Write(n.Value)
	if err != nil {
		return err
	}
	for next := n.FirstChild; next != nil; next = next.NextSibling {
		_, err = h.Write(next.Hash)
		if err != nil {
			return err
		}
	}
	n.Hash = h.Sum(nil)
	return nil
}

// CalculateSignature sets signature value of the node.
func (n *Node) CalculateSignature() {
	var sig [][]byte
	for parent := n.Parent; parent != nil; parent = parent.Parent {
		sig = append([][]byte{parent.Name}, sig...)
	}
	if n.Parent != nil {
		if len(n.Name) > 0 {
			sig = append(sig, n.Name, n.Type.Signature())
		} else {
			sig = append(sig, n.Type.Signature())
		}
		n.Signature = bytes.Join(sig, []byte{0x2f})
	} else {
		n.Signature = []byte{0x2f}
	}
}

// LastChild returns last child of the node.
func (n *Node) LastChild() *Node {
	if n.FirstChild == nil {
		return nil
	}
	return n.FirstChild.PrevSiblingCyclic
}

// PrevSibling returns previous sibling of the node.
func (n *Node) PrevSibling() *Node {
	if n.PrevSiblingCyclic.NextSibling == nil {
		return nil
	}
	return n.PrevSiblingCyclic
}

// Remove removes node from the xtree.
func (n *Node) Remove() {
	if n.Parent != nil && n.Parent.FirstChild == n {
		n.Parent.FirstChild = n.NextSibling
	}
	if n.NextSibling != nil {
		n.NextSibling.PrevSiblingCyclic = n.PrevSiblingCyclic
	}
	if n.PrevSiblingCyclic.NextSibling == n {
		n.PrevSiblingCyclic.NextSibling = n.NextSibling
	}
	n.NextSibling = nil
	n.PrevSiblingCyclic = nil
	n.Parent = nil
}

// AppendChild appends child node to the node.
func (n *Node) AppendChild(child *Node) *Node {
	if n.FirstChild == nil {
		n.FirstChild = child
		child.PrevSiblingCyclic = child
	} else {
		child.PrevSiblingCyclic = n.LastChild()
		n.LastChild().NextSibling = child
		n.FirstChild.PrevSiblingCyclic = child
	}
	child.Parent = n
	return n
}

// Prepare traverses the xtree rooted at n and sets signature and hash for
// all nodes.
func Prepare(n *Node) error {
	h := sha1.New()
	s := Stack{}
	var lastVisited *Node
	for !s.IsEmpty() || n != nil {
		if n != nil {
			if !s.Push(n) {
				return fmt.Errorf("xtree: maximum xtree depth of %d reached", maxStackSize)
			}
			n = n.FirstChild
		} else {
			peekNode := s.Peek()
			if peekNode.NextSibling != nil && lastVisited != peekNode.NextSibling {
				n = peekNode.NextSibling
			} else {
				peekNode.CalculateSignature()
				peekNode.CalculateHash(h)
				h.Reset()
				lastVisited, _ = s.Pop()
			}
		}
	}
	return nil
}

// maxStackSize defines maximum depth of the nested nodes.
const maxStackSize = 10000

// Stack is auxiliary data structure for iterative xtree traversal.
type Stack struct {
	top  int
	data [maxStackSize]*Node
}

// Push pushes node to the top of the stack.
func (s *Stack) Push(i *Node) bool {
	if s.top == len(s.data) {
		return false
	}
	s.data[s.top] = i
	s.top++
	return true
}

// Pop removes node from the top of the stack.
func (s *Stack) Pop() (*Node, bool) {
	if s.top == 0 {
		return nil, false
	}
	i := s.data[s.top-1]
	s.top--
	return i, true
}

// Peek returns node at the top of the stack without removing it.
func (s *Stack) Peek() *Node {
	return s.data[s.top-1]
}

// Get returns stack as slice of nodes without removing them.
func (s *Stack) Get() []*Node {
	return s.data[:s.top]
}

// IsEmpty returns true if stack is empty.
func (s *Stack) IsEmpty() bool {
	return s.top == 0
}

// Empty clears stack contents.
func (s *Stack) Empty() {
	s.top = 0
}

// Len counts number of items on the stack.
func (s *Stack) Len() int {
	return s.top
}
