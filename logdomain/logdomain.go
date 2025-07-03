// /home/krylon/go/src/github.com/blicero/carebear/logdomain/logdomain.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-03 18:16:56 krylon>

package logdomain

// ID represents the various pieces of the application that may want to log messages.
type ID uint8

//go:generate stringer -type=ID

const (
	Common ID = iota
	Database
	Scanner
)
