// /home/krylon/go/src/github.com/blicero/carebear/common/main.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-08-04 18:23:44 krylon>

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
	}

	if addr == "" {
		addr = fmt.Sprintf("[::1]:%d", common.DefaultPort)
	}

	if srv, err = web.Create(addr); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Error creating web interface on %s: %s\n",
			addr,
			err.Error())
		os.Exit(1)
	} else if sched, err = scheduler.Create(); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Error creating Scheduler: %s\n",
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

	// switch strings.ToLower(mode) {
	// case "server":
	// 	if addr == "" {
	// 		addr = fmt.Sprintf("[::1]:%d", common.DefaultPort)
	// 	}

	// 	if srv, err = web.Create(addr); err != nil {
	// 		fmt.Fprintf(
	// 			os.Stderr,
	// 			"Error creating web interface on %s: %s\n",
	// 			addr,
	// 			err.Error())
	// 		os.Exit(1)
	// 	}

	// 	srv.Run()
	// case "scanner":
	// 	fmt.Println("Scanner is not ready, yet.")
	// 	os.Exit(0)
	// case "probe":
	// 	var (
	// 		osname   string
	// 		keyFiles []string
	// 		dev      = &model.Device{
	// 			ID:    42,
	// 			NetID: 23,
	// 			Name:  addr,
	// 			Addr: []net.Addr{
	// 				&net.IPAddr{
	// 					IP: net.ParseIP(addr),
	// 				},
	// 			},
	// 			BigHead: true,
	// 		}
	// 	)

	// 	if keyFiles, err = findKeyFiles(); err != nil {
	// 		fmt.Fprintf(
	// 			os.Stderr,
	// 			"Failed to find SSH keys: %s\n",
	// 			err.Error())
	// 		os.Exit(1)
	// 	}

	// 	fmt.Printf("Using the following keys for authentication:\n%s\n",
	// 		strings.Join(keyFiles, "\n"))

	// 	if p, err = probe.New(username, keyFiles...); err != nil {
	// 		fmt.Fprintf(
	// 			os.Stderr,
	// 			"Failed to create Probe: %s\n",
	// 			err.Error(),
	// 		)
	// 		os.Exit(1)
	// 	} else if osname, err = p.QueryOS(dev, port); err != nil {
	// 		fmt.Fprintf(
	// 			os.Stderr,
	// 			"Failed to query OS running on %s: %s\n",
	// 			dev.Name,
	// 			err.Error())
	// 		os.Exit(1)
	// 	}

	// 	fmt.Printf("%s is running %s\n",
	// 		dev.Name,
	// 		osname)
	// }

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
