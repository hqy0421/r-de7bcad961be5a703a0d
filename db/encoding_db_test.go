package db

import (
	"reflect"
	"testing"
)

// The order SQLite itself gives for these files, per encoding. Note that the
// UTF-16le order differs: its binary collation is a memcmp() over the raw
// little endian bytes, which is not code point order.
var utf16Order = map[string][]string{
	"utf16le.sqlite": {
		"\ue000", "中", "中a", "😀", "A", "a", "zz", "䇠", "é",
	},
	"utf16be.sqlite": {
		"A", "a", "zz", "é", "䇠", "中", "中a", "😀", "\ue000",
	},
}

func TestUTF16Header(t *testing.T) {
	for file, enc := range map[string]textEncoding{
		"utf16le.sqlite": encUTF16le,
		"utf16be.sqlite": encUTF16be,
	} {
		db, err := OpenFile("./../testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		if have, want := db.header.Encoding, enc; have != want {
			t.Errorf("%s: have %v, want %v", file, have, want)
		}
		db.Close()
	}
}

func TestUTF16Scan(t *testing.T) {
	for file := range utf16Order {
		db, err := OpenFile("./../testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		table, err := db.Table("t")
		if err != nil {
			t.Fatal(err)
		}
		// check a non-ascii value
		row, err := table.Rowid(1)
		if err != nil {
			t.Fatal(err)
		}
		if have, want := row, (Record{"中", "中国"}); !reflect.DeepEqual(have, want) {
			t.Errorf("%s: have %#v, want %#v", file, have, want)
		}
		db.Close()
	}
}

func TestUTF16IndexOrder(t *testing.T) {
	for file, want := range utf16Order {
		db, err := OpenFile("./../testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		// The primary key index is a binary collation, which is what we want
		// to check the order of.
		ind, err := db.Index("sqlite_autoindex_t_1")
		if err != nil {
			t.Fatal(err)
		}
		var keys []string
		if err := ind.Scan(func(rec Record) bool {
			keys = append(keys, rec[0].(string))
			return false
		}); err != nil {
			t.Fatal(err)
		}
		if have, want := keys, want; !reflect.DeepEqual(have, want) {
			t.Errorf("%s:\nhave %#v\nwant %#v", file, have, want)
		}
		db.Close()
	}
}

func TestUTF16IndexEq(t *testing.T) {
	for file := range utf16Order {
		db, err := OpenFile("./../testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		ind, err := db.Index("sqlite_autoindex_t_1")
		if err != nil {
			t.Fatal(err)
		}
		var got []Record
		if err := ind.ScanEq(
			Key{{V: "中"}},
			func(rec Record) bool {
				got = append(got, rec)
				return false
			},
		); err != nil {
			t.Fatal(err)
		}
		if have, want := got, ([]Record{{"中", int64(1)}}); !reflect.DeepEqual(have, want) {
			t.Errorf("%s:\nhave %#v\nwant %#v", file, have, want)
		}
		db.Close()
	}
}

func TestUTF16CollateIndexEq(t *testing.T) {
	// the nocase index finds both "A" and "a"
	for file := range utf16Order {
		db, err := OpenFile("./../testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		ind, err := db.Index("t_k_nocase")
		if err != nil {
			t.Fatal(err)
		}
		var keys []string
		if err := ind.ScanEq(
			Key{{V: "A", Collate: "nocase"}},
			func(rec Record) bool {
				keys = append(keys, rec[0].(string))
				return false
			},
		); err != nil {
			t.Fatal(err)
		}
		if have, want := keys, []string{"a", "A"}; !reflect.DeepEqual(have, want) {
			t.Errorf("%s: have %#v, want %#v", file, have, want)
		}
		db.Close()
	}
}

func TestUTF16IndexRange(t *testing.T) {
	// ScanRange uses the index key comparison; it must follow the file
	// encoding. For utf16le "中" <= "😀" (bytes 2d4e.. < 3dd8..), while for
	// utf16be "😀" > "中".
	for file := range utf16Order {
		db, err := OpenFile("./../testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		ind, err := db.Index("sqlite_autoindex_t_1")
		if err != nil {
			t.Fatal(err)
		}
		var keys []string
		if err := ind.ScanRange(
			Key{{V: "中"}},
			Key{{V: "😀"}},
			func(rec Record) bool {
				keys = append(keys, rec[0].(string))
				return false
			},
		); err != nil {
			t.Fatal(err)
		}
		var want []string
		// Both encodings happen to give the same result here. Go's string
		// comparison (UTF-8 byte order) is not used to compute the
		// expectation, since it does not match utf16le.
		switch file {
		case "utf16le.sqlite":
			want = []string{"中", "中a"}
		case "utf16be.sqlite":
			want = []string{"中", "中a"}
		}
		if have, want := keys, want; !reflect.DeepEqual(have, want) {
			t.Errorf("%s:\nhave %#v\nwant %#v", file, have, want)
		}
		db.Close()
	}
}
