// /home/krylon/go/src/github.com/blicero/carebear/scanner/scanner.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-07 16:27:00 krylon>

package scanner

import (
	"log"
	"sync/atomic"

	"github.com/blicero/carebear/scanner/command"
)

// Scanner traverses IP networks looking for Devices.
type Scanner struct {
	log        *log.Logger
	cmdQ       chan command.Command
	activeFlag atomic.Bool
}
