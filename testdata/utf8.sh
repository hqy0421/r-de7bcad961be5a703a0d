#!/bin/bash
set -eu

# UTF-8 (the default) version of the table in utf16.sh, used to check that
# databases with different encodings can be used after each other.
DB=utf8.sqlite
rm -f "$DB"
sqlite3 --batch "$DB" <<HERE
PRAGMA encoding="UTF-8";
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
