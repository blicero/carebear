// /home/krylon/go/src/github.com/blicero/grace/common/idgen.go
// -*- mode: go; coding: utf-8; -*-
// Created on 29. 12. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-12-29 16:54:30 krylon>

package common

import "sync"

// IDGen generates unique IDs (unique per IDGenerator, that is).
type IDGen struct {
	cnt  int64
	lock sync.Mutex
}

// Next returns a fresh, unique ID.
func (g *IDGen) Next() int64 {
	g.lock.Lock()
	g.cnt++
	var i = g.cnt
	g.lock.Unlock()
	return i
} // func (g *IDGen) Next() int64
