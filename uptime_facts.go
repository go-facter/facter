// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"strconv"
	"strings"
)

// collectSystemUptime builds the structured system_uptime fact (seconds, hours,
// days and the human uptime string). Platforms without a pure-Go uptime source
// report none.
func (c *Collection) collectSystemUptime() (any, bool) {
	secs, ok := c.uptimeSeconds()
	if !ok {
		return nil, false
	}
	return map[string]any{
		"seconds": secs,
		"hours":   secs / 3600,
		"days":    secs / 86400,
		"uptime":  formatUptime(secs),
	}, true
}

// uptimeSeconds resolves the host uptime in seconds and memoises it, since the
// structured fact and its legacy aliases share the probe.
func (c *Collection) uptimeSeconds() (int64, bool) {
	type result struct {
		secs int64
		ok   bool
	}
	r := c.memo("uptime", func() any {
		s, ok := c.computeUptime()
		return result{s, ok}
	}).(result)
	return r.secs, r.ok
}

// computeUptime reads the uptime from the per-OS source.
func (c *Collection) computeUptime() (int64, bool) {
	switch c.env.goos {
	case "linux":
		text, ok := c.env.readText("/proc/uptime")
		if !ok {
			return 0, false
		}
		return parseProcUptime(text)
	case "darwin":
		out, ok := c.env.cmd("sysctl", "-n", "kern.boottime")
		if !ok {
			return 0, false
		}
		boot, ok := parseBoottime(out)
		if !ok {
			return 0, false
		}
		now := c.env.now().Unix()
		if now < boot {
			return 0, false
		}
		return now - boot, true
	default:
		return 0, false
	}
}

// parseProcUptime reads the first (float seconds) field of /proc/uptime.
func parseProcUptime(text string) (int64, bool) {
	f := firstField(text)
	if f == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(f, 64)
	if err != nil {
		return 0, false
	}
	return int64(v), true
}

// parseBoottime extracts the sec value from macOS "{ sec = 1699999999, usec = 0 }".
func parseBoottime(out string) (int64, bool) {
	i := strings.Index(out, "sec = ")
	if i < 0 {
		return 0, false
	}
	rest := out[i+len("sec = "):]
	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, false
	}
	v, err := strconv.ParseInt(rest[:end], 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// formatUptime renders seconds the way Facter prints the uptime string.
func formatUptime(secs int64) string {
	days := secs / 86400
	hours := secs / 3600
	minutes := secs / 60
	switch {
	case days > 0:
		return plural(days, "day")
	case hours > 0:
		return strconv.FormatInt(hours, 10) + ":" + pad2((secs%3600)/60) + " hours"
	default:
		return plural(minutes, "minute")
	}
}

// plural renders "1 day" / "3 days".
func plural(n int64, unit string) string {
	s := strconv.FormatInt(n, 10) + " " + unit
	if n != 1 {
		s += "s"
	}
	return s
}

// pad2 renders a minutes value as a two-digit field.
func pad2(n int64) string {
	if n < 10 {
		return "0" + strconv.FormatInt(n, 10)
	}
	return strconv.FormatInt(n, 10)
}
