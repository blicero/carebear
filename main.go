// /home/krylon/go/src/github.com/blicero/carebear/common/main.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-15 17:17:44 krylon>

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/blicero/carebear/common"
	"github.com/blicero/carebear/web"
)

func main() {
	fmt.Printf("%s %s - %s\n",
		common.AppName,
		common.Version,
		common.BuildStamp.Format(common.TimestampFormat))

	var (
		err  error
		addr string
		srv  *web.Server
	)

	flag.StringVar(&addr, "addr", "", "Address of the web interface")

	flag.Parse()

	common.InitApp()

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

} // func main()
