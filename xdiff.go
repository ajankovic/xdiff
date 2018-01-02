package xdiff

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Operation groups all allowed DOM operations.
type Operation int

func (o Operation) String() string {
	switch o {
	case Insert:
		return "Insert"
	case Update:
		return "Update"
	case Delete:
		return "Delete"
	case InsertSubtree:
		return "InsertSubtree"
	case DeleteSubtree:
		return "DeleteSubtree"
	}
	return "UnknownOperation"
}

const (
	// Insert leaf node.
	Insert Operation = iota + 1
	// Update leaf node value.
	Update
	// Delete leaf node.
	Delete
	// InsertSubtree operation.
	InsertSubtree
	// DeleteSubtree operation.
	DeleteSubtree
)

const (
	elementType   = "elem"
	attributeType = "attr"
	textType      = "text"
	commentType   = "comm"
	directiveType = "dire"
	procInstType  = "proc"
)

// Delta is a unit of change to the original doc that would change it into
// edited document.
type Delta struct {
	Op     Operation
	Node   *Node
	Update *Node
}

// CompareStrings is just helper to compare strings instead of readers.
func CompareStrings(original, edited string) ([]Delta, error) {
	return Compare(strings.NewReader(original), strings.NewReader(edited))
}

// Node coresponds to one xml node.
type Node struct {
	Parent      *Node
	LastChild   *Node
	PrevSibling *Node
	Name        string
	Content     []byte
	Hash        []byte
	Signature   string
}

// IsRoot determines if node is root node of the tree.
func (n *Node) IsRoot() bool {
	return n.Parent == nil
}

func (n *Node) String() string {
	hash := hex.EncodeToString(n.Hash)
	child := ""
	if n.LastChild == nil {
		child = string(n.Content)
	} else if n.LastChild.Name == "" {
		child = string(n.LastChild.Content)
	}
	if n.Name != "" {
		return fmt.Sprintf("[%s] %s:%s {%s}",
			n.Signature, n.Name, child, hash)
	}
	if n.Content != nil {
		return fmt.Sprintf("[%s] %q {%s}", n.Signature, n.Content, hash)
	}
	return fmt.Sprintf("[%s] ROOT {%s}", n.Signature, hash)
}

// Tree groups needed elements for comparing documents.
type Tree struct {
	Root  *Node
	Leafs []*Node
}

func (t *Tree) String() string {
	out := ""
	walk(t.Root, 0, func(n *Node, level int) {
		out += fmt.Sprintf("%s> %s\n",
			strings.Repeat("-", level), n.String())
	})
	return out
}

func walk(n *Node, level int, f func(n *Node, level int)) {
	for sbl := n; sbl != nil; sbl = sbl.PrevSibling {
		f(sbl, level)
		if sbl.LastChild != nil {
			walk(sbl.LastChild, level+1, f)
		}
	}
}

type hashes [][]byte

func (h hashes) Less(i, j int) bool {
	l := len(h[i])
	if len(h[i]) > len(h[j]) {
		l = len(h[j])
	}
	for k := 0; k < l; k++ {
		if h[i][k] < h[j][k] {
			return true
		}
	}
	return false
}

func (h hashes) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h hashes) Len() int {
	return len(h)
}

type buffer struct {
	bytes.Buffer
	err error
	n   int
}

func (b *buffer) Concat(strs ...string) (n int, err error) {
	for _, s := range strs {
		if b.err == nil {
			b.n, b.err = b.Buffer.WriteString(s)
		}
	}
	return b.n, b.err
}

func (b *buffer) Error() error {
	return b.err
}

func addChild(parent, child *Node) {
	if parent.LastChild == nil {
		parent.LastChild = child
	} else {
		child.PrevSibling = parent.LastChild
		parent.LastChild = child
	}
}

