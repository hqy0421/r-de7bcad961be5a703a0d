#!/bin/bash
set -eu

# A database stored with UTF-16 TEXT values, and one with UTF-16be. Both have
# non-ASCII keys so that byte order differs from code point order.
# SQLite only uses the requested encoding when the file is created in the same
# invocation as the PRAGMA, so everything happens in one call.

for pair in "UTF-16le:utf16le" "UTF-16be:utf16be"; do
    enc=${pair%:*}
    DB=${pair#*:}.sqlite
    rm -f "$DB"
    sqlite3 --batch "$DB" <<HERE
PRAGMA encoding="$enc";
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
CREATE INDEX t_k_nocase ON t (k collate nocase);
CREATE INDEX t_k_rtrim ON t (k collate rtrim);
HERE
done
