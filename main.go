// /home/krylon/go/src/github.com/blicero/carebear/common/main.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-22 15:25:29 krylon>

package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/blicero/carebear/common"
	"github.com/blicero/carebear/model"
	"github.com/blicero/carebear/probe"
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
		mode     string
		username string
		keyfile  string
		port     int
		srv      *web.Server
		p        *probe.Probe
	)

	flag.StringVar(&addr, "addr", "", "Address of the web interface")
	flag.StringVar(
		&mode,
		"mode",
		"server",
		"What mode to run in (scanner, server, probe)")
	flag.StringVar(
		&username,
		"user",
		os.Getenv("USER"),
		"The username for probing a remote host")
	flag.StringVar(
		&keyfile,
		"key",
		"",
		"Private SSH key to use for probing")
	flag.IntVar(
		&port,
		"port",
		22,
		"TCP port to connect to when probing")

	flag.Parse()

	common.InitApp()

	switch strings.ToLower(mode) {
	case "server":
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
		}

		srv.Run()
	case "scanner":
		fmt.Println("Scanner is not ready, yet.")
		os.Exit(0)
	case "probe":
		const addrStr = "51.195.118.34"
		var (
			osname string
			dev    = &model.Device{
				ID:    42,
				NetID: 23,
				Name:  "blicero",
				Addr: []net.Addr{
					&net.IPAddr{
						IP: net.ParseIP(addrStr),
					},
				},
				BigHead: true,
			}
		)

		if p, err = probe.New(username, keyfile); err != nil {
			fmt.Fprintf(
				os.Stderr,
				"Failed to create Probe: %s\n",
				err.Error(),
			)
			os.Exit(1)
		} else if osname, err = p.QueryOS(dev, port); err != nil {
			fmt.Fprintf(
				os.Stderr,
				"Failed to query OS running on %s: %s\n",
				dev.Name,
				err.Error())
			os.Exit(1)
		}

		fmt.Printf("%s is running %s\n",
			dev.Name,
			osname)
	}

} // func main()
