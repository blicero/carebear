// /home/krylon/go/src/github.com/blicero/carebear/scheduler/scheduler.go
// -*- mode: go; coding: utf-8; -*-
// Created on 24. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-08-16 20:00:15 krylon>

// Package scheduler provides the logic to schedule tasks and execute them.
package scheduler

import (
	"log"
	"os"
	"path/filepath"
	"strings"
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
	probing "github.com/prometheus-community/pro-bing"
)

const (
	dbPoolSize     = 4
	checkInterval  = time.Second * 15 // TODO: Adjust to higher value after testing/debugging
	probeWorkerCnt = 8
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
	} else if s.sc, err = scanner.NewNetworkScanner(); err != nil {
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
	s.log.Println("[INFO] Scheduler starting up.")
	s.log.Printf("[INFO] Scan interval: Net = %s, Devices = %s, Ping = %s, Updates = %s\n",
		settings.Settings.ScanIntervalNet,
		settings.Settings.ScanIntervalDev,
		checkInterval,
		settings.Settings.ProbeIntervalUpdates)

	defer s.log.Println("[INFO] Scheduler is quitting now.")

	var (
		tickScanNet      = time.NewTicker(settings.Settings.ScanIntervalNet)
		tickScanDev      = time.NewTicker(settings.Settings.ScanIntervalDev)
		tickCheckLive    = time.NewTicker(checkInterval)
		tickQueryUpdates = time.NewTicker(settings.Settings.ProbeIntervalUpdates)
	)

	defer tickScanNet.Stop()
	defer tickScanDev.Stop()
	defer tickCheckLive.Stop()
	defer tickQueryUpdates.Stop()

	for s.IsActive() {
		select {
		case <-tickScanNet.C:
			s.log.Println("[INFO] Initiate network scan.")
			s.sc.CmdQ <- command.Command{ID: command.ScanStart}
		case <-tickScanDev.C:
			s.log.Println("[INFO] Probe Devices")
			go s.scanDevices()
		case <-tickCheckLive.C:
			s.log.Println("[INFO] Start Ping scan")
			go s.pingDevices()
		case <-tickQueryUpdates.C:
			s.log.Println("[INFO] Query pending updates")
			var updateQ = make(chan *model.Device)
			go s.deviceDispatch(updateQ)

			for i := range probeWorkerCnt {
				go s.queryDeviceUpdateWorker(i, updateQ)
			}
		}
	}
} // func (s *Scheduler) run()

func (s *Scheduler) pingDevices() {
	var pingQ = make(chan *model.Device)

	go s.deviceDispatch(pingQ)

	for i := range probeWorkerCnt {
		go s.pingWorker(i, pingQ)
	}
} // func (s *Scheduler) pingDevices()

func (s *Scheduler) pingWorker(id int, pq chan *model.Device) {
	var (
		err error
		db  *database.Database
	)

	defer s.log.Printf("[TRACE] Ping worker %02d is finished\n", id)

	if db, err = s.pool.GetNoWait(); err != nil {
		s.log.Printf("[CRITICAL] Cannot open database connection: %s\n",
			err.Error())
		return
	}
	defer s.pool.Put(db)

	for d := range pq {
		var (
			ping *probing.Pinger
		)

		if ping, err = probing.NewPinger(d.AddrStr()); err != nil {
			s.log.Printf("[ERROR] Ping%02d Failed to create Pinger for %s: %s\n",
				id,
				d.AddrStr(),
				err.Error())
			return
		}

		ping.Interval = settings.Settings.PingInterval
		ping.Timeout = settings.Settings.PingTimeout
		ping.Count = int(settings.Settings.PingCount)

		if err = ping.Run(); err != nil {
			s.log.Printf("[ERROR] Ping%02d Failed to run Pinger on %s: %s\n",
				id,
				d.AddrStr(),
				err.Error())
			return
		}

		var stats = ping.Statistics()
		if stats.PacketLoss < 100 {
			if err = db.DeviceUpdateLastSeen(d, time.Now()); err != nil {
				s.log.Printf("[ERROR] Ping%02d Cannot update LastSeen timestamp for %s: %s\n",
					id,
					d.Name,
					err.Error())
			} else {
				s.log.Printf("[DEBUG] Ping%02d - Device %s is alive\n",
					id,
					d.Name)
			}
		} else {
			s.log.Printf("[DEBUG] Ping%02d - Device %s is offline\n",
				id,
				d.Name)
		}
	}
} // func (s *Scheduler) pingWorker(id int, pq chan *model.Device)

