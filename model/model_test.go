// /home/krylon/go/src/github.com/blicero/carebear/model/model_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-11 15:17:27 krylon>

package model

import (
	"fmt"
	"net"
	"testing"
)

const taddr = "192.168.42.0/24"

func TestEnumerate(t *testing.T) {
	var (
		err     error
		n       = new(Network)
		nq      = make(chan net.IP)
		addrExp = make(map[string]bool, 256)
		addrAct = make(map[string]bool, 256)
	)

	for i := range 256 {
		addr := fmt.Sprintf("192.168.42.%d", i)
		addrExp[addr] = true
	}

	if _, n.Addr, err = net.ParseCIDR(taddr); err != nil {
		t.Fatalf("Failed to parse network address %q: %s",
			taddr,
			err.Error())
	} else if err = n.Enumerate(nq); err != nil {
		t.Fatalf("Failed to enumerate network %s: %s",
			taddr,
			err.Error())
	}

	var cnt = 0

	for addr := range nq {
		var astr = addr.String()
		if !addrExp[astr] {
			t.Errorf("We did not expect to see addr %s",
				astr)
		}
		addrAct[astr] = true
		cnt++
	}

	if cnt != 256 {
		t.Errorf("Expected 254 addresses to be generated, but we got %d",
			cnt)
	}

	for addr := range addrAct {
		if !addrExp[addr] {
			t.Errorf("We got an unexpected address %s", addr)
		}
	}

	for addr := range addrExp {
		if !addrAct[addr] {
			t.Errorf("We expected to see %s, but didn't.", addr)
		}
	}
}
