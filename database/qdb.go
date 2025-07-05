// /home/krylon/go/src/github.com/blicero/carebear/database/qdb.go
// -*- mode: go; coding: utf-8; -*-
// Created on 04. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-05 15:03:46 krylon>

package database

import (
	"github.com/blicero/carebear/database/query"
)

var qdb = map[query.ID]string{
	query.DeviceAdd: `
INSERT INTO device (name, addr, bighead)
            VALUES (   ?,    ?,       ?)
RETURNING id
`,
	query.DeviceUpdateLastSeen: "UPDATE device SET last_seen = ? WHERE id = ?",
}
