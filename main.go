// /home/krylon/go/src/github.com/blicero/carebear/common/main.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-09-05 18:30:31 krylon>

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/blicero/carebear/common"
	"github.com/blicero/carebear/database"
	"github.com/blicero/carebear/scanner"
	"github.com/blicero/carebear/scheduler"
	"github.com/blicero/carebear/settings"
	"github.com/blicero/carebear/web"
)

func main() {
	fmt.Printf("%s %s - %s\n",
		common.AppName,
		common.Version,
		common.BuildStamp.Format(common.TimestampFormat))

	var (
		err      error
		addr     string
		username string
		cfgPath  string
		sigQ     chan os.Signal
		port     int
		srv      *web.Server
		scan     *scanner.NetworkScanner
		sched    *scheduler.Scheduler
	)

	flag.StringVar(&addr, "addr", "", "Address of the web interface")
	flag.StringVar(
		&username,
		"user",
		os.Getenv("USER"),
		"The username for probing a remote host")
	flag.StringVar(&cfgPath, "cfg", common.CfgPath, "Path to the configuration file")
	flag.IntVar(
		&port,
		"port",
		22,
		"TCP port to connect to when probing")

	flag.Parse()

	common.InitApp()

	if settings.Settings, err = settings.Parse(cfgPath); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Failed to read configuration file %s: %s\n",
			cfgPath,
			err.Error(),
		)
		os.Exit(1)
	} else if err = database.InitPool(int(settings.Settings.PoolSize)); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Failed to initialize global database connection pool: %s\n",
			err.Error())
		os.Exit(1)
	} else if scan, err = scanner.NewNetworkScanner(); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Failed to create NetworkScanner: %s\n",
			err.Error())
		os.Exit(1)
	}

	if addr == "" {
		addr = fmt.Sprintf("[::1]:%d", common.DefaultPort)
	}

	if sched, err = scheduler.Create(scan); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Error creating Scheduler: %s\n",
			err.Error())
		os.Exit(1)
	} else if srv, err = web.Create(addr, scan, sched); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Error creating web interface on %s: %s\n",
			addr,
			err.Error())
		os.Exit(1)
	}

	go srv.Run()
	go sched.Start()

	// ...

	sigQ = make(chan os.Signal)
	signal.Notify(sigQ, os.Interrupt, syscall.SIGTERM)

	for {
		var sig os.Signal

		sig = <-sigQ

		fmt.Fprintf(
			os.Stderr,
			"Caught Signal %s\n",
			sig)
		os.Exit(0)
	}

} // func main()

func findKeyFiles() ([]string, error) {
	var (
		err   error
		dh    *os.File
		path  string
		names []string
		files = make([]string, 0, 8)
	)

	path = filepath.Join(
		os.Getenv("HOME"),
		".ssh")

	if dh, err = os.Open(path); err != nil {
		return nil, err
	}

	defer dh.Close()

	if names, err = dh.Readdirnames(-1); err != nil {
		return nil, err
	}

	for _, file := range names {
		if strings.HasPrefix(file, "id_") && !strings.HasSuffix(file, ".pub") {
			files = append(
				files,
				filepath.Join(path, file))
		}
	}

	return files, nil
} // func findKeyFiles() ([]string, error)
