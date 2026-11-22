package db

import (
	"bytes"
	"strings"
	"unicode/utf16"
)

// textEncoding is the text encoding of a database file, as stored in the
// database header ("1.2. The Database Header", offset 56).
//
// SQLite stores all TEXT values in the encoding of the database file. The
// encoding is fixed for the lifetime of a file, and every record, index key,
// and schema definition has to be decoded with it. It is also the encoding
// SQLite's built-in "binary" collating function uses for its memcmp(), so
// index searches have to compare values using these exact bytes.
type textEncoding int

const (
	encUTF8 textEncoding = iota
	encUTF16le
	encUTF16be
)

// decode turns the bytes of a TEXT value, as stored in the database, into a Go
// string (UTF-8). ok is false when b is not valid in the encoding (a UTF-16
// value always consists of an even number of bytes).
func (e textEncoding) decode(b []byte) (string, bool) {
	if e == encUTF8 {
		return string(b), true
	}
	if len(b)%2 != 0 {
		return "", false
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		if e == encUTF16be {
			u[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
		} else {
			u[i] = uint16(b[2*i]) | uint16(b[2*i+1])<<8
		}
	}
	return string(utf16.Decode(u)), true
}

// encode turns a Go string back into the bytes SQLite would store for a TEXT
// value in this encoding.
func (e textEncoding) encode(s string) []byte {
	if e == encUTF8 {
		return []byte(s)
	}
	u := utf16.Encode([]rune(s))
	b := make([]byte, 0, len(u)*2)
	for _, c := range u {
		if e == encUTF16be {
			b = append(b, byte(c>>8), byte(c))
		} else {
			b = append(b, byte(c), byte(c>>8))
		}
	}
	return b
}

// compare compares two text values the way SQLite's "binary" collating
// function does: with a memcmp() over the bytes in the database encoding.
// This is *not* the same as comparing the decoded strings for UTF-16: in
// UTF-16le the low byte of a code unit comes first, so the byte order differs
// from the code point order.
func (e textEncoding) compare(a, b string) int {
	if e == encUTF8 {
		return strings.Compare(a, b)
	}
	return bytes.Compare(e.encode(a), e.encode(b))
}
