// /home/krylon/go/src/github.com/blicero/carebear/probe/commands.go
// -*- mode: go; coding: utf-8; -*-
// Created on 23. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-09-06 15:16:44 krylon>

package probe

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/blicero/carebear/model"
	"golang.org/x/crypto/ssh"
)

// This should suffice for now, but in the long run, it might be nice to reuse the ssh.Client

const pkgSep = "\t"

func (p *Probe) executeCommand(d *model.Device, port int, cmd string) ([]string, error) {
	var (
		err     error
		session *ssh.Session
	)

	// 05. 08. 2025
	// I get a panic originating in NewSession when connecting to a Device that is offline.
	if session, err = p.getSession(d, port); err != nil {
		if err == ErrPingOffline {
			return nil, err
		}
		var ex = fmt.Errorf("Failed to create SSH session for %s: %w",
			d.Name,
			err)
		p.log.Printf("[ERROR] %s\n", ex.Error())
		return nil, ex
	}

	defer session.Close()

	var rawOutput []byte

	if rawOutput, err = session.CombinedOutput(cmd); err != nil {
		if strings.Contains(cmd, "dnf") && strings.HasPrefix(err.Error(), "Process exited with status 100") {
			// dnf check-upgrade exits with status 100 if there are updates available.
		} else if strings.Contains(cmd, "checkupdates") && strings.HasPrefix(err.Error(), "Process exited with status 2") {
			// checkupdates on Arch exits with status 2 if no updates are available.
			return nil, nil
		} else if d.OS == "FreeBSD" && strings.HasPrefix(err.Error(), "Process exited with status 2") {
			// On FreeBSD, "freebsd-update updatesready" exits with status 2 if no updates are available.
			return nil, nil
		} else {
			var ex = fmt.Errorf("Failed to execute command on %s: %w\n>>> Command: %s",
				d.Name,
				err,
				cmd)
			p.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}
	}

	var lines = strings.Split(string(rawOutput), "\n")

	return lines, nil
} // func (p *Probe) executeCommand(d *model.Device, port int, cmd string) ([]string, error)

var patUpdateDebian = regexp.MustCompile(`^([^/]+)/(\S+)\s+(\S+)\s+(\S+)`)

// QueryUpdatesDebian asks a Debian-ish system for a list of available updates.
func (p *Probe) QueryUpdatesDebian(d *model.Device, port int) ([]string, error) {
	const cmd = "/usr/bin/apt list --upgradable"
	var (
		err     error
		output  []string
		match   []string
		updates []string
	)

	if output, err = p.executeCommand(d, port, cmd); err != nil {
		if err == ErrPingOffline {
			return nil, err
		}
		_ = p.disconnect(d)
		p.log.Printf("[ERROR] Failed to execute command %q on %s: %s\n",
			cmd,
			d.Name,
			err.Error())
		return nil, err
	}

	updates = make([]string, 0)

	for _, l := range output {
		if match = patUpdateDebian.FindStringSubmatch(l); len(match) > 0 {
			var upd = strings.Join(match[1:], pkgSep)
			updates = append(updates, upd)
		}
	}

	return updates, nil
} // func (p *Probe) QueryUpdatesDebian(d *model.Device, port int) ([]string, error)

var patUpdateSuse = regexp.MustCompile(`\s+\|\s+`)

// QueryUpdatesSuse asks an openSuse system for a list of available updates.
func (p *Probe) QueryUpdatesSuse(d *model.Device, port int) ([]string, error) {
	const cmd = "zypper lu"
	var (
		err     error
		output  []string
		updates []string
	)

	if output, err = p.executeCommand(d, port, cmd); err != nil {
		if err == ErrPingOffline {
			return nil, err
		}
		_ = p.disconnect(d)
		p.log.Printf("[ERROR] Failed to execute command %q on %s: %s\n",
			cmd,
			d.Name,
			err.Error())
		return nil, err
	}

	updates = make([]string, 0)

	for _, l := range output[4:] {
		l = strings.Trim(l, " \t\n")
		var pieces = patUpdateSuse.Split(l, -1)
		if len(pieces) > 0 {
			var upd = strings.Join(pieces[1:], pkgSep)
			updates = append(updates, upd)
		}
	}

	return updates, nil
} // func (p *Probe) QueryUpdatesSuse(d *model.Device, port int) ([]string, error)

