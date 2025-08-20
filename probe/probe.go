// /home/krylon/go/src/github.com/blicero/carebear/probe/probe.go
// -*- mode: go; coding: utf-8; -*-
// Created on 21. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-08-20 17:24:45 krylon>

// Package probe implements probing Devices to determine what OS they run.
package probe

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/blicero/carebear/common"
	"github.com/blicero/carebear/database"
	"github.com/blicero/carebear/logdomain"
	"github.com/blicero/carebear/model"
	"github.com/blicero/carebear/ping"
	"golang.org/x/crypto/ssh"
)

// ErrPingOffline indicates a Device did not respond to a ping.
var ErrPingOffline = errors.New("Device did not respond to ping")

const (
	osReleaseCmd = "/bin/cat /etc/os-release"
	unameCmd     = "/usr/bin/uname -s"
)

// Probe attempts to query Devices for the OS they are running.
type Probe struct {
	log     *log.Logger
	db      *database.Database
	lock    sync.RWMutex // nolint: unused
	cfg     *ssh.ClientConfig
	pp      *ping.Pinger
	clients map[int64]*ssh.Client
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
	} else if p.pp, err = ping.Create(); err != nil {
		return nil, err
	}

	p.clients = make(map[int64]*ssh.Client)

	return p, nil
} // func New(keyPath string) (*Probe, error)

func (p *Probe) initConfig(userName string, keyPath ...string) error {
	var (
		err    error
		keyRaw []byte
		signer ssh.Signer
		keys   = make([]ssh.Signer, 0, len(keyPath))
	)

	for _, path := range keyPath {
		var (
			fh       *os.File
			keyFiles []string
		)

		p.log.Printf("[DEBUG] Trying to import %s\n", path)
		if fh, err = os.Open(path); err != nil {
			p.log.Printf("[ERROR] Cannot open %s: %s\n",
				path,
				err.Error())
			continue
		}

		defer fh.Close()

		if keyFiles, err = fh.Readdirnames(-1); err != nil {
			p.log.Printf("[ERROR] Cannot read files in directory %s: %s\n",
				path,
				err.Error())
			continue
		}

		for _, file := range keyFiles {
			if !strings.HasPrefix(file, "id_") || strings.HasSuffix(file, ".pub") {
				continue
			}

			var fullPath = filepath.Join(path, file)

			p.log.Printf("[DEBUG] Import SSH key %s\n", fullPath)

			if keyRaw, err = os.ReadFile(fullPath); err != nil {
				var ex = fmt.Errorf("Failed to read SSH key from %s: %w",
					fullPath,
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
					fullPath,
					keyRaw)
				p.log.Printf("[ERROR] %s\n",
					ex.Error())
				return ex
			}
			keys = append(keys, signer)
		}
	}

	// XXX The documentation for the ssh package says very explicitly to NOT use
	//     InsecureIgnoreHostKey in production code, which makes sense for obvious reasons.
	//     But I intend to only run this application on my local network, where I own and
	//     administer all the devices.
	//     But if anyone ever intends to use this code (or parts of it) for any other purpose,
	//     please, PLEASE rectify this!!! You have been warned.
	p.cfg = &ssh.ClientConfig{
		User: userName,
		// Auth: make([]ssh.AuthMethod, 0, len(keys)),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(keys...),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	return nil
} // func (p *Probe) initConfig(keyPath string) error

func (p *Probe) connect(d *model.Device, port int) (*ssh.Client, error) {
	var (
		err    error
		client *ssh.Client
	)

	for _, a := range d.Addr {
		if !p.pp.PingAddr(a.String()) {
			p.log.Printf("[INFO] Device %s did not respond to ping\n",
				d.Name)
			continue
		}

		var addr = fmt.Sprintf("%s:%d",
			a,
			port)
		if client, err = ssh.Dial("tcp", addr, p.cfg); err != nil {
			p.log.Printf("[ERROR] Failed to connect to %s at %s: %s\n",
				d.Name,
				a,
				err.Error())
		} else {
			return client, nil
		}
	}

	return nil, ErrPingOffline
} // func (p *Probe) connect(d *model.Device, port int) (*ssh.Client, error)

func (p *Probe) getClient(d *model.Device, port int) (*ssh.Client, error) {
	var (
		err error
		ok  bool
		c   *ssh.Client
	)

	p.lock.Lock()
	defer p.lock.Unlock()

	if c, ok = p.clients[d.ID]; ok {
		return c, nil
	} else if c, err = p.connect(d, port); err != nil {
		return nil, err
	} else if c == nil {
		var ex = fmt.Errorf("probe.connect did not return an error, but connection to %s is nil",
			d.Name)
		p.log.Printf("[ERROR] %s\n", ex.Error())
		return nil, ex
	}

	p.clients[d.ID] = c
	return c, nil
} // func (p *Probe) getClient(d *model.Device, port int) (*ssh.Client, error)

func (p *Probe) getSession(d *model.Device, port int) (s *ssh.Session, e error) {
	var (
		err    error
		client *ssh.Client
		sess   *ssh.Session
	)

	defer func() {
		if ex := recover(); ex != nil {
			p.log.Printf("[ERROR] Panic trying to get SSH session for %s: %s\n",
				d.Name,
				ex)
			s = nil
			e = ex.(error)
		}
	}()

	if client, err = p.getClient(d, port); err != nil {
		if err == ErrPingOffline {
			return nil, err
		}
		var ex = fmt.Errorf("Failed to get SSH client for %s: %w",
			d.Name,
			err)
		p.log.Printf("[ERROR] %s\n", ex.Error())
		return nil, ex
	} else if sess, err = client.NewSession(); err != nil {
		var ex = fmt.Errorf("Failed to create new SSH session for %s: %w",
			d.Name,
			err)
		p.log.Printf("[ERROR] %s\n", ex.Error())
		return nil, ex
	}

	return sess, nil
} // func (p *Probe) getSession(d *model.Device, port int) (*ssh.Session, error)

// QueryOS attempts to find out what operating system the device runs.
func (p *Probe) QueryOS(d *model.Device, port int) (string, error) {
	var (
		err     error
		client  *ssh.Client
		session *ssh.Session
	)

	if client, err = p.getClient(d, port); err != nil {
		return "", err
	} else if client == nil {
		p.log.Printf("[ERROR] Could not connect to %s on any address.\n",
			d.Name)
		return "", err
	}

	// defer client.Close()

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

	// If the kernel isn't Linux, it almost certainly is some kind of BSD, in which
	// case we have the information we want.
	//
	// If it is, we try to read /etc/os-release to determine what distro we
	// are dealing with.
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

//func (p *Probe)
