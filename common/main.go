// /home/krylon/go/src/github.com/blicero/carebear/common/main.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-03 18:17:23 krylon>

package main

import (
	"fmt"

	"github.com/blicero/carebear/common"
)

func main() {
	fmt.Printf("%s %s - %s\n",
		common.AppName,
		common.Version,
		common.BuildStamp.Format(common.TimestampFormat))
}
