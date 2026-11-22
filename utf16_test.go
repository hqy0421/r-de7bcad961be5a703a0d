package sqlittle

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The U+E000 character is written as an escape, it is invisible.
var utf16Expected = map[string]struct {
	scan    []string // rowid order
	indexed []string // index t_v order, i.e. the file's binary order of v
}{
	"testdata/utf16le.sqlite": {
		scan: []string{
			"中", "a", "A", "é", "中a", "😀", "zz", "", "䇠",
		},
		// utf16le compares little endian bytes: e000 < 2d4e < 3dd8 < 4100 ..
		indexed: []string{"中", "A", "a", "䇠", "é", "😀", "", "中a", "zz"},
	},
	"testdata/utf16be.sqlite": {
		scan: []string{
			"中", "a", "A", "é", "中a", "😀", "zz", "", "䇠",
		},
		// utf16be compares big endian bytes, which is code point order
		indexed: []string{"A", "a", "䇠", "é", "😀", "", "中a", "zz", "中"},
	},
}

func TestUTF16Select(t *testing.T) {
	for file, want := range utf16Expected {
		db, err := Open(file)
		require.NoError(t, err)

		var got []string
		require.NoError(t, db.Select("t", func(r Row) {
			s, _ := r.ScanString()
			got = append(got, s)
		}, "k"))
		require.Equal(t, want.scan, got, file)

		got = nil
		require.NoError(t, db.IndexedSelect("t", "t_v", func(r Row) {
			s, _ := r.ScanString()
			got = append(got, s)
		}, "k"))
		require.Equal(t, want.indexed, got, file)

		require.NoError(t, db.Close())
	}
}

func TestUTF16PKSelect(t *testing.T) {
	for file := range utf16Expected {
		db, err := Open(file)
		require.NoError(t, err)

		var got []string
		require.NoError(t, db.PKSelect("t", Key{"中"}, func(r Row) {
			k, v, _ := r.ScanStringString()
			got = append(got, k+":"+v)
		}, "k", "v"))
		require.Equal(t, []string{"中:中国"}, got, file)

		require.NoError(t, db.Close())
	}
}

func TestUTF16IndexedSelectEq(t *testing.T) {
	for file := range utf16Expected {
		db, err := Open(file)
		require.NoError(t, err)

		var got []string
		require.NoError(t, db.IndexedSelectEq("t", "t_v", Key{"中国"}, func(r Row) {
			s, _ := r.ScanString()
			got = append(got, s)
		}, "k"))
		require.Equal(t, []string{"中"}, got, file)

		require.NoError(t, db.Close())
	}
}

// TestUTF16SwitchDatabases checks that databases with different encodings can
// be opened after each other without mixing up the encoding: every database
// keeps using the encoding from its own header.
func TestUTF16SwitchDatabases(t *testing.T) {
	order := []string{
		"testdata/utf16le.sqlite",
		"testdata/utf16be.sqlite",
		"testdata/utf8.sqlite",
		"testdata/utf16be.sqlite",
		"testdata/utf16le.sqlite",
	}
	want := map[string][]string{
		"testdata/utf16le.sqlite": {"中", "A", "a", "䇠", "é", "😀", "", "中a", "zz"},
		"testdata/utf16be.sqlite": {"A", "a", "䇠", "é", "😀", "", "中a", "zz", "中"},
		"testdata/utf8.sqlite":    {"A", "a", "䇠", "é", "😀", "", "中a", "zz", "中"},
	}
	for _, file := range order {
		db, err := Open(file)
		require.NoError(t, err)

		var got []string
		require.NoError(t, db.IndexedSelect("t", "t_v", func(r Row) {
			s, _ := r.ScanString()
			got = append(got, s)
		}, "k"))
		require.Equal(t, want[file], got, file)
		require.NoError(t, db.Close())
	}
}
