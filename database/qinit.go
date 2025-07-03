// /home/krylon/go/src/github.com/blicero/carebear/database/qinit.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-03 18:51:37 krylon>

package database

var qinit = []string{
	`
CREATE TABLE device (
    id		INTEGER PRIMARY KEY,
    name	TEXT UNIQUE NOT NULL,
    addr        TEXT NOT NULL DEFAULT '[]',
    bighead     INTEGER NOT NULL DEFAULT 1,
    last_seen   INTEGER NOT NULL DEFAULT 0,
    CHECK (json_valid(addr)),
) STRICT
`,
}
