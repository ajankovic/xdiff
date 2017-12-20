package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ajankovic/xdiff"
)

func main() {
	var (
		originalFname string
		editedFname   string
	)
	flag.StringVar(&originalFname, "original", "original.xml", "Original xml document.")
	flag.StringVar(&editedFname, "edited", "edited.xml", "Edited xml document.")
	flag.Parse()
	if originalFname == "" || editedFname == "" {
		fail("File names of both original and edited documents are required.")
	}
	oFile, err := os.Open(originalFname)
	if err != nil {
		fail("Failed to open original file %s error: %v",
			originalFname, err.Error())
	}
	eFile, err := os.Open(editedFname)
	if err != nil {
		fail("Failed to open edited file %s error: %v",
			originalFname, err.Error())
	}
	diff, err := xdiff.Compare(oFile, eFile)
	if err != nil {
		fail("Failed to compare files error: %v", err.Error())
	}
	enc := xdiff.PlainTextEncoder(os.Stdout)
	if err := enc.Encode(diff); err != nil {
		fail("Failed to generate output error: %v", err.Error())
	}
}

func fail(msg string, params ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", params...)
	os.Exit(1)
}
