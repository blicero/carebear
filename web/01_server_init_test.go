// /home/krylon/go/src/github.com/blicero/carebear/web/01_server_init_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 25. 08. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2025-07-14 16:28:16 krylon>

package web

import (
	"fmt"
	"testing"
	"time"
)

func TestServerCreate(t *testing.T) {
	var err error

	addr = fmt.Sprintf("[::1]:%d", testPort)

	if srv, err = Create(addr); err != nil {
		srv = nil
		t.Fatalf("Error creating Server: %s",
			err.Error())
	}

	go srv.Run()
	time.Sleep(time.Second)
} // func TestServerCreate(t *testing.T)
