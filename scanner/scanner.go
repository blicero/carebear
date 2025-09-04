// /home/krylon/go/src/github.com/blicero/carebear/scanner/scanner.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-09-03 19:20:26 krylon>

package scanner

import (
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blicero/carebear/common"
	"github.com/blicero/carebear/database"
	"github.com/blicero/carebear/logdomain"
	"github.com/blicero/carebear/model"
	"github.com/blicero/carebear/ping"
	"github.com/blicero/carebear/scanner/command"
	"github.com/blicero/carebear/settings"
)

// TODO Increase this to a reasonable value once testing/debugging is done!
//
//	Better yet, make it configurable!
const (
	pingCount      = 4
	pingInterval   = time.Millisecond * 250
	defaultTimeout = time.Second * 5
	ckPeriod       = time.Second * 5
)

var (
	netScanPeriod time.Duration = time.Minute * 10
)

type scanProgress struct {
	n       *model.Network
	scanned atomic.Uint64
	added   atomic.Uint64
}

// NetworkScanner traverses IP networks looking for Devices.
type NetworkScanner struct {
	log        *log.Logger
	lock       sync.RWMutex
	CmdQ       chan command.Command
	activeFlag atomic.Bool
	db         *database.Database
	scanMap    map[int64]*scanProgress
	workerCnt  int64
	timeout    time.Duration
	pp         *ping.Pinger
}

// NewNetworkScanner creates a new NetworkScanner.
func NewNetworkScanner() (*NetworkScanner, error) {
	var (
		err error
		s   = &NetworkScanner{
			CmdQ:    make(chan command.Command),
			scanMap: make(map[int64]*scanProgress),
			timeout: defaultTimeout,
		}
	)

	if settings.Settings != nil {
		s.workerCnt = settings.Settings.ScanWorkerCount
		netScanPeriod = settings.Settings.ScanIntervalNet
	}

	if s.log, err = common.GetLogger(logdomain.Scanner); err != nil {
		var ex = fmt.Errorf("Cannot create Logger for Scanner: %w", err)
		return nil, ex
	} else if s.db, err = database.Open(common.DbPath); err != nil {
		s.log.Printf("[ERROR] Cannot open database: %s\n", err.Error())
		return nil, err
	} else if s.pp, err = ping.Create(); err != nil {
		s.log.Printf("[ERROR] Cannot create Pinger: %s\n",
			err.Error())
		return nil, err
	}

	return s, nil
} // func New() (*Scanner, error)

// Active returns the value of the Scanner's active flag.
func (s *NetworkScanner) Active() bool {
	return s.activeFlag.Load()
} // func (s *Scanner) Active() bool

// Start initiates the scan engine.
func (s *NetworkScanner) Start() {
	s.activeFlag.Store(true)
	go s.run()
}

// Stop tells the Scanner to stop.
func (s *NetworkScanner) Stop() {
	s.activeFlag.Store(false)
} // func (s *Scanner) Stop()

// ScanCnt returns the number of Networks currently being scanned.
func (s *NetworkScanner) ScanCnt() int {
	s.lock.RLock()
	var cnt = len(s.scanMap)
	s.lock.RUnlock()
	return cnt
} // func (s *Scanner) ScanCnt() int

// ScanProgress returns the progress of scanning the given Network.
// In particular, it returns:
// - the number of IP addresses scanned so far
// - the number Devices added so far
// - whether or not the given Network is currently being scanned.
// If the given Network is not currently being scanned, the first two
// numbers will be 0, obviously.
func (s *NetworkScanner) ScanProgress(nid int64) (uint64, uint64, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var (
		ok              bool
		prog            *scanProgress
		scanCnt, addCnt uint64
	)

	if prog, ok = s.scanMap[nid]; !ok {
		return 0, 0, false
	}

	scanCnt = prog.scanned.Load()
	addCnt = prog.added.Load()

	return scanCnt, addCnt, true
} // func (s *Scanner) ScanProgress(nid int64) (uint64, uint64, bool)

func (s *NetworkScanner) run() {
	var (
		cmd  command.Command
		tick = time.NewTicker(ckPeriod)
	)

	defer tick.Stop()

	for s.Active() {
		select {
		case <-tick.C:
			continue
		case cmd = <-s.CmdQ:
			s.log.Printf("[DEBUG] Scanner received a Command: %s\n",
				cmd.ID)
			s.handleCommand(cmd)
		}
	}
} // func (s *Scanner) run()

func (s *NetworkScanner) handleCommand(c command.Command) {
	var (
		err      error
		networks []*model.Network
	)

	switch c.ID {
	case command.ScanStart:
		s.log.Println("[INFO] Starting scan of Networks.")
		if networks, err = s.db.NetworkGetAll(); err != nil {
			s.log.Printf("[ERROR] Failed to load Networks from Database: %s\n",
				err.Error())
			return
		}

		for _, n := range networks {
			if netIsDue(n) {
				go s.scanStart(n)
			}
		}
	case command.ScanOne:
		var (
			nw *model.Network
			id = c.Target
		)

		if nw, err = s.db.NetworkGetByID(id); err != nil {
			s.log.Printf("[ERROR] Failed to get Network %d from database: %s\n",
				id,
				err.Error())
		} else if nw == nil {
			s.log.Printf("[INFO] Network %d was not found in database.\n",
				id)
		} else if !netIsDue(nw) {
			s.log.Printf("[INFO] Network %s is not due for a scan, but we were told to.\n",
				nw.Addr)
		}

		go s.scanStart(nw)
	}
} // func (s *Scanner) handleCommand(c command.Command)