var patUpdateDNF = regexp.MustCompile(`\s+`)

// QueryUpdatesFedora asks a Fedora system for a list of available updates.
// Or any other system based the dnf package manager.
func (p *Probe) QueryUpdatesFedora(d *model.Device, port int) ([]string, error) {
	const cmd = "env DNF5_FORCE_INTERACTIVE=0 dnf check-upgrade"
	var (
		err             error
		output, updates []string
	)

	if output, err = p.executeCommand(d, port, cmd); err != nil {
		if err == ErrPingOffline {
			return nil, err
		}
		_ = p.disconnect(d)
		return nil, err
	}

	updates = make([]string, 0)

	for _, l := range output {
		var pieces = patUpdateDNF.Split(l, -1)
		if len(pieces) == 3 {
			var upd = strings.Join(pieces, pkgSep)
			updates = append(updates, upd)
		}
	}

	return updates, nil
} // func (p *Probe) QueryUpdatesFedora(d *model.Device, port int) ([]string, error)

var patUpdateArch = regexp.MustCompile(`^(\S+)\s+(\S+)\s+->\s+(\S+)$`)

// QueryUpdatesArch asks an Arch Linux system for a list of pending updates.
func (p *Probe) QueryUpdatesArch(d *model.Device, port int) ([]string, error) {
	const cmd = "checkupdates"
	var (
		err             error
		output, updates []string
	)

	if output, err = p.executeCommand(d, port, cmd); err != nil {
		if err == ErrPingOffline {
			return nil, err
		}
		_ = p.disconnect(d)
		return nil, err
	}

	updates = make([]string, 0)

	for _, l := range output {
		var match []string
		if match = patUpdateArch.FindStringSubmatch(l); len(match) > 0 {
			var upd = strings.Join(match[1:], pkgSep)
			updates = append(updates, upd)
		}
	}

	return updates, nil
} // func (p *Probe) QueryUpdatesArch(d *model.Device, port int) ([]string, error)

var patUpdateOpenBSD = regexp.MustCompile(`\w+`)

// QueryUpdatesOpenBSD checks for available updates on OpenBSD.
func (p *Probe) QueryUpdatesOpenBSD(d *model.Device, port int) ([]string, error) {
	const cmd = "doas syspatch -c"
	var (
		err             error
		output, updates []string
	)

	if output, err = p.executeCommand(d, port, cmd); err != nil {
		if err == ErrPingOffline {
			return nil, err
		}
		_ = p.disconnect(d)
		return nil, err
	}

	updates = make([]string, 0, len(output))

	for _, l := range output {
		if patUpdateOpenBSD.MatchString(l) {
			updates = append(updates, l)
		}
	}

	if len(updates) == 0 {
		return nil, nil
	}

	return updates, nil
} // func (p *Probe) QueryUpdatesOpenBSD(d *model.Device, port int) ([]string, error)

func (p *Probe) QueryUpdatesFreeBSD(d *model.Device, port int) ([]string, error) {
	const cmd = "doas freebsd-update updatesready"
	var (
		err             error
		output, updates []string
	)

	if output, err = p.executeCommand(d, port, cmd); err != nil {
		if err == ErrPingOffline {
			return nil, err
		}
		_ = p.disconnect(d)
		return nil, err
	}

	updates = make([]string, 0, len(output))

	for _, l := range output {
		if patUpdateOpenBSD.MatchString(l) {
			updates = append(updates, l)
		}
	}

	if len(updates) == 0 {
		return nil, nil
	}

	return updates, nil
} // func (p *Probe) QueryUpdatesFreeBSD(d *model.Device, port int) ([]string, error)

