// /home/krylon/go/src/github.com/blicero/carebear/ping/ping.go
// -*- mode: go; coding: utf-8; -*-
// Created on 19. 08. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-08-19 18:43:10 krylon>

// Package ping provides a simple API to ping Devices, mostly so that I can
// control its log level separately.
// And to a lesser degree to remove duplication of code, although it's not a
// primary concern.
package ping

import (
	"log"

	"github.com/blicero/carebear/common"
	"github.com/blicero/carebear/logdomain"
	"github.com/blicero/carebear/model"
	"github.com/blicero/carebear/settings"
	probing "github.com/prometheus-community/pro-bing"
)

// Pinger wraps the pinging of Devices.
type Pinger struct {
	log *log.Logger
}

// Create creates a new Pinger.
//
// Didn't see that coming, now, did you?
func Create() (*Pinger, error) {
	var (
		err error
		p   = new(Pinger)
	)

	if p.log, err = common.GetLogger(logdomain.Ping); err != nil {
		return nil, err
	}

	return p, nil
} // func Create() (*Pinger, error)

func (p *Pinger) Ping(d *model.Device) bool {
	var (
		err   error
		alive bool
		pp    *probing.Pinger
		stats *probing.Statistics
	)

	if pp, err = probing.NewPinger(d.DefaultAddr()); err != nil {
		p.log.Printf("[ERROR] Failed to create Pinger for %s: %s\n",
			d.AddrStr(),
			err.Error())
		goto END
	}

	pp.Interval = settings.Settings.PingInterval
	pp.Timeout = settings.Settings.PingTimeout
	pp.Count = int(settings.Settings.PingCount)

	if err = pp.Run(); err != nil {
		p.log.Printf("[ERROR] Failed to run Pinger on %s: %s\n",
			d.AddrStr(),
			err.Error())
		goto END
	}

	stats = pp.Statistics()
	p.log.Printf("[TRACE] %s - Packet loss is %f%% (%d/%d)\n",
		d.Name,
		stats.PacketLoss,
		stats.PacketsRecv,
		stats.PacketsSent)
	if stats.PacketLoss < 100 {
		p.log.Printf("[DEBUG] Device %s is alive\n",
			d.Name)
		alive = true
	} else {
		p.log.Printf("[TRACE] Device %s is offline\n",
			d.Name)
	}

END:
	return alive
} // func (p *Pinger) Ping(d *model.Device) bool

func (p *Pinger) PingAddr(addr string) bool {
	var (
		err   error
		alive bool
		pp    *probing.Pinger
		stats *probing.Statistics
	)

	if pp, err = probing.NewPinger(addr); err != nil {
		p.log.Printf("[ERROR] Failed to create Pinger for %s: %s\n",
			addr,
			err.Error())
		goto END
	}

	pp.Interval = settings.Settings.PingInterval
	pp.Timeout = settings.Settings.PingTimeout
	pp.Count = int(settings.Settings.PingCount)

	if err = pp.Run(); err != nil {
		p.log.Printf("[ERROR] Failed to run Pinger on %s: %s\n",
			addr,
			err.Error())
		goto END
	}

	stats = pp.Statistics()
	p.log.Printf("[TRACE] %s - Packet loss is %f%% (%d/%d)\n",
		addr,
		stats.PacketLoss,
		stats.PacketsRecv,
		stats.PacketsSent)
	if stats.PacketLoss < 100 {
		p.log.Printf("[DEBUG] %s is alive\n",
			addr)
		alive = true
	} else {
		p.log.Printf("[TRACE] %s is offline\n",
			addr)
	}

END:
	return alive
} // func (p *Pinger) PingAddr(addr string) bool