// ParseDoc parses xml and returns root node. Each node in the
// parsed tree is hashed.
func ParseDoc(r io.Reader) (*Tree, error) {
	dec := xml.NewDecoder(r)
	current := &Node{Signature: "/"}
	var leafs []*Node
	var buff buffer
	h := sha1.New()
loop:
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
			_, err := buff.Concat(parentSig(current.Signature), "/", el.Name.Local, "/", elementType)
			if err != nil {
				return nil, err
			}
			child := &Node{
				Name:      el.Name.Local,
				Parent:    current,
				Signature: buff.String(),
			}
			for _, a := range el.Attr {
				buff.Reset()
				_, err := buff.Concat(parentSig(child.Signature), "/", a.Name.Local, "/", attributeType)
				if err != nil {
					return nil, err
				}
				attr := &Node{
					Name:      a.Name.Local,
					Content:   []byte(a.Value),
					Signature: buff.String(),
				}
				_, err = io.WriteString(h, attributeType)
				if err != nil {
					return nil, err
				}
				_, err = io.WriteString(h, attr.Name)
				if err != nil {
					return nil, err
				}
				_, err = h.Write(attr.Content)
				if err != nil {
					return nil, err
				}
				attr.Hash = h.Sum(nil)
				addChild(child, attr)
			}
			addChild(current, child)
			current = child
		case xml.EndElement:
			// Compute node hash as sum of all children
			// with node type and name.
			_, err := io.WriteString(h, elementType)
			if err != nil {
				return nil, err
			}
			_, err = io.WriteString(h, current.Name)
			if err != nil {
				return nil, err
			}
			if current.LastChild != nil {
				var hs hashes
				for sbl := current.LastChild; sbl != nil; sbl = sbl.PrevSibling {
					hs = append(hs, sbl.Hash)
				}
				if len(hs) > 1 {
					// Sorting hashes to ensure unordered model.
					sort.Sort(hs)
				}
				for _, hash := range hs {
					_, err := h.Write(hash)
					if err != nil {
						return nil, err
					}
				}
			} else {
				leafs = append(leafs, current)
			}
			current.Hash = h.Sum(nil)
			if current.Parent == nil {
				h.Reset()
				break loop
			}
			current = current.Parent
		case xml.CharData:
			content := make([]byte, len(el))
			copy(content, el)
			if strings.TrimSpace(string(content)) == "" {
				continue
			}
			_, err := buff.Concat(parentSig(current.Signature), "/", textType)
			if err != nil {
				return nil, err
			}
			child := &Node{
				Parent:    current,
				Content:   content,
				Signature: buff.String(),
			}
			leafs = append(leafs, child)
			_, err = io.WriteString(h, textType)
			if err != nil {
				return nil, err
			}
			_, err = h.Write(child.Content)
			if err != nil {
				return nil, err
			}
			child.Hash = h.Sum(nil)
			addChild(current, child)
		case xml.Comment:
			_, err := buff.Concat(parentSig(current.Signature), "/", commentType)
			if err != nil {
				return nil, err
			}
			content := make([]byte, len(el))
			copy(content, el)
			child := &Node{
				Parent:    current,
				Content:   content,
				Signature: buff.String(),
			}
			leafs = append(leafs, child)
			_, err = io.WriteString(h, commentType)
			if err != nil {
				return nil, err
			}
			_, err = h.Write(child.Content)
			if err != nil {
				return nil, err
			}
			child.Hash = h.Sum(nil)
			addChild(current, child)
		case xml.Directive:
			_, err := buff.Concat(parentSig(current.Signature), "/", directiveType)
			if err != nil {
				return nil, err
			}
			content := make([]byte, len(el))
			copy(content, el)
			child := &Node{
				Parent:    current,
				Content:   content,
				Signature: buff.String(),
			}
			leafs = append(leafs, child)
			_, err = io.WriteString(h, directiveType)
			if err != nil {
				return nil, err
			}
			_, err = h.Write(child.Content)
			if err != nil {
				return nil, err
			}
			child.Hash = h.Sum(nil)
			addChild(current, child)
		case xml.ProcInst:
			_, err := buff.Concat(parentSig(current.Signature), "/", el.Target, "/", procInstType)
			if err != nil {
				return nil, err
			}
			child := &Node{
				Parent:    current,
				Content:   append([]byte(el.Target), el.Inst...),
				Signature: buff.String(),
			}
			leafs = append(leafs, child)
			_, err = io.WriteString(h, procInstType)
			if err != nil {
				return nil, err
			}
			_, err = h.Write(child.Content)
			if err != nil {
				return nil, err
			}
			child.Hash = h.Sum(nil)
			addChild(current, child)
		}
		buff.Reset()
		h.Reset()
	}
	for sbl := current.LastChild; sbl != nil; sbl = sbl.PrevSibling {
		_, err := h.Write(sbl.Hash)
		if err != nil {
			return nil, err
		}
	}
	current.Hash = h.Sum(nil)
	return &Tree{
		Root:  current,
		Leafs: leafs,
	}, nil
}

func parentSig(sig string) string {
	if len(sig) > 5 {
		return sig[:len(sig)-5]
	}
	return sig
}

