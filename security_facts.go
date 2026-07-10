// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "strings"

// collectFIPSEnabled reports whether the kernel runs in FIPS mode. On Linux it
// reads /proc/sys/crypto/fips_enabled (1 = enabled); on every other platform, and
// when the source is absent, it reports false — a total boolean fact, as Facter
// always publishes fips_enabled.
func (c *Collection) collectFIPSEnabled() (any, bool) {
	if c.env.goos == "linux" {
		if v, ok := c.env.readText("/proc/sys/crypto/fips_enabled"); ok {
			return strings.TrimSpace(v) == "1", true
		}
	}
	return false, true
}

// collectAIOAgentVersion reports the puppet-agent all-in-one package version from
// the VERSION file the AIO layout ships. Absent when puppet-agent is not
// installed, so it never fabricates a value.
func (c *Collection) collectAIOAgentVersion() (any, bool) {
	if v, ok := c.env.readText("/opt/puppetlabs/puppet/VERSION"); ok {
		if s := strings.TrimSpace(v); s != "" {
			return s, true
		}
	}
	return nil, false
}

// collectAugeasVersion reports the installed Augeas library version, read from
// augparse --version. It is best-effort: absent when Augeas is not installed.
func (c *Collection) collectAugeasVersion() (any, bool) {
	out, ok := c.env.cmd("augparse", "--version")
	if !ok {
		return nil, false
	}
	if v := versionToken(out); v != "" {
		return v, true
	}
	return nil, false
}

// collectEnvWindowsInstalldir reports the puppet-agent install directory on
// Windows, taken from the FACTER_env_windows_installdir environment variable the
// installer exports. Absent off Windows or when the variable is unset.
func (c *Collection) collectEnvWindowsInstalldir() (any, bool) {
	if c.env.goos != "windows" {
		return nil, false
	}
	if v, ok := c.env.lookupEnv("FACTER_env_windows_installdir"); ok && v != "" {
		return v, true
	}
	return nil, false
}

// versionToken returns the first whitespace-separated field that begins with a
// digit, the version number in a "tool X.Y.Z ..." banner.
func versionToken(s string) string {
	for _, f := range strings.Fields(s) {
		if f != "" && f[0] >= '0' && f[0] <= '9' {
			return f
		}
	}
	return ""
}
