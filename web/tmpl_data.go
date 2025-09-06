// /home/krylon/go/src/carebear/web/tmpl_data.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 05. 2020 by Benjamin Walkenhorst
// (c) 2020 Benjamin Walkenhorst
// Time-stamp: <2025-09-06 16:16:11 krylon>
//
// This file contains data structures to be passed to HTML templates.

package web

import (
	"github.com/blicero/carebear/model"
	"github.com/blicero/carebear/scanner"
)

type tmplDataBase struct { // nolint: unused
	Title      string
	Messages   []*message
	Debug      bool
	TestMsgGen bool
	URL        string
}

type tmplDataIndex struct { // nolint: unused,deadcode
	tmplDataBase
}

type tmplDataNetworkAll struct { // nolint: unused,deadcode
	tmplDataBase
	Networks []*model.Network
	DevCnt   map[int64]int
	Network  *model.Network
	Scans    map[int64]*scanner.ScanProgress
}

type tmplDataNetworkDetails struct {
	tmplDataBase
	Network *model.Network
	Devices []*model.Device
}

type tmplDataDeviceAll struct {
	tmplDataBase
	Devices []*model.Device
	Updates map[int64]*model.Updates
	Disk    map[int64]*model.DiskFree
}

func (d *tmplDataDeviceAll) DiskFree(devID int64) int64 {
	var (
		free *model.DiskFree
		ok   bool
	)

	if free, ok = d.Disk[devID]; ok {
		return free.PercentFree
	}

	return 100
} // func (d *tmplDataDeviceAll) DiskFree(devID int64) (int64, bool)

type tmplDataDeviceDetails struct {
	tmplDataBase
	Device  *model.Device
	Network *model.Network
	Uptime  *model.Uptime
	Updates *model.Updates
}

// Local Variables:  //
// compile-command: "go generate && go vet && go build -v -p 16 && gometalinter && go test -v" //
// End: //
