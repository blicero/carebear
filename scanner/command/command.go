// /home/krylon/go/src/github.com/blicero/carebear/scanner/command/command.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-03 21:40:50 krylon>

package command

//go:generate stringer -type=Command

// Command identifies commands sent to the Scanner.
type Command uint8

const (
	ScanStart Command = iota
	ScanStop
)
