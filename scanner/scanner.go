// /home/krylon/go/src/github.com/blicero/carebear/scanner/scanner.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-08 20:29:27 krylon>

package scanner

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/blicero/carebear/common"
	"github.com/blicero/carebear/database"
	"github.com/blicero/carebear/logdomain"
	"github.com/blicero/carebear/scanner/command"
)

const ckPeriod = time.Second * 5

// Scanner traverses IP networks looking for Devices.
type Scanner struct {
	log        *log.Logger
	cmdQ       chan command.Command
	activeFlag atomic.Bool
	db         *database.Database
}

// New creates a new Scanner.
func New() (*Scanner, error) {
	var (
		err error
		s   = &Scanner{
			cmdQ: make(chan command.Command),
		}
	)

	if s.log, err = common.GetLogger(logdomain.Scanner); err != nil {
		var ex = fmt.Errorf("Cannot create Logger for Scanner: %w", err)
		return nil, ex
	} else if s.db, err = database.Open(common.DbPath); err != nil {
		s.log.Printf("[ERROR] Cannot open database: %s\n", err.Error())
		return nil, err
	}

	return s, nil
} // func New() (*Scanner, error)

// Active returns the value of the Scanner's active flag.
func (s *Scanner) Active() bool {
	return s.activeFlag.Load()
} // func (s *Scanner) Active() bool

// Start initiates the scan engine.
func (s *Scanner) Start() {
	s.activeFlag.Store(true)
	go s.run()
}

// Stop tells the Scanner to stop.
func (s *Scanner) Stop() {
	s.activeFlag.Store(false)
} // func (s *Scanner) Stop()

func (s *Scanner) run() {
	var (
		cmd  command.Command
		tick = time.NewTicker(ckPeriod)
	)

	defer tick.Stop()

	for s.Active() {
		select {
		case <-tick.C:
			continue
		case cmd = <-s.cmdQ:
			s.log.Printf("[DEBUG] Scanner received a Command: %s\n",
				cmd)
			s.handleCommand(cmd)
		}
	}
} // func (s *Scanner) run()

func (s *Scanner) handleCommand(c command.Command) {
	var (
		err error
	)
} // func (s *Scanner) handleCommand(c command.Command)
