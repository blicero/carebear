// /home/krylon/go/src/github.com/blicero/carebear/model/model.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-08-09 16:56:31 krylon>

// Package model provides data types used throughout the application.
package model

import (
	"net"
	"strings"
	"time"

	"github.com/blicero/carebear/settings"
	"github.com/korylprince/ipnetgen"
)

// Network represents a range of IP addresses where Devices may reside.
type Network struct {
	ID          int64
	Addr        *net.IPNet
	Description string
	LastScan    time.Time
}

// NewNetwork creates a fresh Network with the given address and description.
func NewNetwork(addr, desc string) (*Network, error) {
	var (
		err error
		n   = &Network{Description: desc}
	)

	if _, n.Addr, err = net.ParseCIDR(addr); err != nil {
		return nil, err
	}

	return n, nil
} // func NewNetwork(addr, desc string) (*Network, error)

// Enumerate generates all IP addresses for the Network and sends them through the channel
// passed in as its argument.
func (n *Network) Enumerate(q chan<- net.IP) error {
	gen, err := ipnetgen.New(n.Addr.String())

	if err != nil {
		return err
	}

	go func() {
		for ip := gen.Next(); ip != nil; ip = gen.Next() {
			q <- ip
		}
		close(q)
	}()

	return nil
} // func (n *Network) Enumerate(q chan<- net.IP)

// Device is a computer - in the most inclusive sense of the word - that is connected to
// an IP network.
// It has zero or more IP addresses, a name, and is considered a BigHead if it is a *REAL* computer,
// which by my definition is one you can do some coding on (i.e. smartphones, tablets, smart TVs, etc.
// are NOT BigHeads).
type Device struct {
	ID       int64
	NetID    int64
	Name     string
	OS       string
	Addr     []net.Addr
	BigHead  bool
	LastSeen time.Time
}

// IsLive returns true if the last interaction with the device was within the
// period configured as LiveTimeout.
func (d *Device) IsLive() bool {
	return time.Since(d.LastSeen) < settings.Settings.LiveTimeout
} // func (d *Device) IsLive() bool

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

// DefaultAddr returns the first IP address, stringified.
func (d *Device) DefaultAddr() string {
	return d.Addr[0].String()
} // func (d *Device) DefaultAddr() string

// Uptime captures the time a Device has been running since last reboot/power-on
// as well as the current system load average.
type Uptime struct {
	ID        int64
	DevID     int64
	Timestamp time.Time
	Uptime    time.Duration
	Load      [3]float64
}

// Updates is a set of available Updates on a given Device at a certain point in time.
type Updates struct {
	ID               int64
	DevID            int64
	Timestamp        time.Time
	AvailableUpdates []string
}

// For posterity, I leave this commented out without removing it:
// http://play.golang.org/p/m8TNTtygK0
// func inc(ip net.IP) {
// 	for j := len(ip) - 1; j >= 0; j-- {
// 		ip[j]++
// 		if ip[j] > 0 {
// 			break
// 		}
// 	}
// }
