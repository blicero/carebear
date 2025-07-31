// /home/krylon/go/src/github.com/blicero/carebear/scheduler/task/tag.go
// -*- mode: go; coding: utf-8; -*-
// Created on 26. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-31 17:48:43 krylon>

// Package task defines constants to refer to Task types
package task

//go:generate stringer -type=Tag

// Tag is a symbolic constant to describe different types of Tasks
type Tag uint8

const (
	NetworkScan Tag = iota
	DevicePing
	DeviceProbeSysload
	Shutdown
)
