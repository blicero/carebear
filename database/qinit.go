// /home/krylon/go/src/github.com/blicero/carebear/database/qinit.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-07 15:43:04 krylon>

package database

// This files contains the SQL queries to initialize a fresh database.
// Having that defined inside the application is both convenient for reference
// and for testing.

var qinit = []string{
	`
CREATE TABLE device (
    id		INTEGER PRIMARY KEY,
    name	TEXT UNIQUE NOT NULL,
    addr        TEXT NOT NULL DEFAULT '[]',
    bighead     INTEGER NOT NULL DEFAULT 1,
    last_seen   INTEGER NOT NULL DEFAULT 0,
    CHECK (json_valid(addr))
) STRICT
`,
	"CREATE INDEX dev_big_idx ON device (bighead <> 0)",
	"CREATE INDEX dev_last_idx ON device (last_seen)",
}
