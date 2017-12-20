package xdiff

import (
	"fmt"
	"io"
)

// Encoder knows how to interpret diff results.
type Encoder interface {
	Encode([]Delta) error
}

type plainTextEncoder struct {
	w io.Writer
}

func (pte *plainTextEncoder) Encode(deltas []Delta) error {
	for _, d := range deltas {
		fmt.Fprintf(pte.w, "%s('%s')\n", d.Op, d.Desc)
	}
	return nil
}

// PlainTextEncoder outputs diff results in plain text format.
func PlainTextEncoder(w io.Writer) Encoder {
	return &plainTextEncoder{w}
}
