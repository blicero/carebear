// /home/krylon/go/src/github.com/blicero/carebear/database/01_database_create_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2025-07-14 14:34:54 krylon>

package database

import (
	"database/sql"
	"testing"

	"github.com/blicero/carebear/common"
)

var tdb *Database

func TestDBCreate(t *testing.T) {
	var err error

	if tdb, err = Open(common.DbPath); err != nil {
		tdb = nil
		t.Fatalf("Cannot create database: %s",
			err.Error())
	}
} // func TestDBCreate(t *testing.T)

func TestDBQueryPrepare(t *testing.T) {
	var (
		err error
		q   *sql.Stmt
	)

	if tdb == nil {
		t.SkipNow()
	}

	for k, s := range qdb {
		if q, err = tdb.getQuery(k); err != nil {
			t.Errorf("Error preparing query %s: %s\n%s\n",
				k,
				err.Error(),
				s)
		} else if q == nil {
			t.Errorf("Query handle %s is nil!", k)
		}
	}
} // func TestDBQueryPrepare(t *testing.T)
