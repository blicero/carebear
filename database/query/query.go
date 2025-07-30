// /home/krylon/go/src/github.com/blicero/carebear/database/query/query.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-30 18:37:59 krylon>

// Package query provides symbolic constants to identifiy database queries.
package query

//go:generate stringer -type=ID

// ID represents a database query.
type ID uint8

const (
	NetworkAdd ID = iota
	NetworkUpdateScanStamp
	NetworkUpdateDesc
	NetworkGetAll
	NetworkGetByID
	NetworkGetByAddr
	NetworkDevCnt
	DeviceAdd
	DeviceUpdateLastSeen
	DeviceGetAll
	DeviceGetByID
	DeviceGetByName
	DeviceGetByNetwork
	UptimeAdd
	UptimeGetByID
	UptimeGetByDevice
	UptimeGetByPeriod
	UptimeGetCurrent
	UptimeGetMostRecent
)
