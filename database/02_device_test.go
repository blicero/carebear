// /home/krylon/go/src/github.com/blicero/carebear/database/02_device_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 07. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-07 15:51:27 krylon>

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
	if tdb == nil {
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
			Name: fmt.Sprintf("dev%02d", n),
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