// Compare runs X-Diff comparing algorithm on the provided arguments.
// Original reader is compared to edited and slice of deltas is
// returned.
// Compare returns nil, nil if there is no difference.
func Compare(original, edited io.Reader) ([]Delta, error) {
	oTree, err := ParseDoc(original)
	if err != nil {
		return nil, err
	}
	eTree, err := ParseDoc(edited)
	if err != nil {
		return nil, err
	}
	if bytesEqual(oTree.Root.Hash, eTree.Root.Hash) {
		return nil, nil
	}
	minMatch, distTbl, err := MinCostMatching(oTree, eTree)
	if err != nil {
		return nil, err
	}
	return EditScript(oTree.Root, eTree.Root, minMatch, distTbl), nil
}

// NodePair just pairs up two nodes for easier reference.
type NodePair struct {
	X *Node
	Y *Node
}

// MinCostMatch is table of matched node pairs.
type MinCostMatch map[NodePair]struct{}

// Add idempotently adds new match to the given index.
func (mcm MinCostMatch) Add(match NodePair) MinCostMatch {
	_, ok := mcm[match]
	if !ok {
		mcm[match] = struct{}{}
	}
	return mcm
}

// HasX detects if match has node in x position.
func (mcm MinCostMatch) HasX(n *Node) bool {
	for p := range mcm {
		if p.X == n {
			return true
		}
	}
	return false
}

// HasY detects if match has node in y position.
func (mcm MinCostMatch) HasY(n *Node) bool {
	for p := range mcm {
		if p.Y == n {
			return true
		}
	}
	return false
}

func (mcm MinCostMatch) String() string {
	out := ""
	for pair := range mcm {
		out += fmt.Sprintf("%s\n\n", pair)
	}
	return out
}

// DistTable maps pairs with costs.
type DistTable map[NodePair]int

// Has determines if cost for the pair is already set.
func (dt DistTable) Has(pair NodePair) bool {
	_, ok := dt[pair]
	return ok
}

// Set updates cost for pair in the table.
func (dt DistTable) Set(pair NodePair, cost int) {
	dt[pair] = cost
}

func (dt DistTable) String() string {
	out := ""
	for pair, cost := range dt {
		out += fmt.Sprintf("%s -> %d\n", pair, cost)
	}
	return out
}

func pairIn(pair NodePair, pairs []NodePair) bool {
	for _, p := range pairs {
		if p == pair {
			return true
		}
	}
	return false
}

// MinCostMatching finds minimum-cost matching between two trees.
func MinCostMatching(oTree, eTree *Tree) (MinCostMatch, DistTable, error) {
	minMatching := MinCostMatch{}
	distTbl := DistTable{}
	var exclude []string
	rootPair := NodePair{oTree.Root, eTree.Root}

	if oTree.Root.Signature != oTree.Root.Signature {
		return minMatching, distTbl, nil
	}
	minMatching.Add(rootPair)

	for oCh := oTree.Root.LastChild; oCh != nil; oCh = oCh.PrevSibling {
		for eCh := eTree.Root.LastChild; eCh != nil; eCh = eCh.PrevSibling {
			if oCh.LastChild != nil && eCh.LastChild != nil && bytesEqual(oCh.Hash, eCh.Hash) {
				// Remove trailing type from the signature for easier
				// prefix matching.
				sig := oCh.Signature[:len(oCh.Signature)-5]
				exclude = append(exclude, sig)
				minMatching.Add(NodePair{oCh, eCh})
			}
		}
	}
	n1 := oTree.Leafs
	n2 := eTree.Leafs
	for len(n1) > 0 && len(n2) > 0 {
		var parents1 []*Node
		var parents2 []*Node
		for _, x := range n1 {
			if isExcluded(x.Signature, exclude) {
				continue
			}
			if x.Parent != nil && !contains(x.Parent, parents1) {
				parents1 = append(parents1, x.Parent)
			}
			for _, y := range n2 {
				if isExcluded(y.Signature, exclude) {
					continue
				}
				if y.Parent != nil && !contains(y.Parent, parents2) {
					parents2 = append(parents2, y.Parent)
				}
				if x.Signature == y.Signature {
					computeDist(x, y, minMatching, distTbl)
				}
			}
		}
		n1 = parents1
		n2 = parents2
	}
	return minMatching, distTbl, nil
}

type costPair struct {
	NodePair
	Cost int
}

type costPairs []costPair

