// /home/krylon/go/src/github.com/blicero/carebear/logdomain/logdomain.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-08-19 18:09:59 krylon>

package logdomain

// ID represents the various pieces of the application that may want to log messages.
type ID uint8

//go:generate stringer -type=ID

const (
	Common ID = iota
	Database
	DBPool
	Ping
	Probe
	Scanner
	Scheduler
	Web
)

// AllDomains returns a slice of all valid values for logdomain.ID
func AllDomains() []ID {
	return []ID{
		Common,
		Database,
		DBPool,
		Ping,
		Probe,
		Scanner,
		Scheduler,
		Web,
	}
} // func AllDomains() []ID
