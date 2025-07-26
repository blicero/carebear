// /home/krylon/go/src/github.com/blicero/carebear/scheduler/scheduler.go
// -*- mode: go; coding: utf-8; -*-
// Created on 24. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-26 18:05:09 krylon>

// Package scheduler provides the logic to schedule tasks and execute them.
package scheduler

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blicero/carebear/common"
	"github.com/blicero/carebear/database"
	"github.com/blicero/carebear/logdomain"
)

const (
	dbPoolSize    = 4
	checkInterval = time.Second * 15 // TODO: Adjust to higher value after testing/debugging
)

type TaskTag uint8

const (
	NetworkScan TaskTag = iota
	DevicePing
	DeviceProbeSysload
)

// Scheduler wraps the state needed to schedule the scanning of networks, probing
// of devices, possibly other tasks, too.
type Scheduler struct {
	log    *log.Logger
	pool   *database.Pool
	lock   sync.RWMutex
	active atomic.Bool
}

// Create returns a fresh Scheduler.
func Create() (*Scheduler, error) {
	var (
		err error
		s   = new(Scheduler)
	)

	if s.log, err = common.GetLogger(logdomain.Scheduler); err != nil {
		return nil, err
	} else if s.pool, err = database.NewPool(dbPoolSize); err != nil {
		return nil, err
	}

	return s, nil
} // func Create() (*Scheduler, error)

// IsActive returns the state of the Scheduler's active flag.
func (s *Scheduler) IsActive() bool {
	return s.active.Load()
} // func (s *Scheduler) IsActive() bool

// Stop clears the Scheduler's active flag.
func (s *Scheduler) Stop() {
	s.active.Store(false)
} // func (s *Scheduler) Stop()

func (s *Scheduler) run() {
	var (
		ticker = time.NewTicker(checkInterval)
	)

	defer ticker.Stop()

	for s.IsActive() {
		select {
		case <-ticker.C:
			continue
		}
	}
} // func (s *Scheduler) run()