func (cp costPairs) Less(i, j int) bool {
	return cp[i].Cost < cp[j].Cost
}

func (cp costPairs) Swap(i, j int) {
	cp[i], cp[j] = cp[j], cp[i]
}

func (cp costPairs) Len() int {
	return len(cp)
}

func computeDist(x, y *Node, minMatching MinCostMatch, distTbl DistTable) {
	pair := NodePair{x, y}
	if x.LastChild == nil && y.LastChild == nil {
		minMatching.Add(pair)
		if bytesEqual(x.Content, y.Content) {
			distTbl.Set(pair, 0)
		} else {
			distTbl.Set(pair, 1)
		}
		return
	}
	// Group children of the non-leaf nodes by signature.
	groupX := make(map[string][]*Node)
	groupY := make(map[string][]*Node)
	for ch := x.LastChild; ch != nil; ch = ch.PrevSibling {
		groupX[ch.Signature] = append(groupX[ch.Signature], ch)
	}
	for ch := y.LastChild; ch != nil; ch = ch.PrevSibling {
		groupY[ch.Signature] = append(groupY[ch.Signature], ch)
	}
	costs := costPairs{}
	dist := 0
	// Calculate cost for current roots.
	for sig, childrenX := range groupX {
		if _, ok := groupY[sig]; ok {
			for _, chX := range childrenX {
				for _, chY := range groupY[sig] {
					pair := NodePair{chX, chY}
					c, ok := distTbl[pair]
					if !ok {
						computeDist(chX, chY, minMatching, distTbl)
						c = distTbl[pair]
					}
					costs = append(costs, costPair{NodePair: pair, Cost: c})
				}
			}
			if len(costs) > 1 {
				sort.Sort(costs)
			}
			for _, cost := range costs {
				if contains(cost.X, childrenX) && contains(cost.Y, groupY[sig]) {
					if !minMatching.HasX(cost.X) && !minMatching.HasY(cost.Y) {
						minMatching.Add(cost.NodePair)
					}
					// Calculate cost for mapped nodes.
					d, ok := distTbl[cost.NodePair]
					if ok {
						dist += d
						continue
					}
					// Delete + Insert cost.
					dist++
					continue
				}
				// Handle unmapped nodes.
				// Delete cost.
				dist++
				// Insert cost.
				dist++
			}
		}
	}
	distTbl.Set(pair, dist)
}

func contains(n *Node, nodes []*Node) bool {
	for _, x := range nodes {
		if x == n {
			return true
		}
	}
	return false
}

// exclude slice has parent node signature so we are checking
// if signature is prefixed with them.
func isExcluded(sig string, exclude []string) bool {
	for _, excl := range exclude {
		if strings.HasPrefix(sig, excl) {
			return true
		}
	}
	return false
}

// EditScript generates slice of deltas that forms minimum-cost edit
// script to transform original tree into edited tree.
func EditScript(oRoot, eRoot *Node, minCostM MinCostMatch, distTbl DistTable) []Delta {
	rootPair := NodePair{oRoot, eRoot}
	_, ok := minCostM[rootPair]
	if !ok {
		return []Delta{
			Delta{Op: DeleteSubtree, Node: oRoot},
			Delta{Op: InsertSubtree, Node: eRoot},
		}
	}
	if distTbl[rootPair] == 0 {
		return nil
	}
	var script []Delta
	for x := oRoot.LastChild; x != nil; x = x.PrevSibling {
		for y := eRoot.LastChild; y != nil; y = y.PrevSibling {
			if bytesEqual(x.Hash, y.Hash) {
				continue
			}
			pair := NodePair{x, y}
			if _, ok := minCostM[pair]; ok {
				if x.LastChild == nil && y.LastChild == nil {
					if distTbl[pair] == 0 {
						continue
					}
					script = append(script, Delta{Op: Update, Node: x, Update: y})
					continue
				}
				script = append(script, EditScript(x, y, minCostM, distTbl)...)
			}
		}
		if !minCostM.HasX(x) {
			if x.LastChild == nil {
				script = append(script, Delta{Op: Delete, Node: x})
				continue
			}
			script = append(script, Delta{Op: DeleteSubtree, Node: x})
		}
	}
	for y := eRoot.LastChild; y != nil; y = y.PrevSibling {
		if !minCostM.HasY(y) {
			if y.LastChild == nil {
				script = append(script, Delta{Op: Insert, Node: y})
				continue
			}
			script = append(script, Delta{Op: InsertSubtree, Node: y})
		}
	}
	return script
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
