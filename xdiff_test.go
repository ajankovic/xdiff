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
	if len(tree.Leafs) != 9 {
		t.Errorf("Incorrect number of leafs, got %d.", len(tree.Leafs))
	}
	for i, leaf := range tree.Leafs {
		if i == 2 {
			if string(leaf.Content) != "WooCommerce" {
				t.Errorf("Third leaf incorrect, got %s.", leaf.Content)
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
	if len(deltas) != 5 {
		t.Errorf("Incorrect number of deltas, got %d.", len(deltas))
	}
}
