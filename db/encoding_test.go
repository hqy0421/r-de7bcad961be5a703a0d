package db

import (
	"reflect"
	"testing"
)

func TestEncodingDecode(t *testing.T) {
	test := func(enc textEncoding, b []byte, want string, wantOK bool) {
		t.Helper()
		have, ok := enc.decode(b)
		if have, want := ok, wantOK; have != want {
			t.Errorf("%v %x: ok have %t, want %t", enc, b, have, want)
		}
		if !wantOK {
			return
		}
		if have, want := have, want; have != want {
			t.Errorf("%v %x: have %q, want %q", enc, b, have, want)
		}
	}
	test(encUTF8, []byte("aap"), "aap", true)
	test(encUTF8, []byte("\xe4\xb8\xad"), "中", true)
	test(encUTF8, []byte{0x00}, "\x00", true)

	// "中" is 0x4e2d
	test(encUTF16le, []byte{0x2d, 0x4e}, "中", true)
	test(encUTF16be, []byte{0x4e, 0x2d}, "中", true)

	// U+1F600 is stored as a surrogate pair
	test(encUTF16le, []byte{0x3d, 0xd8, 0x00, 0xde}, "😀", true)
	test(encUTF16be, []byte{0xd8, 0x3d, 0xde, 0x00}, "😀", true)

	// UTF-16 values always have an even number of bytes
	test(encUTF16le, []byte{0x2d}, "", false)
	test(encUTF16be, []byte{0x4e, 0x2d, 0x00}, "", false)
}

func TestEncodingEncode(t *testing.T) {
	test := func(enc textEncoding, s string, want []byte) {
		t.Helper()
		if have, want := enc.encode(s), want; string(have) != string(want) {
			t.Errorf("%v: have %x, want %x", enc, have, want)
		}
	}
	test(encUTF8, "中", []byte("\xe4\xb8\xad"))
	test(encUTF16le, "中", []byte{0x2d, 0x4e})
	test(encUTF16be, "中", []byte{0x4e, 0x2d})
	test(encUTF16le, "😀", []byte{0x3d, 0xd8, 0x00, 0xde})
	test(encUTF16be, "😀", []byte{0xd8, 0x3d, 0xde, 0x00})
}

func TestEncodingCompare(t *testing.T) {
	test := func(enc textEncoding, a, b string, want int) {
		t.Helper()
		if have, want := enc.compare(a, b), want; have != want {
			t.Errorf("%v: %q <=> %q: have %d, want %d", enc, a, b, have, want)
		}
	}
	// UTF-8 compares by code point, same as byte order.
	test(encUTF8, "a", "b", -1)
	test(encUTF8, "中", "䇠", 1) // 4e2d > 41e0

	// In UTF-16le the bytes are little endian, so the byte order (which is
	// what SQLite's binary collation uses) is not the code point order.
	test(encUTF16le, "䇠", "中", 1) // 41e0 > 4e2d by bytes: e0 41 vs 2d 4e
	test(encUTF16le, "é", "中", 1) // e900 > 2d4e
	test(encUTF16le, "中", "中a", -1)
	test(encUTF16le, "😀", "A", -1) // 3dd8... < 4100

	// In UTF-16be the byte order is the code point order.
	test(encUTF16be, "中", "䇠", 1)
	test(encUTF16be, "😀", "é", 1)
}

func TestParseRecordUTF16(t *testing.T) {
	// A record with one text value "中", stored as UTF-16.
	test := func(enc textEncoding, body []byte, want string) {
		t.Helper()
		// header: size 1 + one serial type. The text serial type is
		// 13 + 2*len(body).
		st := byte(13 + 2*len(body))
		rec := append([]byte{2, st}, body...)
		got, err := parseRecord(rec, enc)
		if err != nil {
			t.Fatalf("%v: %s", enc, err)
		}
		if have, want := got, (Record{want}); !reflect.DeepEqual(have, want) {
			t.Errorf("%v: have %#v, want %#v", enc, have, want)
		}
	}
	test(encUTF16le, []byte{0x2d, 0x4e}, "中")
	test(encUTF16be, []byte{0x4e, 0x2d}, "中")
	// non-ascii with a surrogate pair
	test(encUTF16le, []byte{0x3d, 0xd8, 0x00, 0xde}, "😀")

	// odd number of bytes is not valid UTF-16
	if _, err := parseRecord([]byte{2, 15, 0x2d}, encUTF16le); err != ErrCorrupted {
		t.Errorf("have %v, want %v", err, ErrCorrupted)
	}
}

func TestCollateFuncEncoding(t *testing.T) {
	// binary uses the file encoding: "é" (e900) sorts after "䇠" (e041) in LE
	if have, want := collateFunc("binary", encUTF16le)("é", "䇠"), 1; have != want {
		t.Errorf("binary: have %d, want %d", have, want)
	}
	// in BE "中" (4e2d) sorts after "䇠" (41e0)
	if have, want := collateFunc("binary", encUTF16be)("中", "䇠"), 1; have != want {
		t.Errorf("binary: have %d, want %d", have, want)
	}

	// nocase and rtrim work on decoded text, so they give the same result for
	// every encoding: code point order.
	for _, enc := range []textEncoding{encUTF8, encUTF16le, encUTF16be} {
		// nocase folds ascii
		if have, want := collateFunc("nocase", enc)("A", "a"), 0; have != want {
			t.Errorf("nocase %v: have %d, want %d", enc, have, want)
		}
		if have, want := collateFunc("nocase", enc)("中", "中"), 0; have != want {
			t.Errorf("nocase %v: have %d, want %d", enc, have, want)
		}
		// "中" (4e2d) is after "䇠" (41e0) by code point
		if have, want := collateFunc("nocase", enc)("中", "䇠"), 1; have != want {
			t.Errorf("nocase %v: have %d, want %d", enc, have, want)
		}
		// rtrim only trims ascii spaces
		if have, want := collateFunc("rtrim", enc)("a  ", "a"), 0; have != want {
			t.Errorf("rtrim %v: have %d, want %d", enc, have, want)
		}
		if have, want := collateFunc("rtrim", enc)("中", "䇠"), 1; have != want {
			t.Errorf("rtrim %v: have %d, want %d", enc, have, want)
		}
	}

	// an unknown collate falls back to binary in the file encoding
	if have, want := collateFunc("no-such-collate", encUTF16le)("é", "䇠"), 1; have != want {
		t.Errorf("unknown: have %d, want %d", have, want)
	}
}