func (s *NetworkScanner) scanStart(n *model.Network) {
	s.lock.Lock()
	// defer s.lock.Unlock()
	if _, ok := s.scanMap[n.ID]; ok {
		s.log.Printf("[TRACE] There appears to be a scan of network %s (%d) going on already.\n",
			n.Addr.String(),
			n.ID)
		s.lock.Unlock()
		return
	}

	s.log.Printf("[INFO] Starting scan of Network #%d (%s)\n",
		n.ID,
		n.Addr)

	s.scanMap[n.ID] = &scanProgress{n: n}
	s.lock.Unlock()

	defer func() {
		s.lock.Lock()
		delete(s.scanMap, n.ID)
		s.lock.Unlock()
	}()

	var (
		err   error
		wid   int64
		wg    sync.WaitGroup
		addrQ = make(chan net.IP)
		devQ  = make(chan *model.Device)
	)

	if err = n.Enumerate(addrQ); err != nil {
		s.log.Printf("[ERROR] Failed to enumerate network %s (%d): %s\n",
			n.Addr,
			n.ID,
			err.Error())
		return
	}

	for wid = range s.workerCnt {
		wg.Add(1)
		go s.netScanWorker(n.ID, wid+1, addrQ, devQ, &wg)
	}

	go s.netScanCollector(n, devQ)

	wg.Wait()
	close(devQ)

	s.log.Printf("[INFO] Scan of network %s has concluded\n",
		n.Addr)
} // func (s *Scanner) scanStart(n *model.Network)

func (s *NetworkScanner) netScanWorker(nid, wid int64, addrQ <-chan net.IP, devQ chan<- *model.Device, wg *sync.WaitGroup) {
	// s.log.Printf("[TRACE] netScanWorker%03d coming up...\n", wid)
	defer s.log.Printf("[TRACE] netScanWorker%03d quitting...\n", wid)
	defer wg.Done()

	var prog = s.scanMap[nid]

	for addr := range addrQ {
		prog.scanned.Add(1)
		if s.pp.PingAddr(addr.String()) {
			var (
				err   error
				names []string
			)

			// On my home network, I run my own DNS server and create reverse
			// mappings for all devices I own. So even if a given address is
			// pingable, if it has no reverse mappings, it's probably a smart phone
			// or a visitor's laptop or tablet. I.e. something we are not
			// interested in.
			if names, err = net.LookupAddr(addr.String()); err != nil {
				s.log.Printf("[ERROR] Error looking up name for %s: %s\n",
					addr,
					err.Error())
				continue
			} else if len(names) == 0 {
				s.log.Printf("[TRACE] No name was found for %s\n",
					addr)
				continue
			}

			var dev = &model.Device{
				NetID: nid,
				Name:  names[0],
				Addr:  []net.Addr{&net.IPAddr{IP: addr}},
			}

			devQ <- dev
			prog.added.Add(1)
		}
	}
} // func (s *Scanner) netScanWorker(id int, addrQ <-chan net.IP, wg *sync.WaitGroup)

func (s *NetworkScanner) netScanCollector(n *model.Network, devQ <-chan *model.Device) {
	s.log.Printf("[TRACE] Collector for network %d (%s) starting up\n",
		n.ID,
		n.Addr)
	defer s.log.Printf("[TRACE] Collector for network %d (%s) quitting\n",
		n.ID,
		n.Addr)

	var (
		err error
		db  *database.Database
	)

	if db, err = database.DBPool.GetNoWait(); err != nil {
		s.log.Printf("[ERROR] Cannot open database at %s: %s\n",
			common.DbPath,
			err.Error())
		return
	}

	defer database.DBPool.Put(db)

	for dev := range devQ {
		var (
			xdev *model.Device
		)

		if xdev, err = db.DeviceGetByName(dev.Name); err != nil {
			s.log.Printf("[ERROR] Couldn't look up device named %s: %s\n",
				dev.Name,
				err.Error())
			continue
		} else if xdev != nil {
			// Apparently, this device is already known
			s.log.Printf("[DEBUG] Device %s already exists in database.\n",
				dev.Name)
			continue
		} else if err = db.DeviceAdd(dev); err != nil {
			s.log.Printf("[ERROR] Failed to add Device %s (%s) to database: %s\n",
				dev.Name,
				dev.DefaultAddr(),
				err.Error())
			continue
		} else {
			s.log.Printf("[DEBUG] Added new Device %s (%s) to database\n",
				dev.Name,
				dev.DefaultAddr())
		}
	}
} // func (s *Scanner) netScanCollector(devQ <-chan *model.Device)

func netIsDue(n *model.Network) bool {
	return time.Since(n.LastScan) >= netScanPeriod
} // func netIsDue(n *model.Network) bool
