// /home/krylon/go/src/github.com/blicero/carebear/scheduler/scheduler.go
// -*- mode: go; coding: utf-8; -*-
// Created on 24. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-08-02 16:30:05 krylon>

// Package scheduler provides the logic to schedule tasks and execute them.
package scheduler

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blicero/carebear/common"
	"github.com/blicero/carebear/database"
	"github.com/blicero/carebear/logdomain"
	"github.com/blicero/carebear/model"
	"github.com/blicero/carebear/probe"
	"github.com/blicero/carebear/scanner"
	"github.com/blicero/carebear/scanner/command"
	"github.com/blicero/carebear/scheduler/task"
	"github.com/blicero/carebear/settings"
)

const (
	dbPoolSize    = 4
	checkInterval = time.Second * 15 // TODO: Adjust to higher value after testing/debugging
	workerCnt     = 64
)

// Task defines describes a Task. Aren't you sorry, you asked?
type Task struct {
	Kind     task.Tag
	ObjectID int64
}

// Scheduler wraps the state needed to schedule the scanning of networks, probing
// of devices, possibly other tasks, too.
type Scheduler struct {
	log    *log.Logger
	pool   *database.Pool
	lock   sync.RWMutex // nolint: unused
	active atomic.Bool
	sc     *scanner.NetworkScanner
	p      *probe.Probe
	TaskQ  chan Task
}

// Create returns a fresh Scheduler.
func Create() (*Scheduler, error) {
	var (
		err                     error
		username, keypath, home string
		s                       = &Scheduler{
			TaskQ: make(chan Task),
		}
	)

	username = os.Getenv("USER")
	home = os.Getenv("HOME")
	keypath = filepath.Join(home, ".ssh")

	if s.log, err = common.GetLogger(logdomain.Scheduler); err != nil {
		return nil, err
	} else if s.pool, err = database.NewPool(dbPoolSize); err != nil {
		return nil, err
	} else if s.sc, err = scanner.NewNetworkScanner(workerCnt); err != nil {
		return nil, err
	} else if s.p, err = probe.New(username, keypath); err != nil {
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

// Start starts the Scheduler's main loop.
func (s *Scheduler) Start() {
	s.active.Store(true)
	go s.run()
} // func (s *Scheduler) Start()

func (s *Scheduler) run() {
	var (
		tickScanNet   = time.NewTicker(settings.Settings.ScanIntervalNet)
		tickScanDev   = time.NewTicker(settings.Settings.ScanIntervalDev)
		tickCheckLive = time.NewTicker(checkInterval)
	)

	defer tickScanNet.Stop()
	defer tickScanDev.Stop()
	defer tickCheckLive.Stop()

	for s.IsActive() {
		select {
		case <-tickScanNet.C:
			s.log.Println("[DEBUG] Initiate network scan.")
			s.sc.CmdQ <- command.Command{ID: command.ScanStart}
		case <-tickScanDev.C:
			s.log.Println("[INFO] IMPLEMENTME - Scan Devices")
		case <-tickCheckLive.C:
			continue
		}
	}
} // func (s *Scheduler) run()

func (s *Scheduler) scanDevices() {
	var (
		err  error
		db   *database.Database
		devs []*model.Device
	)

	db = s.pool.Get()
	defer s.pool.Put(db)

	if err = db.Begin(); err != nil {
		s.log.Printf("[ERROR] Cannot start database transaction: %s\n",
			err.Error())
	} else if devs, err = db.DeviceGetAll(); err != nil {
		s.log.Printf("[ERROR] Failed to load all Devices: %s\n",
			err.Error())
		return
	}

	for _, d := range devs {
		if !d.BigHead {
			continue
		} else if d.OS == "" {
			var osname string
			if osname, err = s.p.QueryOS(d, 22); err != nil {
				s.log.Printf("[ERROR] Failed to query %s for its OS: %s\n",
					d.Name, err.Error())
			} else if err = db.DeviceUpdateOS(d, osname); err != nil {
				s.log.Printf("[ERROR] Failed to set OS of %s to %s: %s\n",
					d.Name,
					osname,
					err.Error())
			}
		}
	}
} // func (s *Scheduler) scanDevices()

// func (s *Scheduler) run() {
// 	var (
// 		ticker = time.NewTicker(checkInterval)
// 	)

// 	defer ticker.Stop()

// 	for s.IsActive() {
// 		var t Task
// 		select {
// 		case <-ticker.C:
// 			continue
// 		case t = <-s.TaskQ:
// 			switch t.Kind {
// 			case task.NetworkScan:
// 				var cmd command.Command

// 				if t.ObjectID == 0 {
// 					cmd.ID = command.ScanStart
// 				} else {
// 					cmd.ID = command.ScanOne
// 					cmd.Target = t.ObjectID
// 				}

// 				s.sc.CmdQ <- cmd
// 			}
// 		}
// 	}
// } // func (s *Scheduler) run()
