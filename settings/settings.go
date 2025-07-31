// /home/krylon/go/src/github.com/blicero/carebear/settings/settings.go
// -*- mode: go; coding: utf-8; -*-
// Created on 31. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-31 18:30:15 krylon>

// Package settings deals with the configuration file. Duh.
package settings

import (
	"fmt"
	"os"
	"time"

	"github.com/blicero/carebear/common"
	"github.com/blicero/krylib"
	"github.com/pelletier/go-toml"
)

const defaultConfig = `
# Time-stamp: <>
[Global]
Debug = true

[Web]
Port = 3819

[Scanner]
IntervalNet = 300
IntervalDev = 60
Workers = 32

[Device]
LiveTimeout = 300
`

// Settings defines several configurable parameters used throughout the application.
type Settings struct {
	WebPort         int64
	LiveTimeout     time.Duration
	ScanIntervalNet time.Duration
	ScanIntervalDev time.Duration
	ScanWorkerCount int64
	Debug           bool
}

// Parse reads the configuration file at the given path.
// If path is an empty string, it uses the global default path.
func Parse(path string) (*Settings, error) {
	if path == "" {
		path = common.CfgPath
	}

	var (
		err  error
		ok   bool
		cfg  *Settings
		tree *toml.Tree
	)

	if ok, err = krylib.Fexists(path); err != nil {
		return nil, err
	} else if !ok {
		if err = createDefaultConfig(path); err != nil {
			return nil, err
		}
	}

	if tree, err = toml.LoadFile(path); err != nil {
		return nil, err
	}

	cfg = new(Settings)

	cfg.WebPort = tree.Get("Web.Port").(int64)
	cfg.LiveTimeout = time.Duration(tree.Get("Device.LiveTimeout").(int64)) * time.Second
	cfg.ScanIntervalNet = time.Duration(tree.Get("Scanner.IntervalNet").(int64)) * time.Second
	cfg.ScanIntervalDev = time.Duration(tree.Get("Scanner.IntervalDev").(int64)) * time.Second
	cfg.ScanWorkerCount = tree.Get("Scanner.Workers").(int64)
	cfg.Debug = tree.Get("Global.Debug").(bool)

	return cfg, nil
} // func Parse(path string) (*Settings, error)

func createDefaultConfig(path string) error {
	var (
		err     error
		written int
		fh      *os.File
	)

	if fh, err = os.Create(path); err != nil {
		return err
	}

	defer fh.Close()

	if written, err = fh.WriteString(defaultConfig); err != nil {
		return err
	} else if written != len(defaultConfig) {
		err = fmt.Errorf("Unexpected number of bytes written to config file: %d (expected %d)",
			written,
			len(defaultConfig))
		return err
	}

	return nil
} // func createDefaultConfig(path string) error
