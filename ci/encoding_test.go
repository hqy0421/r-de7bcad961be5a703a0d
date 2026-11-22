// +build ci

package ci

import (
	"testing"

	"github.com/alicebob/sqlittle"
)

// UTF-16 databases must use the file's own encoding for both decoding values
// and comparing index keys. These compare against sqlite3 itself.

func TestUTF16Select(t *testing.T) {
	for _, enc := range []string{"UTF-8", "UTF-16le", "UTF-16be"} {
		Compare(
			t,
			`
PRAGMA encoding="`+enc+`";
CREATE TABLE t (k text primary key, v text);
INSERT INTO t VALUES
    ('中', '中国'),
    ('a', 'apple'),
    ('A', 'Apple'),
    ('é', 'eacute'),
    ('中a', 'zhong-a'),
    ('😀', 'emoji'),
    ('zz', 'zz'),
    (char(0xE000), 'private'),
    (char(0x41E0), 'cjk-e000');
`,
			`SELECT k, v FROM t`,
			func(t *testing.T, db *sqlittle.DB) [][]string {
				var rows [][]string
				err := db.Select("t", func(r sqlittle.Row) {
					rows = append(rows, r.ScanStrings())
				}, "k", "v")
				if err != nil {
					t.Fatal(err)
				}
				return rows
			},
		)
	}
}

func TestUTF16IndexOrder(t *testing.T) {
	for _, enc := range []string{"UTF-8", "UTF-16le", "UTF-16be"} {
		Compare(
			t,
			`
PRAGMA encoding="`+enc+`";
CREATE TABLE t (k text primary key, v text);
INSERT INTO t VALUES
    ('中', '中国'),
    ('a', 'apple'),
    ('A', 'Apple'),
    ('é', 'eacute'),
    ('中a', 'zhong-a'),
    ('😀', 'emoji'),
    ('zz', 'zz'),
    (char(0xE000), 'private'),
    (char(0x41E0), 'cjk-e000');
CREATE INDEX t_k ON t (k);
`,
			`SELECT k FROM t ORDER BY k`,
			func(t *testing.T, db *sqlittle.DB) [][]string {
				var rows [][]string
				err := db.IndexedSelect("t", "t_k", func(r sqlittle.Row) {
					s, _ := r.ScanString()
					rows = append(rows, []string{s})
				}, "k")
				if err != nil {
					t.Fatal(err)
				}
				return rows
			},
		)
	}
}

func TestUTF16IndexEq(t *testing.T) {
	for _, enc := range []string{"UTF-8", "UTF-16le", "UTF-16be"} {
		Compare(
			t,
			`
PRAGMA encoding="`+enc+`";
CREATE TABLE t (k text primary key, v text);
INSERT INTO t VALUES
    ('中', '中国'),
    ('a', 'apple'),
    ('A', 'Apple'),
    ('é', 'eacute'),
    ('中a', 'zhong-a'),
    ('😀', 'emoji'),
    ('zz', 'zz'),
    (char(0xE000), 'private'),
    (char(0x41E0), 'cjk-e000');
CREATE INDEX t_v ON t (v);
`,
			`SELECT k FROM t WHERE v='中国'`,
			func(t *testing.T, db *sqlittle.DB) [][]string {
				var rows [][]string
				err := db.IndexedSelectEq("t", "t_v", sqlittle.Key{"中国"}, func(r sqlittle.Row) {
					s, _ := r.ScanString()
					rows = append(rows, []string{s})
				}, "k")
				if err != nil {
					t.Fatal(err)
				}
				return rows
			},
		)
	}
}

