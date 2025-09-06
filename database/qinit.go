// /home/krylon/go/src/github.com/blicero/carebear/database/qinit.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-09-05 20:01:48 krylon>

package database

// This files contains the SQL queries to initialize a fresh database.
// Having that defined inside the application is both convenient for reference
// and for testing.

var qinit = []string{
	`
CREATE TABLE network (
    id		INTEGER PRIMARY KEY,
    addr	TEXT UNIQUE NOT NULL,
    desc	TEXT NOT NULL DEFAULT '',
    last_scan	INTEGER NOT NULL DEFAULT 0
) STRICT
`,
	`
CREATE TABLE device (
    id		INTEGER PRIMARY KEY,
    net_id	INTEGER NOT NULL,
    name	TEXT UNIQUE NOT NULL,
    addr        TEXT NOT NULL DEFAULT '[]',
    os          TEXT NOT NULL DEFAULT '',
    bighead     INTEGER NOT NULL DEFAULT 1,
    last_seen   INTEGER NOT NULL DEFAULT 0,
    CHECK (json_valid(addr)),
    FOREIGN KEY (net_id) REFERENCES network (id)
        ON UPDATE RESTRICT
        ON DELETE CASCADE
) STRICT
`,
	"CREATE INDEX dev_big_idx ON device (bighead <> 0)",
	"CREATE INDEX dev_last_idx ON device (last_seen)",
	`
CREATE TABLE uptime (
    id INTEGER PRIMARY KEY,
    dev_id INTEGER NOT NULL,
    timestamp INTEGER NOT NULL,
    uptime INTEGER NOT NULL DEFAULT 0,
    load1 REAL NOT NULL,
    load5 REAL NOT NULL,
    load15 REAL NOT NULL,
    FOREIGN KEY (dev_id) REFERENCES device (id)
        ON UPDATE RESTRICT
        ON DELETE CASCADE,
    CHECK (uptime >= 0),
    CHECK (load1 >= 0 AND load5 >= 0 AND load15 >= 0)
) STRICT
`,
	"CREATE INDEX up_dev_idx ON uptime (dev_id)",
	"CREATE INDEX up_time_idx ON uptime (timestamp)",
	`
CREATE TRIGGER up_host_contact_tr
AFTER INSERT ON uptime
BEGIN
    UPDATE device
    SET last_seen = NEW.timestamp
    WHERE id = NEW.dev_id;
END
`,
	`
CREATE TABLE updates (
    id INTEGER PRIMARY KEY,
    dev_id INTEGER NOT NULL,
    timestamp INTEGER NOT NULL,
    updates TEXT NOT NULL DEFAULT '[]',
    UNIQUE (dev_id, timestamp),
    CHECK (json_valid(updates)),
    FOREIGN KEY (dev_id) REFERENCES device (id)
      ON UPDATE RESTRICT
      ON DELETE CASCADE
) STRICT
`,
	"CREATE INDEX upd_dev_idx ON updates (dev_id)",
	"CREATE INDEX upd_time_idx ON updates (timestamp)",
	`
CREATE TRIGGER upd_host_contact_tr
AFTER INSERT ON updates
BEGIN
    UPDATE device
    SET last_seen = NEW.timestamp
    WHERE id = NEW.dev_id;
END
`,
	`
CREATE TABLE info (
    id INTEGER PRIMARY KEY,
    dev_id INTEGER NOT NULL,
    timestamp INTEGER NOT NULL,
    info_type INTEGER NOT NULL,
    data TEXT NOT NULL DEFAULT '',
    UNIQUE (dev_id, info_type, timestamp),
    CHECK (json_valid(data)),
    FOREIGN KEY (dev_id) REFERENCES device (id)
        ON UPDATE RESTRICT
        ON DELETE CASCADE
) STRICT
`,
	"CREATE INDEX info_dev_idx ON info (dev_id)",
	"CREATE INDEX info_time_idx ON info (timestamp)",
	"CREATE INDEX info_type_idx ON info (info_type)",
	`
CREATE TRIGGER info_host_tr
AFTER INSERT ON info
BEGIN
    UPDATE device
    SET last_seen = NEW.timestamp
    WHERE id = NEW.dev_id;
END
`,
}
