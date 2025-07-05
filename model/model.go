// /home/krylon/go/src/github.com/blicero/carebear/model/model.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-05 15:10:23 krylon>

// Package model provides data types used throughout the application.
package model

import (
	"net"
	"strings"
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

// AddrStr returns a string representation of the receiver's addresses
// that is also valid JSON.
func (d *Device) AddrStr() string {
	var buf strings.Builder
	var max = len(d.Addr) - 1

	buf.WriteString("[")
	for idx, a := range d.Addr {
		buf.WriteString("\"")
		buf.WriteString(a.String())
		buf.WriteString("\"")
		if idx < max {
			buf.WriteString(", ")
		}
	}

	buf.WriteString("]")
	return buf.String()
} // func (d *Device) AddrStr() string