func TestUTF16PKSelect(t *testing.T) {
	for _, enc := range []string{"UTF-8", "UTF-16le", "UTF-16be"} {
		Compare(
			t,
			`
PRAGMA encoding="`+enc+`";
CREATE TABLE t (k text primary key, v text);
INSERT INTO t VALUES
    ('中', '中国'),
    ('a', 'apple'),
    ('A', 'Apple'),
    ('é', 'eacute'),
    ('中a', 'zhong-a'),
    ('😀', 'emoji'),
    ('zz', 'zz'),
    (char(0xE000), 'private'),
    (char(0x41E0), 'cjk-e000');
`,
			`SELECT k, v FROM t WHERE k='中'`,
			func(t *testing.T, db *sqlittle.DB) [][]string {
				var rows [][]string
				err := db.PKSelect("t", sqlittle.Key{"中"}, func(r sqlittle.Row) {
					rows = append(rows, r.ScanStrings())
				}, "k", "v")
				if err != nil {
					t.Fatal(err)
				}
				return rows
			},
		)
	}
}

func TestUTF16NonRowid(t *testing.T) {
	for _, enc := range []string{"UTF-8", "UTF-16le", "UTF-16be"} {
		Compare(
			t,
			`
PRAGMA encoding="`+enc+`";
CREATE TABLE t (k text primary key, v text) WITHOUT ROWID;
INSERT INTO t VALUES
    ('中', '中国'),
    ('a', 'apple'),
    ('A', 'Apple'),
    ('é', 'eacute'),
    ('中a', 'zhong-a'),
    ('😀', 'emoji'),
    ('zz', 'zz'),
    (char(0xE000), 'private'),
    (char(0x41E0), 'cjk-e000');
CREATE INDEX t_v ON t (v);
`,
			`SELECT k FROM t ORDER BY k`,
			func(t *testing.T, db *sqlittle.DB) [][]string {
				var rows [][]string
				err := db.Select("t", func(r sqlittle.Row) {
					s, _ := r.ScanString()
					rows = append(rows, []string{s})
				}, "k")
				if err != nil {
					t.Fatal(err)
				}
				return rows
			},
		)
	}
}

// BINARY compares the raw file-encoding bytes, but NOCASE and RTRIM compare
// the decoded text (code point order). So for UTF-16le all three collations
// give a different order, and lookups must follow the index's own collation.
func TestUTF16Collations(t *testing.T) {
	for _, enc := range []string{"UTF-8", "UTF-16le", "UTF-16be"} {
		for _, coll := range []string{"BINARY", "NOCASE", "RTRIM"} {
			Compare(
				t,
				`
PRAGMA encoding="`+enc+`";
CREATE TABLE r (a text);
INSERT INTO r VALUES ('中'), ('é'), ('䇠'), ('a'), ('A'), ('a ');
CREATE INDEX r_idx ON r (a COLLATE `+coll+`);
`,
				`SELECT a FROM r ORDER BY a COLLATE `+coll,
				func(t *testing.T, db *sqlittle.DB) [][]string {
					var rows [][]string
					err := db.IndexedSelect("r", "r_idx", func(r sqlittle.Row) {
						s, _ := r.ScanString()
						rows = append(rows, []string{s})
					}, "a")
					if err != nil {
						t.Fatal(err)
					}
					return rows
				},
			)
		}
	}
}

func TestUTF16CollationEq(t *testing.T) {
	for _, enc := range []string{"UTF-8", "UTF-16le", "UTF-16be"} {
		for _, coll := range []string{"BINARY", "NOCASE", "RTRIM"} {
			Compare(
				t,
				`
PRAGMA encoding="`+enc+`";
CREATE TABLE r (a text, b);
INSERT INTO r VALUES ('中', 1), ('é', 2), ('䇠', 3), ('a', 4), ('A', 5), ('a ', 6);
CREATE INDEX r_idx ON r (a COLLATE `+coll+`);
`,
				`SELECT b FROM r WHERE a='a' COLLATE `+coll,
				func(t *testing.T, db *sqlittle.DB) [][]string {
					var rows [][]string
					// The collation comes from the index definition.
					err := db.IndexedSelectEq("r", "r_idx", sqlittle.Key{"a"}, func(r sqlittle.Row) {
						s, _ := r.ScanString()
						rows = append(rows, []string{s})
					}, "b")
					if err != nil {
						t.Fatal(err)
					}
					return rows
				},
			)
		}
	}
}
