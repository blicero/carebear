// /home/krylon/go/src/github.com/blicero/carebear/scanner/scanner.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-12 19:11:10 krylon>

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
	"github.com/blicero/carebear/scanner/command"
	probing "github.com/prometheus-community/pro-bing"
)

// TODO Increase this to a reasonable value once testing/debugging is done!
//
//	Better yet, make it configurable!
const (
	ckPeriod       = time.Second * 5
	netScanPeriod  = time.Minute
	pingCount      = 4
	pingInterval   = time.Millisecond * 250
	defaultTimeout = time.Second * 5
)

type scanProgress struct {
	n       *model.Network
	scanned atomic.Uint64
	added   atomic.Uint64
}

// Scanner traverses IP networks looking for Devices.
type Scanner struct {
	log        *log.Logger
	lock       sync.RWMutex
	CmdQ       chan command.Command
	activeFlag atomic.Bool
	db         *database.Database
	scanMap    map[int64]*scanProgress
	workerCnt  int64
	timeout    time.Duration
}

// New creates a new Scanner.
func New(wcnt int64) (*Scanner, error) {
	var (
		err error
		s   = &Scanner{
			CmdQ:      make(chan command.Command),
			scanMap:   make(map[int64]*scanProgress),
			workerCnt: wcnt,
			timeout:   defaultTimeout,
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

// ScanCnt returns the number of Networks currently being scanned.
func (s *Scanner) ScanCnt() int {
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
func (s *Scanner) ScanProgress(nid int64) (uint64, uint64, bool) {
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
		case cmd = <-s.CmdQ:
			s.log.Printf("[DEBUG] Scanner received a Command: %s\n",
				cmd)
			s.handleCommand(cmd)
		}
	}
} // func (s *Scanner) run()

func (s *Scanner) handleCommand(c command.Command) {
	var (
		err      error
		networks []*model.Network
	)

	switch c {
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
	}
} // func (s *Scanner) handleCommand(c command.Command)

func (s *Scanner) scanStart(n *model.Network) {
	s.lock.Lock()
	// defer s.lock.Unlock()
	if _, ok := s.scanMap[n.ID]; ok {
		s.log.Printf("[INFO] There appears to be a scan of network %s (%d) going on already.\n",
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
			n.Addr.String(),
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
} // func (s *Scanner) scanStart(n *model.Network)

func (s *Scanner) netScanWorker(nid, wid int64, addrQ <-chan net.IP, devQ chan<- *model.Device, wg *sync.WaitGroup) {
	// s.log.Printf("[TRACE] netScanWorker%03d coming up...\n", wid)
	defer s.log.Printf("[TRACE] netScanWorker%03d quitting...\n", wid)
	defer wg.Done()

	var prog = s.scanMap[nid]

	for addr := range addrQ {
		prog.scanned.Add(1)
		if s.pingAddr(addr) {
			var (
				err   error
				names []string
			)

			if names, err = net.LookupAddr(addr.String()); err != nil {
				s.log.Printf("[ERROR] Error looking up name for %s: %s\n",
					addr,
					err.Error())
				continue
			} else if len(names) == 0 {
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

func (s *Scanner) netScanCollector(n *model.Network, devQ <-chan *model.Device) {
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

	if db, err = database.Open(common.DbPath); err != nil {
		s.log.Printf("[ERROR] Cannot open database at %s: %s\n",
			common.DbPath,
			err.Error())
		return
	}

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
			continue
		} else if err = db.DeviceAdd(dev); err != nil {
			s.log.Printf("[ERROR] Failed to add Device %s (%s) to database: %s\n",
				dev.Name,
				dev.Addr[0],
				err.Error())
			continue
		}
	}
} // func (s *Scanner) netScanCollector(devQ <-chan *model.Device)

func (s *Scanner) pingAddr(addr net.IP) bool {
	var (
		err  error
		ping *probing.Pinger
	)

	if ping, err = probing.NewPinger(addr.String()); err != nil {
		s.log.Printf("[ERROR] Failed to create Pinger for %s: %s\n",
			addr,
			err.Error())
		return false
	}

	ping.Interval = pingInterval
	ping.Timeout = s.timeout
	ping.Count = pingCount

	if err = ping.Run(); err != nil {
		s.log.Printf("[ERROR] Failed to run Pinger on %s: %s\n",
			addr,
			err.Error())
		return false
	}

	var stats = ping.Statistics()

	return stats.PacketLoss < 100
} // func pingAddr(addr net.IP) bool

func netIsDue(n *model.Network) bool {
	return n.LastScan.Add(netScanPeriod).Before(time.Now())
} // func netIsDue(n *model.Network) bool
