// /home/krylon/go/src/carebear/web/tmpl_data.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 05. 2020 by Benjamin Walkenhorst
// (c) 2020 Benjamin Walkenhorst
// Time-stamp: <2025-07-14 15:39:07 krylon>
//
// This file contains data structures to be passed to HTML templates.

package web

type tmplDataBase struct { // nolint: unused
	Title      string
	Messages   []*message
	Debug      bool
	TestMsgGen bool
	URL        string
}

type tmplDataIndex struct { // nolint: unused,deadcode
	tmplDataBase
}

// Local Variables:  //
// compile-command: "go generate && go vet && go build -v -p 16 && gometalinter && go test -v" //
// End: //
