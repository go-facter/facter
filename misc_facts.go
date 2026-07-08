// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

// collectTimezone reports the host's timezone abbreviation (for example "UTC" or
// "CEST"), taken from the current local time.
func (c *Collection) collectTimezone() (any, bool) {
	name, _ := c.env.now().Zone()
	if name == "" {
		return nil, false
	}
	return name, true
}

// collectPath reports the process PATH environment variable.
func (c *Collection) collectPath() (any, bool) {
	if v, ok := c.env.lookupEnv("PATH"); ok {
		return v, true
	}
	return nil, false
}

// collectFacterVersion reports the engine's Facter schema version.
func (c *Collection) collectFacterVersion() (any, bool) {
	return Version, true
}
