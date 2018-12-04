// Package xdiff provides an implementation of the X-Diff algorithm used for finding
// minimum edit script between two xml documents.
//
//go:generate stringer -type=Operation
package xdiff

import (
	"fmt"
	"sort"

	"github.com/ajankovic/xdiff/xtree"
)

// Operation defines possible modifying operations of the edit script.
type Operation int

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

// Delta is a unit of change to the original doc that would change it into
// edited document.
type Delta struct {
	Operation Operation
	Subject   *xtree.Node
	Object    *xtree.Node
}

// nodePair just pairs up two nodes for easier reference.
type nodePair struct {
	Left  *xtree.Node
	Right *xtree.Node
}

// minCostMatch is table of matched node pairs.
type minCostMatch map[nodePair]struct{}

// Add idempotently adds new match to the given index.
func (mcm minCostMatch) Add(match nodePair) minCostMatch {
	_, ok := mcm[match]
	if !ok && !mcm.HasLeft(match.Left) && !mcm.HasRight(match.Right) {
		mcm[match] = struct{}{}
		leftParent := match.Left.Parent
		rightParent := match.Right.Parent
		for leftParent != nil && rightParent != nil {
			mcm[nodePair{leftParent, rightParent}] = struct{}{}
			leftParent = leftParent.Parent
			rightParent = rightParent.Parent
		}
	}
	return mcm
}

// HasPair returns true if match table has pair matched.
func (mcm minCostMatch) HasPair(match nodePair) bool {
	_, ok := mcm[match]
	return ok
}

// HasLeft returns true if match table has the node in left position.
func (mcm minCostMatch) HasLeft(n *xtree.Node) bool {
	for p := range mcm {
		if p.Left == n {
			return true
		}
	}
	return false
}

// HasRight returns true if match table has the node in right position.
func (mcm minCostMatch) HasRight(n *xtree.Node) bool {
	for p := range mcm {
		if p.Right == n {
			return true
		}
	}
	return false
}

func (mcm minCostMatch) String() string {
	out := ""
	for pair := range mcm {
		out += fmt.Sprintf("%s\n", pair)
	}
	return out
}

// distTable holds editing distance from one pair to the other.
type distTable map[nodePair]int

// Has determines if distance for the pair is already set.
func (dt distTable) Has(pair nodePair) bool {
	_, ok := dt[pair]
	return ok
}

// Set updates pair distance.
func (dt distTable) Set(pair nodePair, cost int) {
	dt[pair] = cost
}

func (dt distTable) String() string {
	out := ""
	for pair, cost := range dt {
		out += fmt.Sprintf("%s -> %d\n", pair, cost)
	}
	return out
}