// QueryUpdates attempts to query the given Device for available updates.
func (p *Probe) QueryUpdates(d *model.Device, port int) ([]string, error) {
	switch d.OS {
	case "Debian GNU/Linux":
		fallthrough
	case "Raspbian GNU/Linux":
		return p.QueryUpdatesDebian(d, port)
	case "openSUSE Tumbleweed":
		fallthrough
	case "openSUSE Leap":
		return p.QueryUpdatesSuse(d, port)
	case "Fedora Linux":
		return p.QueryUpdatesFedora(d, port)
	case "Arch Linux":
		return p.QueryUpdatesArch(d, port)
	case "OpenBSD":
		return p.QueryUpdatesOpenBSD(d, port)
	case "FreeBSD":
		return p.QueryUpdatesFreeBSD(d, port)
	default:
		p.log.Printf("[TRACE] Don't know how to query %s (running %s) for updates\n",
			d.Name,
			d.OS)
		return nil, nil
	}
} // func (p *Probe) QueryUpdates(d *model.Device, port int) ([]string, error)

// Sample output:
// 18:01:18  2 Tage  0:22 an,  2 Benutzer,  Durchschnittslast: 1,08, 0,98, 0,94
// 6:02PM  up 56 days,  5:16, 4 users, load averages: 0.00, 0.01, 0.00

var uptimePat = regexp.MustCompile(
	`:\s+(\d+[,.]\d+),?\s+(\d+[,.]\d+),?\s+(\d+[,.]\d+)$`)

// QueryUptime attempts to extract the system load average from the given Device.
func (p *Probe) QueryUptime(d *model.Device, port int) (*model.Uptime, error) {
	const cmd = "/usr/bin/uptime"
	var (
		err   error
		res   []string
		match []string
		up    = &model.Uptime{
			DevID:     d.ID,
			Timestamp: time.Now(),
		}
	)

	if res, err = p.executeCommand(d, port, cmd); err != nil {
		if err == ErrPingOffline {
			return nil, err
		}
		var ex = fmt.Errorf("Failed to query uptime/loadavg on %s: %w",
			d.Name,
			err)
		p.log.Printf("[ERROR] %s\n", ex.Error())
		return nil, ex
	} else if match = uptimePat.FindStringSubmatch(res[0]); match == nil {
		var ex = fmt.Errorf("Cannot parse the output of uptime(1) from %s: %q",
			d.Name,
			res[0])
		p.log.Printf("[ERROR] %s\n", ex.Error())
		return nil, ex
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
				return nil, ex
			}

			up.Load[idx] = f
		}
	}

	// ...

	return up, nil
} // func (p *Probe) QueryLoadAvg(d *model.Device, port int) ([3]float64, error)

var dfPat = regexp.MustCompile(`(\d+)%`)

// QueryDiskFree queries a Device for the free disk space on its root filesystem.
func (p *Probe) QueryDiskFree(d *model.Device, port int) (int64, error) {
	const cmd = "env LC_ALL=en_EN.UTF-8 df -h /"
	var (
		err        error
		res, match []string
		used, free int64
	)

	if res, err = p.executeCommand(d, port, cmd); err != nil {
		if err == ErrPingOffline {
			return 0, err
		}
		var ex = fmt.Errorf("Failed to query free disk space on %s: %w",
			d.Name,
			err)
		p.log.Printf("[ERROR] %s\n", ex.Error())
		return 0, ex
	} else if len(res) < 2 {
		var ex = fmt.Errorf("Cannot parse output of \"df -h\" on %s: %s\n",
			d.Name,
			strings.Join(res, "\n"))
		p.log.Printf("[ERROR] %s\n", ex.Error())
		return 0, ex
	} else if match = dfPat.FindStringSubmatch(res[1]); match == nil {
		var ex = fmt.Errorf("Cannot parse output of \"df -h\" on %s: %s\n",
			d.Name,
			strings.Join(res, "\n"))
		p.log.Printf("[ERROR] %s\n", ex.Error())
		return 0, ex
	} else if used, err = strconv.ParseInt(match[1], 10, 64); err != nil {
		var ex = fmt.Errorf("Cannot parse free disk space on %s: %q - %w\n",
			d.Name,
			match[1],
			err)
		p.log.Printf("[ERROR] %s\n", ex.Error())
		return 0, ex
	}

	free = 100 - used

	return free, nil
} // func (p *Probe) QueryDiskFree(d *model.Device, port int) (int64, error)
