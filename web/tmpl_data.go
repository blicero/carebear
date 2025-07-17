// /home/krylon/go/src/carebear/web/tmpl_data.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 05. 2020 by Benjamin Walkenhorst
// (c) 2020 Benjamin Walkenhorst
// Time-stamp: <2025-07-17 10:56:18 krylon>
//
// This file contains data structures to be passed to HTML templates.

package web

import "github.com/blicero/carebear/model"

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
}

type tmplDataNetworkDetails struct {
	tmplDataBase
	Network *model.Network
	Devices []*model.Device
}

type tmplDataDeviceAll struct {
	tmplDataBase
	Devices []*model.Device
}

// Local Variables:  //
// compile-command: "go generate && go vet && go build -v -p 16 && gometalinter && go test -v" //
// End: //
