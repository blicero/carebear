// /home/krylon/go/src/github.com/blicero/carebear/database/02_device_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 07. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-08 19:40:34 krylon>

package database

import (
	"fmt"
	"net"
	"testing"

	"github.com/blicero/carebear/model"
)

const (
	devCnt = 16
)

var tdev []*model.Device

func TestDeviceAdd(t *testing.T) {
	if tdb == nil || tnet == nil || tnet.ID == 0 {
		t.SkipNow()
	}

	var (
		err    error
		status = false
		n      int
	)

	tdev = make([]*model.Device, devCnt)

	tdb.Begin() // nolint: errcheck
	defer func() {
		if status {
			tdb.Commit() // nolint: errcheck
		} else {
			t.Log("Rolling back database transaction.")
			tdb.Rollback() // nolint: errcheck
		}
	}()

	for n = 1; n <= devCnt; n++ {
		var dev = &model.Device{
			Name:  fmt.Sprintf("dev%02d", n),
			NetID: tnet.ID,
			Addr: []net.Addr{
				&net.IPAddr{
					IP: net.IPv4(192, 168, 0, byte(n)),
				},
			},
			BigHead: true,
		}

		if err = tdb.DeviceAdd(dev); err != nil {
			t.Fatalf("Cannot add Device %s: %s",
				dev.Name,
				err.Error())
		} else {
			tdev[n-1] = dev
		}
	}

	status = true
} // func TestDeviceAdd(t *testing.T)

func TestDeviceGetall(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	var (
		err  error
		xdev []*model.Device
	)

	if xdev, err = tdb.DeviceGetAll(); err != nil {
		t.Fatalf("Failed to load all Devices: %s",
			err.Error())
	} else if xdev == nil {
		t.Fatal("DeviceGetAll returned nil")
	} else if len(xdev) != devCnt {
		t.Fatalf("DeviceGetAll returned %d Devices, we expected to get %d",
			len(xdev),
			devCnt)
	}
} // func TestDeviceGetall(t *testing.T)

func TestDeviceGetByID(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	var (
		err  error
		xdev *model.Device
	)

	for _, dev := range tdev {
		if xdev, err = tdb.DeviceGetByID(dev.ID); err != nil {
			t.Fatalf("DeviceGetByID failed: %s", err.Error())
		} else if xdev == nil {
			t.Fatalf("DeviceGetByID return nil for Device with ID %d (%s)",
				dev.ID,
				dev.Name)
		}

		var addr01, addr02 string

		addr01 = dev.AddrStr()
		addr02 = xdev.AddrStr()

		if addr01 != addr02 {
			t.Fatalf("Unexpected address(es) for Device %d (%s):\nExpected:\t%s\nGot:\t%s\n",
				dev.ID,
				dev.Name,
				addr01,
				addr02)
		}
	}

	var i int64

	for i = 1000; i < 2000; i++ {
		var dev *model.Device

		if dev, err = tdb.DeviceGetByID(i); err != nil {
			t.Fatalf("Failed to look up Device %d: %s",
				i,
				err.Error())
		} else if dev != nil {
			t.Fatalf("Looking for Device %d should not have returned a value: %#v",
				i,
				dev)
		}
	}
} // func TestDeviceGetByID(t *testing.T)

func TestDeviceGetByName(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	var (
		err  error
		xdev *model.Device
	)

	for _, dev := range tdev {
		if xdev, err = tdb.DeviceGetByName(dev.Name); err != nil {
			t.Fatalf("DeviceGetByID failed: %s", err.Error())
		} else if xdev == nil {
			t.Fatalf("DeviceGetByID return nil for Device with ID %d (%s)",
				dev.ID,
				dev.Name)
		}

		var addr01, addr02 string

		addr01 = dev.AddrStr()
		addr02 = xdev.AddrStr()

		if addr01 != addr02 {
			t.Fatalf("Unexpected address(es) for Device %d (%s):\nExpected:\t%s\nGot:\t%s\n",
				dev.ID,
				dev.Name,
				addr01,
				addr02)
		}
	}
} // func TestDeviceGetByID(t *testing.T)
