// /home/krylon/go/src/github.com/blicero/carebear/model/model.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-03 18:43:19 krylon>

// Package model provides data types used throughout the application.
package model

import (
	"net"
	"time"
)

// Device is a computer - in the most inclusive sense of the word - that is connected to
// an IP network.
// It has zero or more IP addresses, a name, and is considered a BigHead if it is a *REAL* computer,
// which by my definition is one you can do some coding on (i.e. smartphones, tablets, smart TVs, etc.
// are NOT BigHeads).
type Device struct {
	ID       int64
	Name     string
	Addr     []net.Addr
	BigHead  bool
	LastSeen time.Time
}
