// /home/krylon/go/src/github.com/blicero/carebear/database/02_net_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 08. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-08 19:38:56 krylon>

package database

import (
	"testing"

	"github.com/blicero/carebear/model"
)

const tnetAddr = "192.168.0.0/24"

var tnet *model.Network

func TestNetworkAdd(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	var (
		err error
	)

	if tnet, err = model.NewNetwork(tnetAddr, "Sample network"); err != nil {
		t.Fatalf("Creating a Network failed: %s", err.Error())
	} else if err = tdb.NetworkAdd(tnet); err != nil {
		t.Fatalf("Adding network to database failed: %s", err.Error())
	}
} // func TestNetworkAdd(t *testing.T)
