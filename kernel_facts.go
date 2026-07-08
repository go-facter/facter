// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "strings"

// kernelInfo is the resolved kernel identity Facter surfaces as four facts.
type kernelInfo struct {
	kernel       string
	release      string
	version      string
	majorVersion string
}

// kernelData resolves the kernel identity once and memoises it, since kernel,
// kernelrelease, kernelversion and kernelmajversion all derive from it.
func (c *Collection) kernelData() kernelInfo {
	return c.memo("kernel", func() any {
		return c.computeKernel()
	}).(kernelInfo)
}

// computeKernel gathers the kernel name and release per operating system and
// derives the numeric version fields.
func (c *Collection) computeKernel() kernelInfo {
	switch c.env.goos {
	case "windows":
		os, _ := c.Value("os")
		full := nestedString(os, "release", "full")
		return kernelInfo{
			kernel:       "windows",
			release:      full,
			version:      full,
			majorVersion: majorMinor(full),
		}
	default:
		name := "unknown"
		if out, ok := c.env.cmd("uname", "-s"); ok {
			name = firstField(out)
		}
		release := ""
		if out, ok := c.env.cmd("uname", "-r"); ok {
			release = firstField(out)
		}
		version := numericPrefix(release)
		return kernelInfo{
			kernel:       name,
			release:      release,
			version:      version,
			majorVersion: majorMinor(version),
		}
	}
}

// numericPrefix returns the leading run of digits and dots, so a Linux
// "6.5.0-27-generic" release yields the "6.5.0" kernelversion.
func numericPrefix(s string) string {
	end := 0
	for end < len(s) {
		ch := s[end]
		if (ch < '0' || ch > '9') && ch != '.' {
			break
		}
		end++
	}
	return strings.Trim(s[:end], ".")
}

// majorMinor keeps the first two dotted components of a version string.
func majorMinor(v string) string {
	parts := strings.Split(v, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return v
}

// nestedString reads a string at a nested map path, returning "" when any
// segment is missing or not a map/string.
func nestedString(v any, segs ...string) string {
	cur, ok := descend(v, segs)
	if !ok {
		return ""
	}
	s, _ := cur.(string)
	return s
}
