// /home/krylon/go/src/github.com/blicero/carebear/database/qdb.go
// -*- mode: go; coding: utf-8; -*-
// Created on 04. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-08 19:50:12 krylon>

package database

import (
	"github.com/blicero/carebear/database/query"
)

var qdb = map[query.ID]string{
	query.NetworkAdd:             "INSERT INTO network (addr, desc) VALUES (?, ?) RETURNING id",
	query.NetworkUpdateScanStamp: "UPDATE network SET last_scan = ? WHERE id = ?",
	query.NetworkUpdateDesc:      "UPDATE network SET desc = ? WHERE id = ?",
	query.NetworkGetAll: `
SELECT
	id,
	addr,
	desc,
	last_scan
FROM network
`,
	query.NetworkGetByID: `
SELECT
	addr,
	desc,
	last_scan
FROM network
WHERE id = ?
`,
	query.NetworkGetByAddr: `
SELECT
	id,
	desc,
	last_scan
FROM network
WHERE addr = ?
`,
	query.DeviceAdd: `
INSERT INTO device (name, net_id, addr, bighead)
            VALUES (   ?,      ?,    ?,       ?)
RETURNING id
`,
	query.DeviceUpdateLastSeen: "UPDATE device SET last_seen = ? WHERE id = ?",
	query.DeviceGetAll: `
SELECT
    id,
    net_id,
    name,
    addr,
    bighead,
    last_seen
FROM device
ORDER BY name
`,
	query.DeviceGetByID: `
SELECT
    net_id,
    name,
    addr,
    bighead,
    last_seen
FROM device
WHERE id = ?
`,
	query.DeviceGetByName: `
SELECT
    id,
    net_id,
    addr,
    bighead,
    last_seen
FROM device
WHERE name = ?
`,
}
