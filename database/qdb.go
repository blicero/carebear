// /home/krylon/go/src/github.com/blicero/carebear/database/qdb.go
// -*- mode: go; coding: utf-8; -*-
// Created on 04. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-30 18:45:41 krylon>

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
	query.NetworkDevCnt: `
SELECT
	net_id,
	COUNT(net_id) AS cnt
FROM device
GROUP BY net_id
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
	query.DeviceGetByNetwork: `
SELECT
    id,
    name,
    addr,
    bighead,
    last_seen
FROM device
WHERE net_id = ?
`,
	query.UptimeAdd: `
INSERT INTO uptime (dev_id, timestamp, uptime, load1, load5, load15)
            VALUES (     ?,         ?,      ?,     ?,     ?,      ?)
RETURNING id
`,
	query.UptimeGetByID: `
SELECT
    dev_id,
    timestamp,
    uptime,
    load1,
    load5,
    load15
FROM uptime
WHERE id = ?
`,
	query.UptimeGetByDevice: `
SELECT
    id,
    timestamp,
    load1,
    load5,
    load15
FROM uptime
WHERE dev_id = ?
ORDER BY timestamp DESC
LIMIT ?
`,
	query.UptimeGetByPeriod: `
SELECT
    id,
    dev_id,
    timestamp,
    uptime,
    load1,
    load5,
    load15
FROM uptime
WHERE timestamp BETWEEN ? AND ?
ORDER BY timestamp DESC
`,
}
