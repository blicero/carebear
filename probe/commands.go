// /home/krylon/go/src/github.com/blicero/carebear/probe/commands.go
// -*- mode: go; coding: utf-8; -*-
// Created on 23. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-26 16:40:02 krylon>

package probe

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/blicero/carebear/model"
	"golang.org/x/crypto/ssh"
)

// This should suffice for now, but in the long run, it might be nice to reuse the ssh.Client.

func (p *Probe) executeCommand(d *model.Device, port int, cmd string) ([]string, error) {
	var (
		err     error
		client  *ssh.Client
		session *ssh.Session
	)

	if client, err = p.connect(d, port); err != nil {
		p.log.Printf("Failed to connect to %s: %s\n",
			d.Name,
			err.Error())
	}

	defer client.Close()

	if session, err = client.NewSession(); err != nil {
		var ex = fmt.Errorf("Failed to create SSH session for %s: %w",
			d.Name,
			err)
		p.log.Printf("[ERROR] %s\n", ex.Error())
		return nil, ex
	}

	defer session.Close()

	var rawOutput []byte

	if rawOutput, err = session.CombinedOutput(cmd); err != nil {
		var ex = fmt.Errorf("Failed to execute command on %s: %w\n>>> Command: %s",
			d.Name,
			err,
			cmd)
		p.log.Printf("[ERROR] %s\n", ex.Error())
		return nil, ex
	}

	var lines = strings.Split(string(rawOutput), "\n")

	return lines, nil
} // func (p *Probe) executeCommand(d *model.Device, port int, cmd string) ([]string, error)

// QueryUpdatesDebian asks a Debian-ish system for a list of available updates.
func (p *Probe) QueryUpdatesDebian(d *model.Device, port int) ([]string, error) {
	const cmd = "/usr/bin/apt update && /usr/bin/apt list --upgradable"

	return p.executeCommand(d, port, cmd)
} // func (p *Probe) QueryUpdatesDebian(d *model.Device, port int) ([]string, error)

// QueryUpdatesSuse asks an openSuse system for a list of available updates.
func (p *Probe) QueryUpdatesSuse(d *model.Device, port int) ([]string, error) {
	const cmd = "zypper ref -f && zypper lu"
	return p.executeCommand(d, port, cmd)
} // func (p *Probe) QueryUpdatesSuse(d *model.Device, port int) ([]string, error)

// Sample output:
// 18:01:18  2 Tage  0:22 an,  2 Benutzer,  Durchschnittslast: 1,08, 0,98, 0,94
// 6:02PM  up 56 days,  5:16, 4 users, load averages: 0.00, 0.01, 0.00

var uptimePat = regexp.MustCompile(
	`(?msi:durchschnittslast|load averages?):\s+(\d+[,.]\d+),\s+(\d+[,.]\d+),\s+(\d+[,.]\d+)\s*$`)

// QueryLoadAvg attempts to extract the system load average from the given Device.
func (p *Probe) QueryLoadAvg(d *model.Device, port int) ([3]float64, error) {
	const cmd = "/usr/bin/uptime"
	var (
		err   error
		res   []string
		load  [3]float64
		match []string
	)

	if res, err = p.executeCommand(d, port, cmd); err != nil {
		var ex = fmt.Errorf("Failed to query uptime/loadavg on %s: %w",
			d.Name,
			err)
		p.log.Printf("[ERROR] %s\n", ex.Error())
		return load, ex
	} else if match = uptimePat.FindStringSubmatch(res[0]); match == nil {
		var ex = fmt.Errorf("Cannot parse the output of uptime(1) from %s: %q",
			d.Name,
			res[0])
		p.log.Printf("[ERROR] %s\n", ex.Error())
		return load, ex
	} else if len(match) > 0 {
		for idx, val := range match[1:] {
			var (
				s = strings.ReplaceAll(val, ",", ".")
				f float64
			)

			if f, err = strconv.ParseFloat(s, 64); err != nil {
				var ex = fmt.Errorf("Cannot parse load avg %q: %w",
					s,
					err)
				p.log.Printf("[ERROR] %s\n", ex.Error())
				return load, ex
			}

			load[idx] = f
		}
	}

	// ...

	return load, nil
} // func (p *Probe) QueryLoadAvg(d *model.Device, port int) ([3]float64, error)
