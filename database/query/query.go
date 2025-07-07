// /home/krylon/go/src/github.com/blicero/carebear/database/query/query.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-07 14:50:37 krylon>

// Package query provides symbolic constants to identifiy database queries.
package query

//go:generate stringer -type=ID

// ID represents a database query.
type ID uint8

const (
	DeviceAdd ID = iota
	DeviceUpdateLastSeen
	DeviceGetAll
	DeviceGetByID
	DeviceGetByName
)
