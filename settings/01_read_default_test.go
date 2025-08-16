// /home/krylon/go/src/github.com/blicero/carebear/settings/01_read_default_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 31. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-08-16 19:55:30 krylon>

package settings

import (
	"os"
	"testing"
	"time"
)

func TestReadDefault(t *testing.T) {
	var (
		err  error
		path string
		cfg  *Options
	)

	const (
		liveTimeout = time.Second * 600
		webPort     = 3819
	)

	path = time.Now().Format("/tmp/carebear_test_cfg_20060102_150405.toml")

	defer os.Remove(path) // nolint: errcheck

	if cfg, err = Parse(path); err != nil {
		t.Fatalf("Error Parsing configuration file: %s",
			err.Error())
	} else if cfg == nil {
		t.Fatalf("Parse did not return an error, but no Settings, either")
	}

	if cfg.WebPort != webPort {
		t.Errorf("Unexpected WebPort %d (expect %d)",
			cfg.WebPort,
			webPort)
	}

	if cfg.LiveTimeout != liveTimeout {
		t.Errorf("Unexpected LiveTimeout: %s (expect %s)",
			cfg.LiveTimeout,
			liveTimeout)
	}
} // func TestReadDefault(t *testing.T)
