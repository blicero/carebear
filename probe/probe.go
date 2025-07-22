// /home/krylon/go/src/github.com/blicero/carebear/probe/probe.go
// -*- mode: go; coding: utf-8; -*-
// Created on 21. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-22 20:20:38 krylon>

// Package probe implements probing Devices to determine what OS they run.
package probe

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/blicero/carebear/common"
	"github.com/blicero/carebear/database"
	"github.com/blicero/carebear/logdomain"
	"github.com/blicero/carebear/model"
	"golang.org/x/crypto/ssh"
)

const (
	osReleaseCmd = "/bin/cat /etc/os-release"
	unameCmd     = "/usr/bin/uname -s"
)

// Probe attempts to query Devices for the OS they are running.
type Probe struct {
	log  *log.Logger
	db   *database.Database
	lock sync.RWMutex // nolint: unused
	cfg  *ssh.ClientConfig
}

// New creates a new Probe.
func New(userName string, keyPath ...string) (*Probe, error) {
	var (
		err error
		p   = new(Probe)
	)

	if p.log, err = common.GetLogger(logdomain.Probe); err != nil {
		return nil, err
	} else if p.db, err = database.Open(common.DbPath); err != nil {
		p.log.Printf("[ERROR] Failed to open database: %s\n",
			err.Error())
		return nil, err
	} else if err = p.initConfig(userName, keyPath...); err != nil {
		return nil, err
	}

	return p, nil
} // func New(keyPath string) (*Probe, error)

func (p *Probe) initConfig(userName string, keyPath ...string) error {
	var (
		err    error
		keyRaw []byte
		signer ssh.Signer
		keys   = make([]ssh.Signer, 0, len(keyPath))
		// hostKey ssh.PublicKey
	)

	for _, path := range keyPath {
		p.log.Printf("[DEBUG] Trying to import %s\n", path)
		if keyRaw, err = os.ReadFile(path); err != nil {
			var ex = fmt.Errorf("Failed to read SSH key from %s: %w",
				keyPath,
				err)
			p.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		} else if signer, err = ssh.ParsePrivateKey(keyRaw); err != nil {
			var ex = fmt.Errorf("Failed to parse SSH key: %w",
				err)
			p.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		} else if signer == nil {
			var ex = fmt.Errorf("ParsePrivateKey did not return an error, but signer is nil!\nKey File: %s\nKey: %s",
				keyPath,
				keyRaw)
			p.log.Printf("[ERROR] %s\n",
				ex.Error())
			return ex
		}
		keys = append(keys, signer)
	}

	p.cfg = &ssh.ClientConfig{
		User: userName,
		// Auth: make([]ssh.AuthMethod, 0, len(keys)),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(keys...),
		},
		// HostKeyCallback: ssh.FixedHostKey(hostKey),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// for _, key := range keys {
	// 	p.log.Printf("[TRACE] Adding key to Authentication methods: %#v\n",
	// 		key)
	// 	p.cfg.Auth = append(p.cfg.Auth, ssh.PublicKeys(key))
	// }

	// slices.Reverse(p.cfg.Auth)
	// p.log.Printf("[DEBUG] %#v\n", p.cfg.Auth)
	return nil
} // func (p *Probe) initConfig(keyPath string) error

// QueryOS attempts to find out what operating system the device runs.
func (p *Probe) QueryOS(d *model.Device, port int) (string, error) {
	var (
		err     error
		client  *ssh.Client
		session *ssh.Session
	)

	for _, a := range d.Addr {
		var addr = fmt.Sprintf("%s:%d",
			a,
			port)
		if client, err = ssh.Dial("tcp", addr, p.cfg); err != nil {
			p.log.Printf("[ERROR] Failed to connect to %s at %s: %s\n",
				d.Name,
				a,
				err.Error())

		} else {
			break
		}
	}

	if client == nil {
		p.log.Printf("[ERROR] Could not connect to %s on any address.\n",
			d.Name)
		return "", err
	}

	defer client.Close()

	if session, err = client.NewSession(); err != nil {
		var ex = fmt.Errorf("Failed to create SSH session with %s: %w",
			d.Name,
			err)
		p.log.Printf("[ERROR] %s\n", ex.Error())
		return "", ex
	}

	defer session.Close()

	var rawOutput []byte

	if rawOutput, err = session.CombinedOutput(unameCmd); err != nil {
		var ex = fmt.Errorf("Failed to run %q on %s: %w",
			unameCmd,
			d.Name,
			err)
		p.log.Printf("[ERROR] %s\n", ex.Error())
		return "", ex
	}

	rawOutput = bytes.Trim(rawOutput, "\n\t ")

	var kernel = string(rawOutput)

	if kernel != "Linux" {
		return kernel, nil
	} else if session, err = client.NewSession(); err != nil {
		var ex = fmt.Errorf("Failed to create SSH session on %s: %w",
			d.Name,
			err)
		p.log.Printf("[ERROR] %s\n", ex.Error())
	}

	defer session.Close()

	if rawOutput, err = session.CombinedOutput(osReleaseCmd); err != nil {
		var ex = fmt.Errorf("Failed to cat(1) /etc/os-release on %s: %w",
			d.Name,
			err)
		p.log.Printf("[ERROR] %s\n", ex.Error())
		return "", ex
	}

	var (
		osname      string
		releaseInfo = string(rawOutput)
	)

	for l := range strings.Lines(releaseInfo) {
		if strings.HasPrefix(l, "NAME=") {
			osname = strings.Trim(l[5:], "\"\n\t ")
		}
	}

	return osname, nil
} // func (p *Probe) QueryOS(d *model.Device) (string, error)
