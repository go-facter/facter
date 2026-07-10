// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"strconv"
	"strings"
)

// collectLoadAverages builds the structured load_averages fact: the 1-, 5- and
// 15-minute run-queue averages as floats, keyed "1m"/"5m"/"15m" exactly as
// Facter. Linux reads /proc/loadavg; Darwin reads the vm.loadavg sysctl. A
// platform with no load source (Windows) reports no fact.
func (c *Collection) collectLoadAverages() (any, bool) {
	var one, five, fifteen float64
	var ok bool
	switch c.env.goos {
	case "linux":
		text, present := c.env.readText("/proc/loadavg")
		if !present {
			return nil, false
		}
		one, five, fifteen, ok = parseLoadAvg(text)
	case "darwin":
		out, present := c.env.cmd("sysctl", "-n", "vm.loadavg")
		if !present {
			return nil, false
		}
		one, five, fifteen, ok = parseLoadAvg(strings.Trim(strings.TrimSpace(out), "{}"))
	default:
		return nil, false
	}
	if !ok {
		return nil, false
	}
	return map[string]any{
		"1m":  one,
		"5m":  five,
		"15m": fifteen,
	}, true
}

// parseLoadAvg reads the first three whitespace-separated floats of a loadavg
// string (Linux "/proc/loadavg" or the brace-stripped Darwin vm.loadavg), all of
// which start "1m 5m 15m ...".
func parseLoadAvg(text string) (one, five, fifteen float64, ok bool) {
	fields := strings.Fields(text)
	if len(fields) < 3 {
		return 0, 0, 0, false
	}
	var err error
	if one, err = strconv.ParseFloat(fields[0], 64); err != nil {
		return 0, 0, 0, false
	}
	if five, err = strconv.ParseFloat(fields[1], 64); err != nil {
		return 0, 0, 0, false
	}
	if fifteen, err = strconv.ParseFloat(fields[2], 64); err != nil {
		return 0, 0, 0, false
	}
	return one, five, fifteen, true
}
