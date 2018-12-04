package parser

import (
	"bytes"
	"errors"
	"unicode/utf16"
	"unicode/utf8"
)

// ensureUTF8 will check bytes for byte order marks and return proper UTF8 encoded data
// along with starting position of the content.
func ensureUTF8(b []byte) ([]byte, int) {
	if bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}) {
		// UTF8
		return b, 3
	} else if bytes.HasPrefix(b, []byte{0xFF, 0xFE}) {
		// UTF 16 LE
		var err error
		d, err := decodeUTF16toUTF8(b)
		if err != nil {
			panic(err)
		}
		return d, 3
	} else if bytes.HasPrefix(b, []byte{0xFE, 0xFF}) {
		// UTF 16 BE
		var err error
		d, err := decodeUTF16toUTF8(b)
		if err != nil {
			panic(err)
		}
		return d, 3
	}
	return b, 0
}

// decodeUTF16toUTF8 handles content encoded with UTF16.
func decodeUTF16toUTF8(b []byte) ([]byte, error) {
	if len(b)%2 != 0 {
		return nil, errors.New("must have even length byte slice")
	}
	u16s := make([]uint16, 1)
	ret := &bytes.Buffer{}
	b8buf := make([]byte, 4)
	lb := len(b)
	for i := 0; i < lb; i += 2 {
		u16s[0] = uint16(b[i]) + (uint16(b[i+1]) << 8)
		r := utf16.Decode(u16s)
		n := utf8.EncodeRune(b8buf, r[0])
		ret.Write(b8buf[:n])
	}
	return ret.Bytes(), nil
}