type costPair struct {
	nodePair
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

// Compare generates slice of deltas that forms minimum-cost edit
// script to transform the left xtree into the right xtree.
func Compare(left *xtree.Node, right *xtree.Node) ([]Delta, error) {
	if bytesEqual(left.Hash, right.Hash) {
		return nil, nil
	}

	reduceMatchingSpace(left, right)

	distTbl := make(distTable)
	minCostM := make(minCostMatch)
	minCostM.Add(nodePair{left, right})
	var leftS, rightS xtree.Stack
	var leftLastVisited, rightLastVisited *xtree.Node
	var l, r = left, right
	for !leftS.IsEmpty() || l != nil {
		if l != nil {
			leftS.Push(l)
			l = l.FirstChild
		} else {
			leftPeek := leftS.Peek()
			if leftPeek.NextSibling != nil && leftLastVisited != leftPeek.NextSibling {
				l = leftPeek.NextSibling
			} else {
				r = right
				for !rightS.IsEmpty() || r != nil {
					if r != nil {
						rightS.Push(r)
						r = r.FirstChild
					} else {
						rightPeek := rightS.Peek()
						if rightPeek.NextSibling != nil && rightLastVisited != rightPeek.NextSibling {
							r = rightPeek.NextSibling
						} else {
							match(leftPeek, rightPeek, distTbl, minCostM)
							rightLastVisited, _ = rightS.Pop()
						}
					}
				}
				leftLastVisited, _ = leftS.Pop()
			}
		}
	}
	costBySig := make(map[string]costPairs)
	for pair, cost := range distTbl {
		if _, ok := costBySig[string(pair.Left.Signature)]; !ok {
			costBySig[string(pair.Left.Signature)] = costPairs{costPair{pair, cost}}
		} else {
			costBySig[string(pair.Left.Signature)] = append(costBySig[string(pair.Left.Signature)], costPair{pair, cost})
		}
	}
	for _, costs := range costBySig {
		if len(costs) > 1 {
			sort.Sort(costs)
		}
		for _, cost := range costs {
			minCostM.Add(cost.nodePair)
		}
	}

	return editScript(left, right, minCostM), nil
}

// reduceMatchingSpace removes nodes with the same signature and hash value
// to reduce number of comparisons for the matching step.
// Every matching child is removed except one which is needed prerequisite for more
// accurate matching between subtrees.
func reduceMatchingSpace(left, right *xtree.Node) {
	l := left.FirstChild
	r := right.FirstChild

	var candidates []nodePair
	for l != nil && r != nil {
		leftNext := l.NextSibling
		rightNext := r.NextSibling
		if (l.Type == xtree.Element && r.Type == xtree.Element) &&
			bytesEqual(l.Signature, r.Signature) {
			if bytesEqual(l.Hash, r.Hash) {
				candidates = append(candidates, nodePair{l, r})
			} else {
				reduceMatchingSpace(l, r)
			}
		}
		l = leftNext
		r = rightNext
	}
	for i := 0; i < len(candidates)-1; i++ {
		candidates[i].Left.Remove()
		candidates[i].Right.Remove()
	}
}

func match(l, r *xtree.Node, distTbl distTable, minCostM minCostMatch) {
	if !bytesEqual(l.Signature, r.Signature) {
		return
	}
	pair := nodePair{l, r}
	if bytesEqual(l.Hash, r.Hash) {
		// Nodes match, no cost.
		distTbl.Set(pair, 0)
		minCostM.Add(pair)
		return
	}
	if l.FirstChild == nil && r.FirstChild == nil {
		// Set distance for Update.
		distTbl.Set(pair, 1)
		return
	} else if l.FirstChild == nil {
		// Set distance for inserting all missing children into left.
		distTbl.Set(pair, len(r.Children()))
		return
	} else if r.FirstChild == nil {
		// Set distance for deleting all children from left tree.
		distTbl.Set(pair, len(l.Children()))
		return
	}
	// Group children of the non-leaf nodes by signature.
	leftG := make(map[string][]*xtree.Node)
	rightG := make(map[string][]*xtree.Node)
	leftCount := 0
	rightCount := 0
	for ch := l.FirstChild; ch != nil; ch = ch.NextSibling {
		leftCount++
		leftG[string(ch.Signature)] = append(leftG[string(ch.Signature)], ch)
	}
	for ch := r.FirstChild; ch != nil; ch = ch.NextSibling {
		rightCount++
		rightG[string(ch.Signature)] = append(rightG[string(ch.Signature)], ch)
	}

	var costs costPairs
	dist := 0
	for sig, leftChildren := range leftG {
		if rightChildren, ok := rightG[sig]; ok {
			for _, leftCh := range leftChildren {
				for _, rightCh := range rightChildren {
					pair := nodePair{leftCh, rightCh}
					c := distTbl[pair]
					costs = append(costs, costPair{nodePair: pair, Cost: c})
				}
			}
		}
	}
	if len(costs) > 1 {
		sort.Sort(costs)
	}
	usedLeft := make(map[*xtree.Node]struct{})
	usedRight := make(map[*xtree.Node]struct{})
	mapped := 0
	for _, cost := range costs {
		if _, ok := usedLeft[cost.Left]; ok {
			continue
		}
		if _, ok := usedRight[cost.Right]; ok {
			continue
		}
		dist += cost.Cost
		mapped++
		usedLeft[cost.Left] = struct{}{}
		usedRight[cost.Right] = struct{}{}
	}
	dist += leftCount + rightCount - 2*mapped

	distTbl.Set(pair, dist)
}

// editScript generates slice of deltas that forms minimum-cost edit script to transform
// left xtree into a right xtree.
// TODO change algorithm to the iterative traversal
func editScript(left, right *xtree.Node, minCostM minCostMatch) []Delta {
	var script []Delta
	rootPair := nodePair{left, right}
	_, ok := minCostM[rootPair]
	if !ok {
		return []Delta{
			Delta{Operation: DeleteSubtree, Subject: left, Object: left.Parent},
			Delta{Operation: InsertSubtree, Subject: right, Object: left.Parent},
		}
	}
	for l := left.FirstChild; l != nil; l = l.NextSibling {
		for r := right.FirstChild; r != nil; r = r.NextSibling {
			pair := nodePair{l, r}
			if _, ok := minCostM[pair]; ok {
				if l.FirstChild == nil && r.FirstChild == nil {
					if bytesEqual(l.Hash, r.Hash) {
						continue
					}
					script = append(script, Delta{Operation: Update, Subject: l, Object: r})
					continue
				}
				script = append(script, editScript(l, r, minCostM)...)
			}
		}
		if !minCostM.HasLeft(l) {
			if l.FirstChild == nil {
				script = append(script, Delta{Operation: Delete, Subject: l, Object: l.Parent})
				continue
			}
			script = append(script, Delta{Operation: DeleteSubtree, Subject: l, Object: l.Parent})
		}
	}
	for r := right.FirstChild; r != nil; r = r.NextSibling {
		if !minCostM.HasRight(r) {
			if r.FirstChild == nil {
				script = append(script, Delta{Operation: Insert, Subject: r, Object: r.Parent})
				continue
			}
			script = append(script, Delta{Operation: InsertSubtree, Subject: r, Object: r.Parent})
		}
	}
	return script
}

func contains(n *xtree.Node, nodes []*xtree.Node) bool {
	for _, x := range nodes {
		if x == n {
			return true
		}
	}
	return false
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
