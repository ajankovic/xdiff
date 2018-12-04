package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ajankovic/xdiff/xtree"
)

func TestStandardValidFiles(t *testing.T) {
	t.Skip()
	p := NewStandard()
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

func TestStandardInvalidFiles(t *testing.T) {
	t.Skip()
	p := NewStandard()
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

func TestStandard_ParseReader(t *testing.T) {
	p := NewStandard()
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

func TestStandard_ParseFile(t *testing.T) {
	p := NewStandard()
	root, err := p.ParseFile(basicFilename)
	if err != nil {
		t.Fatal(err)
	}
	validateBasicXML(t, root)
}

func TestStandard_ParseDir(t *testing.T) {
	testDir := "testfiles/xmldir/a"
	p := NewStandard()
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