func (s *Scheduler) scanDevices() {
	var (
		err  error
		db   *database.Database
		devs []*model.Device
	)

	db = s.pool.Get()
	defer s.pool.Put(db)

	if devs, err = db.DeviceGetAll(); err != nil {
		s.log.Printf("[ERROR] Failed to load all Devices: %s\n",
			err.Error())
		return
	}

	if len(devs) == 0 {
		return
	}

	var devQ = make(chan *model.Device, 2)

	for i := range probeWorkerCnt {
		go s.deviceProbeWorker(i+1, devQ)
	}

	for _, d := range devs {
		devQ <- d
	}

	close(devQ)
} // func (s *Scheduler) scanDevices()

func (s *Scheduler) deviceProbeWorker(id int, devQ <-chan *model.Device) {
	var (
		err error
		up  *model.Uptime
		db  *database.Database
	)

	defer s.log.Printf("[TRACE] Device Probe Worker #%02d is quitting.\n",
		id)

	if db, err = s.pool.GetNoWait(); err != nil {
		s.log.Printf("[ERROR] Failed to open database connection for Probe worker %02d: %s\n",
			id,
			err.Error())
		return
	}

	defer s.pool.Put(db)

	for d := range devQ {
		s.log.Printf("[DEBUG] Probe device %s (%d)\n",
			d.Name, d.ID)

		if !d.BigHead {
			s.log.Printf("[DEBUG] Device %s is irrelevant.\n",
				d.Name)
			continue
		} else if d.OS == "" {
			s.log.Printf("[INFO] Probing OS of device %s\n",
				d.Name)
			var osname string
			if osname, err = s.p.QueryOS(d, 22); err != nil {
				s.log.Printf("[ERROR] Failed to query %s for its OS: %s\n",
					d.Name, err.Error())
				continue
			} else if err = db.DeviceUpdateOS(d, osname); err != nil {
				s.log.Printf("[ERROR] Failed to set OS of %s to %s: %s\n",
					d.Name,
					osname,
					err.Error())
				continue
			}
		}

		if up, err = s.p.QueryUptime(d, 22); err != nil {
			s.log.Printf("[ERROR] Failed to query uptime of Device %s: %s\n",
				d.Name,
				err.Error())
			continue
		} else if up == nil {
			s.log.Println("[CANTHAPPEN] QueryUptime did not return an error, but value was nil")
			continue
		} else if err = db.UptimeAdd(up); err != nil {
			s.log.Printf("[ERROR] Failed to add Uptime for Device %s to database: %s\n",
				d.Name,
				err.Error())
			continue
		}
	}
} // func (s *Scheduler) deviceProbeWorker(id int, devQ <-chan *model.Device)

func (s *Scheduler) deviceDispatch(devQ chan<- *model.Device) {
	var (
		err  error
		db   *database.Database
		devs []*model.Device
		cnt  int
	)

	defer func() {
		close(devQ)
		s.log.Printf("[TRACE] deviceDispatch is quitting after dispatching %02d Devices.\n",
			cnt)
	}()

	db = s.pool.Get()
	defer s.pool.Put(db)

	if devs, err = db.DeviceGetAll(); err != nil {
		s.log.Printf("[ERROR] Failed to load all Devices: %s\n",
			err.Error())
		return
	}

	for _, d := range devs {
		devQ <- d
		cnt++
	}
} // func (s *Scheduler) deviceDispatch(devQ chan <-*model.Device)

func (s *Scheduler) queryDeviceUpdateWorker(id int, devQ <-chan *model.Device) {
	var (
		err error
		db  *database.Database
	)

	defer s.log.Printf("[DEBUG] queryDeviceUpdateWorker #%02d is quitting.\n", id)

	db = s.pool.Get()
	defer s.pool.Put(db)

	for d := range devQ {
		s.log.Printf("[DEBUG] %02d: Query %s for pending updates\n",
			id+1,
			d.Name)

		var updates = &model.Updates{
			DevID:     d.ID,
			Timestamp: time.Now(),
		}

		if updates.AvailableUpdates, err = s.p.QueryUpdates(d, 22); err != nil {
			s.log.Printf("[ERROR] Failed to query %s for pending updates: %s\n",
				d.Name,
				err.Error())
			continue
		} else if err = db.UpdatesAdd(updates); err != nil {
			s.log.Printf("[ERROR] Failed to stored update set for %s to database: %s\n%s\n",
				d.Name,
				err.Error(),
				strings.Join(updates.AvailableUpdates, "\n"))
			continue
		}

		if len(updates.AvailableUpdates) > 0 {
			s.log.Printf("[DEBUG] Device %s has %d available updates:\n%s\n",
				d.Name,
				len(updates.AvailableUpdates),
				strings.Join(updates.AvailableUpdates, "\n"))
		}

	}
} // func (s *Scheduler) queryDeviceUpdateWorker()
