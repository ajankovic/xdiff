package xdiff

import (
	"fmt"
	"io"
)

// TextEncoder knows how to convert edit script to plain text.
type TextEncoder struct {
	w io.Writer
}

// Encode sends edit script in plain text to the stream.
func (pte *TextEncoder) Encode(deltas []Delta) error {
	if len(deltas) == 0 {
		fmt.Fprint(pte.w, "No difference.\n")
	}
	for _, d := range deltas {
		if d.Operation == Update {
			fmt.Fprintf(pte.w, "%s('%s'->'%s')\n", d.Operation, d.Subject, d.Object)
			continue
		}
		fmt.Fprintf(pte.w, "%s('%s')\n", d.Operation, d.Subject)
	}
	return nil
}

// NewTextEncoder creates new text encoder.
func NewTextEncoder(w io.Writer) *TextEncoder {
	return &TextEncoder{w}
}
