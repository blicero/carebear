// /home/krylon/go/src/github.com/blicero/carebear/web/msgbuf.go
// -*- mode: go; coding: utf-8; -*-
// Created on 14. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-14 16:25:04 krylon>

package web

import (
	"crypto/sha512"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/blicero/carebear/common"
	"github.com/hashicorp/logutils"
)

type message struct { // nolint: unused
	Timestamp time.Time
	Level     logutils.LogLevel
	Message   string
}

func (m *message) TimeString() string { // nolint: unused
	return m.Timestamp.Format(common.TimestampFormat)
} // func (m *Message) TimeString() string

func (m *message) Checksum() string { // nolint: unused
	var str = m.Timestamp.Format(common.TimestampFormat) + "##" +
		string(m.Level) + "##" +
		m.Message

	var hash = sha512.New()
	hash.Write([]byte(str)) // nolint: gosec,errcheck

	var cksum = hash.Sum(nil)
	var ckstr = fmt.Sprintf("%x", cksum)

	return ckstr
} // func (m *message) Checksum() string

type msgLink struct {
	msg  *message
	next *msgLink
}

type msgBuf struct {
	lock sync.RWMutex
	cnt  int
	link *msgLink
}

func newMsgBuf() *msgBuf {
	var buf = new(msgBuf)

	return buf
} // func newMsgBuf() *msgBuf

// Size returns the number of messages in the buffer.
func (mb *msgBuf) Size() int {
	mb.lock.RLock()
	var n = mb.cnt
	mb.lock.RUnlock()
	return n
}

func (mb *msgBuf) put(m *message) {
	mb.lock.Lock()

	var lnk = &msgLink{
		msg:  m,
		next: mb.link,
	}

	mb.link = lnk
	mb.cnt++

	mb.lock.Unlock()
}

func (mb *msgBuf) getAll() []*message {
	mb.lock.Lock()
	defer mb.lock.Unlock()

	var (
		list = make([]*message, mb.cnt)
		idx  = 0
		lnk  = mb.link
	)

	for lnk != nil {
		list[idx] = lnk.msg
		mb.cnt--
		idx++
		lnk = lnk.next
	}

	mb.link = nil
	slices.Reverse[[]*message](list)
	return list
} // func (mb *msgBuf) getAll() []*message
