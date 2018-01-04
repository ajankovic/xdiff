package xdiff

import (
	"encoding/xml"
	"strings"
	"testing"
)

var (
	originalDoc = `<?xml version="1.0" encoding="UTF-8"?>
<ConnectedApp xmlns="http://soap.sforce.com/2006/04/metadata">
	<contactEmail>foo@example.org</contactEmail>
	<label>WooCommerce</label>
	<oauthConfig>
		<callbackUrl>https://login.salesforce.com/services/oauth2/callback</callbackUrl>
		<consumerKey required="true">CLIENTID</consumerKey>
		<scopes>Basic</scopes>
		<scopes>Api</scopes>
		<scopes>Web</scopes>
		<scopes>Full</scopes>
	</oauthConfig>
</ConnectedApp>
`
	editedDoc = `<?xml version="1.0" encoding="UTF-8"?>
<ConnectedApp xmlns="http://soap.sforce.com/2006/04/metadata">
    <contactEmail>foo@example.org</contactEmail>
    <label>WooCommerce</label>
    <oauthConfig>
        <callbackUrl>https://login.salesforce.com/services/oauth2/callback</callbackUrl>
		<consumerKey>OTHER</consumerKey>
		<!--Comment-->
        <scopes>Full</scopes>
        <scopes>Basic</scopes>
        <configurable>
            <empty></empty>
        </configurable>
    </oauthConfig>
</ConnectedApp>
`
)

func TestParseDoc(t *testing.T) {
	tree, err := ParseDoc(strings.NewReader(originalDoc))
	if err != nil {
		t.Fatal(err)
	}
	if !tree.Root.IsRoot() {
		t.Error("Not root.")
	}
	leafs := 0
	walk(tree.Root, 0, func(n *Node, l int) {
		if n.LastChild == nil {
			leafs++
		}
	})
	if leafs != 10 {
		t.Errorf("Incorrect number of leafs, got %d.", leafs)
	}
	content := string(tree.Root.LastChild.LastChild.PrevSibling.LastChild.Content)
	if content != "WooCommerce" {
		t.Errorf("Third leaf incorrect, got %s.", content)
	}
	attr := tree.Root.LastChild.LastChild.PrevSibling.PrevSibling.PrevSibling
	if attr.Name != "xmlns" {
		t.Errorf("ConnectedApp xmlns attribute has incorrect name %s", attr.Name)
	}
	if string(attr.Content) != "http://soap.sforce.com/2006/04/metadata" {
		t.Errorf("ConnectedApp xmlns attribute has incorrect content %q", attr.Content)
	}
	if attr.Signature != "//ConnectedApp/xmlns/attr" {
		t.Errorf("ConnectedApp xmlns attribute has incorrect signature %s", attr.Signature)
	}
}

func TestCompare(t *testing.T) {
	deltas, err := Compare(
		strings.NewReader(originalDoc),
		strings.NewReader(editedDoc),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(deltas) != 6 {
		t.Errorf("Incorrect number of deltas, got %d.", len(deltas))
	}
	opCount := make(map[Operation]int)
	for _, d := range deltas {
		if c, ok := opCount[d.Op]; ok {
			opCount[d.Op] = c + 1
		} else {
			opCount[d.Op] = 1
		}
	}
	if opCount[InsertSubtree] != 1 {
		t.Errorf("Incorrect number of InsertSubtree deltas, got %d.", opCount[InsertSubtree])
	}
}

func BenchmarkParseDocOneElement(b *testing.B) {
	for i := 0; i < b.N; i++ {
		doc, err := ParseDoc(strings.NewReader("<xml>innertext</xml>"))
		if err != nil {
			b.Fatal(err)
		}
		_ = doc
	}
}

func BenchmarkParseDocTwoElements(b *testing.B) {
	for i := 0; i < b.N; i++ {
		doc, err := ParseDoc(strings.NewReader("<xml><xml>innertext</xml></xml>"))
		if err != nil {
			b.Fatal(err)
		}
		_ = doc
	}
}

func BenchmarkParseDocFourElements(b *testing.B) {
	for i := 0; i < b.N; i++ {
		doc, err := ParseDoc(strings.NewReader("<xml><xml><xml><xml>innertext</xml></xml></xml></xml>"))
		if err != nil {
			b.Fatal(err)
		}
		_ = doc
	}
}

func BenchmarkParseDocOriginal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		doc, err := ParseDoc(strings.NewReader(originalDoc))
		if err != nil {
			b.Fatal(err)
		}
		_ = doc
	}
}

func BenchmarkPlainXmlParse(b *testing.B) {
	type Node struct {
		XMLName xml.Name
		Attrs   []xml.Attr `xml:"-"`
		Content []byte     `xml:",innerxml"`
		Nodes   []Node     `xml:",any"`
	}
	var node Node
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := xml.Unmarshal([]byte(originalDoc), &node)
		if err != nil {
			b.Fatal(err)
		}
	}
	_ = node
}

func BenchmarkCompare(b *testing.B) {
	for i := 0; i < b.N; i++ {
		deltas, err := Compare(
			strings.NewReader(originalDoc),
			strings.NewReader(editedDoc),
		)
		if err != nil {
			b.Fatal(err)
		}
		_ = deltas
	}
}
