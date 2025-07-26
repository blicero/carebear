// /home/krylon/go/src/github.com/blicero/carebear/probe/probe_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 24. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-07-26 16:41:13 krylon>

package probe

import (
	"strconv"
	"strings"
	"testing"
)

func TestUptimePattern(t *testing.T) {
	type testCase struct {
		output         string
		expectErr      bool
		expectedResult [3]float64
	}

	var cases = []testCase{
		{
			output: "18:01:18  2 Tage  0:22 an,  2 Benutzer,  Durchschnittslast: 1,08, 0,98, 0,94",
			expectedResult: [3]float64{
				1.08,
				0.98,
				0.94,
			},
		},
		{
			output: "6:02PM  up 56 days,  5:16, 4 users, load averages: 0.00, 0.01, 0.00",
			expectedResult: [3]float64{
				0.0,
				0.01,
				0.0,
			},
		},
	}

	for _, c := range cases {
		var match = uptimePat.FindStringSubmatch(c.output)

		if match == nil {
			if !c.expectErr {
				t.Errorf("Failed to match sample output of uptime(1):\n\t%q",
					c.output)
			}
		} else {
			var load [3]float64

			for i, x := range match[1:] {
				var (
					err error
					s   string
				)

				s = strings.Replace(x, ",", ".", 1)

				if load[i], err = strconv.ParseFloat(s, 64); err != nil {
					t.Errorf("Cannot parse float %q: %s",
						s,
						err.Error())
				} else if load[i] != c.expectedResult[i] {
					t.Errorf("ParseFloat returned unpexected result: %f (expected %f)",
						load[i],
						c.expectedResult[i])
				}
			}
		}
	}
} // func TestUptimePattern(t *testing.T)
