package xdiff

import (
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
		<consumerKey>CLIENTID</consumerKey>
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
        <scopes>Full</scopes>
        <scopes>Basic</scopes>
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
	if len(tree.Leafs) != 22 {
		t.Errorf("Incorrect number of leafs, got %d.", len(tree.Leafs))
	}
	for i, leaf := range tree.Leafs {
		if i == 5 {
			if string(leaf.Content) != "WooCommerce" {
				t.Errorf("Sixth leaf incorrect, got %s.", leaf.Content)
			}
		}
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
	if len(deltas) != 3 {
		t.Errorf("Incorrect number of deltas, got %d.", len(deltas))
	}
}
