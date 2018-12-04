package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ajankovic/xdiff/xtree"
)

func TestParserValidFiles(t *testing.T) {
	t.Skip()
	p := New()
	testdirs := []string{
		"testfiles/xmlconf/ibm/valid/*/*.xml",
		"testfiles/xmlconf/japanese/*.xml",
		"testfiles/xmlconf/oasis/*/*.xml",
		"testfiles/xmlconf/sun/valid/sa/*.xml",
		"testfiles/xmlconf/xmltest/valid/*.xml",
	}
	exclude := []string{
		"weekly-iso-2022-jp.xml", // not supported encoding
		"pr-xml-iso-2022-jp.xml", // not supported encoding
		"pr-xml-utf-16.xml",      // TODO needs more work
		"weekly-utf-16.xml",      // TODO needs more work
	}
	for _, pat := range testdirs {
		fs, err := filepath.Glob(pat)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range fs {
			if inSlice(f, exclude) {
				continue
			}
			_, err := p.ParseFile(f)
			if err != nil {
				t.Errorf("Expected valid file %s got error\n%v", f, err)
			}
		}
	}
}

func TestParserInvalidFiles(t *testing.T) {
	t.Skip()
	p := New()
	testdirs := []string{
		"testfiles/xmlconf/xmltest/not-wf/sa/*.xml",
	}
	exclude := []string{}
	for _, pat := range testdirs {
		fs, err := filepath.Glob(pat)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range fs {
			if inSlice(f, exclude) {
				continue
			}
			_, err := p.ParseFile(f)
			if err == nil {
				t.Errorf("Expected error with file %s", f)
			}
		}
	}
}

func inSlice(needle string, sl []string) bool {
	for _, str := range sl {
		if strings.Contains(needle, str) {
			return true
		}
	}
	return false
}

var basicFilename = "testfiles/basic.xml"

func TestXDiff_ParseReader(t *testing.T) {
	p := New()
	f, err := os.Open(basicFilename)
	if err != nil {
		t.Fatal(err)
	}
	root, err := p.ParseReader(f)
	if err != nil {
		t.Fatal(err)
	}
	validateBasicXML(t, root)
}

func TestXDiff_ParseFile(t *testing.T) {
	p := New()
	root, err := p.ParseFile(basicFilename)
	if err != nil {
		t.Fatal(err)
	}
	validateBasicXML(t, root)
}

func validateBasicXML(t *testing.T, n *xtree.Node) {
	txt, _ := xtree.TextString(n)
	if len(n.Children()) != 2 {
		t.Logf("\n%s", txt)
		t.Errorf("Expected two root nodes got %d", len(n.Children()))
	}
	if len(n.FirstChild.Children()) != 2 {
		t.Logf("\n%s", txt)
		t.Errorf("Expected declaration to have two nodes got %d", len(n.FirstChild.Children()))
	}
	if len(n.FirstChild.NextSibling.Children()) != 4 {
		t.Logf("\n%s", txt)
		t.Errorf("Expected root element to have four nodes got %d", len(n.FirstChild.NextSibling.Children()))
	}
	if string(n.FirstChild.FirstChild.Signature) != "/xml/version/Attribute" {
		t.Logf("\n%s", txt)
		t.Errorf("Expected signature to be '/xml/version/Attribute', got '%s'", n.FirstChild.FirstChild.Signature)
	}
}

func TestXDiff_ParseDir(t *testing.T) {
	testDir := "testfiles/xmldir/a"
	p := New()
	n, err := p.ParseDir(testDir)
	if err != nil {
		t.Fatal(err)
	}
	xtree.Prepare(n)
	txt, _ := xtree.TextString(n)
	if n.Type != xtree.Directory {
		t.Logf("\n%s", txt)
		t.Errorf("Expected root dir node got %s", n)
	}
	if len(n.Children()) != 3 {
		t.Logf("\n%s", txt)
		t.Errorf("Expected root dir to have three nodes got %d", len(n.Children()))
	}
	if string(n.FirstChild.Name) != "b" {
		t.Logf("\n%s", txt)
		t.Errorf("Expected first child of the root to be named 'b' got %s", n.FirstChild.Name)
	}
}
