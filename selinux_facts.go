// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "strings"

// collectSELinux builds the structured selinux fact. On a host where selinuxfs is
// mounted it reports enabled/enforced, the current runtime mode, the policy
// version and the configured mode/policy from /etc/selinux/config; where SELinux
// is absent it reports {enabled: false}, matching Facter. Linux only.
func (c *Collection) collectSELinux() (any, bool) {
	if c.env.goos != "linux" {
		return nil, false
	}
	enforce, ok := c.env.readText("/sys/fs/selinux/enforce")
	if !ok {
		// selinuxfs not mounted: SELinux is not active on this host.
		return map[string]any{"enabled": false}, true
	}
	enforced := strings.TrimSpace(enforce) == "1"
	out := map[string]any{
		"enabled":  true,
		"enforced": enforced,
	}
	out["current_mode"] = modeLabel(enforced)
	if v, ok := c.env.readText("/sys/fs/selinux/policyvers"); ok {
		if pv := strings.TrimSpace(v); pv != "" {
			out["policy_version"] = pv
		}
	}
	if v, ok := c.env.readText("/etc/selinux/config"); ok {
		mode, policy := parseSELinuxConfig(v)
		putNonEmpty(out, "config_mode", mode)
		putNonEmpty(out, "config_policy", policy)
	}
	return out, true
}

// modeLabel renders the enforce bit as Facter's mode string.
func modeLabel(enforced bool) string {
	if enforced {
		return "enforcing"
	}
	return "permissive"
}

// parseSELinuxConfig reads SELINUX (config_mode) and SELINUXTYPE (config_policy)
// from an /etc/selinux/config file, ignoring comments and blanks.
func parseSELinuxConfig(text string) (mode, policy string) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "SELINUX":
			mode = strings.TrimSpace(v)
		case "SELINUXTYPE":
			policy = strings.TrimSpace(v)
		}
	}
	return mode, policy
}
