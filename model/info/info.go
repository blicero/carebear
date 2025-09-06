// /home/krylon/go/src/github.com/blicero/carebear/model/info/info.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 09. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-09-05 19:44:35 krylon>

// Package info provides symbolic constants to identify the types of information
// queried on remote Devices.
package info

//go:generate stringer -type=ID

// ID represents a type of information gathered from a Device
type ID uint8

const (
	DiskFree ID = iota
	Temperature
	NeedReboot
	LoadAvg
)
