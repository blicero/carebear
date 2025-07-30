// /home/krylon/go/src/github.com/blicero/carebear/scanner/command/command.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-30 18:07:37 krylon>

package command

//go:generate stringer -type=Command

// ID identifies commands sent to the Scanner.
type ID uint8

const (
	ScanStart ID = iota
	ScanStop
	ScanOne
)

type Command struct {
	ID     ID
	Target int64
}
