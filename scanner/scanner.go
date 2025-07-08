// /home/krylon/go/src/github.com/blicero/carebear/scanner/scanner.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-08 11:57:28 krylon>

package scanner

import (
	"fmt"
	"log"
	"sync/atomic"

	"github.com/blicero/carebear/common"
	"github.com/blicero/carebear/database"
	"github.com/blicero/carebear/logdomain"
	"github.com/blicero/carebear/scanner/command"
)

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
