// /home/krylon/go/src/github.com/blicero/carebear/common/main.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-12 19:04:11 krylon>

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/blicero/carebear/common"
	"github.com/blicero/carebear/database"
	"github.com/blicero/carebear/model"
	"github.com/blicero/carebear/scanner"
	"github.com/blicero/carebear/scanner/command"
)

func main() {
	fmt.Printf("%s %s - %s\n",
		common.AppName,
		common.Version,
		common.BuildStamp.Format(common.TimestampFormat))

	var (
		err           error
		db            *database.Database
		sc            *scanner.Scanner
		addr          string
		mynet         *model.Network
		scanWorkerCnt int64
	)

	flag.StringVar(&addr, "addr", "", "CIDR address of the network to scan")
	flag.Int64Var(&scanWorkerCnt, "wcnt", 32, "Number of worker goroutines for network scans")

	flag.Parse()

	common.InitApp()

	if db, err = database.Open(common.DbPath); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Failed to open Database at %s: %s\n",
			common.DbPath,
			err.Error())
		os.Exit(1)
	} else if mynet, err = model.NewNetwork(addr, "Bla bla bla"); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Invalid network address %q: %s\n",
			addr,
			err.Error())
		os.Exit(1)
	} else if err = db.NetworkAdd(mynet); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Failed to add network to database: %s\n",
			err.Error())
	} else if sc, err = scanner.New(scanWorkerCnt); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Failed to create Scanner: %s\n",
			err.Error())
		os.Exit(1)
	}

	sc.Start()

	sc.CmdQ <- command.ScanStart

	time.Sleep(time.Second)

	for n := sc.ScanCnt(); n > 0; n = sc.ScanCnt() {
		time.Sleep(time.Second * 5)
		scanned, added, ok := sc.ScanProgress(mynet.ID)
		if !ok {
			fmt.Printf("Scan of Network %s appears to be over.\n",
				mynet.Addr)
			break
		}
		fmt.Printf("IP Addresses scanned: %4d / Devices added: %3d\n",
			scanned,
			added)
	}

	fmt.Println("Scan is done.")
} // func main()
