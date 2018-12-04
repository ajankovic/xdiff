package xdiff

import (
	"testing"

	"github.com/ajankovic/xdiff/xtree"
)

func TestCompare(t *testing.T) {
	type args struct {
		left  *xtree.Node
		right *xtree.Node
	}
	tests := []struct {
		name    string
		args    args
		want    []Delta
		wantErr bool
	}{
		// TODO add more edge cases
		{
			"Empty document with no difference",
			args{
				left:  doc(""),
				right: doc(""),
			},
			nil,
			false,
		},
		{
			"Single element with no difference",
			args{
				left: doc("",
					el("root")),
				right: doc("",
					el("root")),
			},
			nil,
			false,
		},
		{
			"Insert root element",
			args{
				left: doc(""),
				right: doc("",
					el("root")),
			},
			[]Delta{
				{
					Operation: Insert,
				},
			},
			false,
		},
		{
			"Delete root element",
			args{
				left: doc("",
					el("root")),
				right: doc(""),
			},
			[]Delta{
				{
					Operation: Delete,
				},
			},
			false,
		},
		{
			"Update root element data",
			args{
				left: doc("",
					el("root",
						dat("value"))),
				right: doc("",
					el("root",
						dat("edited"))),
			},
			[]Delta{
				{
					Operation: Update,
				},
			},
			false,
		},
		{
			"Replace subtree",
			args{
				left: doc("",
					el("root",
						dat("value"))),
				right: doc("",
					el("replace",
						dat("edited"))),
			},
			[]Delta{
				{
					Operation: DeleteSubtree,
				},
				{
					Operation: InsertSubtree,
				},
			},
			false,
		},
		{
			"Trees of different height",
			args{
				left: doc("",
					el("outer")),
				right: doc("",
					el("outer",
						el("inner",
							dat("value")))),
			},
			[]Delta{
				{
					Operation: InsertSubtree,
				},
			},
			false,
		},
		{
			"Matching parents by children",
			args{
				left: doc("",
					el("root",
						el("element",
							attr("id", "1"),
							el("subelement",
								dat("value1"))),
						el("element",
							attr("id", "2"),
							el("subelement",
								dat("value2"))),
						el("element",
							attr("id", "3"),
							el("subelement",
								dat("value3"))))),

				right: doc("",
					el("root",
						el("element",
							attr("id", "1"),
							attr("name", "John"),
							el("subelement",
								dat("value1"))),
						el("element",
							el("subelement",
								dat("value2"))),
						el("element",
							attr("id", "4"),
							el("subelement",
								dat("value3"))))),
			},
			[]Delta{
				{
					Operation: Insert,
				},
				{
					Operation: Delete,
				},
				{
					Operation: Update,
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xtree.Prepare(tt.args.left)
			xtree.Prepare(tt.args.right)
			l, _ := xtree.TextString(tt.args.left)
			r, _ := xtree.TextString(tt.args.right)
			got, err := Compare(tt.args.left, tt.args.right)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error =\n%v, wantErr\n%v", err, tt.wantErr)
				return
			}
			if len(tt.want) != len(got) {
				t.Logf("\n%s", l)
				t.Logf("\n%s", r)
				t.Fatalf("Compare() =\n%v, want\n%v", got, tt.want)
			}
			for i := range tt.want {
				if !operationIn(tt.want[i].Operation, got) {
					t.Logf("\n%s", l)
					t.Logf("\n%s", r)
					t.Fatalf("Compare() =\n%v, want\n%v", got, tt.want)
				}
			}
		})
	}
}

func operationIn(op Operation, ed []Delta) bool {
	for _, d := range ed {
		if op == d.Operation {
			return true
		}
	}
	return false
}

func TestReducingMatchingSpace(t *testing.T) {
	left := doc("",
		el("root",
			el("element",
				attr("id", "1"),
				el("subelement",
					dat("value1"))),
			el("element",
				attr("id", "2"),
				el("subelement",
					dat("value2"))),
			el("element",
				attr("id", "3"),
				el("subelement",
					dat("value3")))))
	right := doc("",
		el("root",
			el("element",
				attr("id", "1"),
				el("subelement",
					dat("value1"))),
			el("element",
				attr("id", "2"),
				el("subelement",
					dat("value3"))),
			el("element",
				attr("id", "3"),
				el("subelement",
					dat("value3")))))
	xtree.Prepare(left)
	xtree.Prepare(right)
	reduceMatchingSpace(left, right)
	l, _ := xtree.TextString(left)
	r, _ := xtree.TextString(right)

	if len(left.FirstChild.Children()) != 2 {
		t.Logf("\n%s", l)
		t.Logf("\n%s", r)
		t.Error("root doesn't have two child nodes")
	}
	if len(left.FirstChild.FirstChild.Children()) != 2 {
		t.Logf("\n%s", l)
		t.Logf("\n%s", r)
		t.Error("element doesn't have two child nodes")
	}
	if len(left.FirstChild.FirstChild.FirstChild.NextSibling.Children()) != 1 {
		t.Logf("\n%s", l)
		t.Logf("\n%s", r)
		t.Error("subelement doesn't have one child node")
	}
}

func addChildren(n *xtree.Node, children []*xtree.Node) *xtree.Node {
	for _, ch := range children {
		n.AppendChild(ch)
	}
	return n
}

func dir(name string, children ...*xtree.Node) *xtree.Node {
	return addChildren(xtree.NewDirectory([]byte(name)), children)
}

func doc(name string, children ...*xtree.Node) *xtree.Node {
	return addChildren(xtree.NewDocument([]byte(name)), children)
}

func el(name string, children ...*xtree.Node) *xtree.Node {
	return addChildren(xtree.NewElement([]byte(name)), children)
}

func attr(name, value string) *xtree.Node {
	return xtree.NewAttribute([]byte(name), []byte(value))
}

func doct(value string) *xtree.Node {
	return xtree.NewDoctype([]byte(value))
}

func dat(value string) *xtree.Node {
	return xtree.NewData([]byte(value))
}

func cdat(value string) *xtree.Node {
	return xtree.NewCData([]byte(value))
}

func comm(value string) *xtree.Node {
	return xtree.NewComment([]byte(value))
}

func decl(children ...*xtree.Node) *xtree.Node {
	return addChildren(xtree.NewDeclaration(), children)
}

func proc(name, value string) *xtree.Node {
	return xtree.NewProcInstr([]byte(name), []byte(value))
}
