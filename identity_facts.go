// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "strconv"

// collectIdentity builds the structured identity fact: the current user and
// group, their numeric ids where applicable, and whether the process is
// privileged.
func (c *Collection) collectIdentity() (any, bool) {
	u, err := c.env.curUser()
	if err != nil {
		return nil, false
	}
	out := map[string]any{
		"user":       u.Username,
		"group":      u.Group,
		"uid":        numericID(u.UID),
		"gid":        numericID(u.GID),
		"privileged": c.privileged(),
	}
	return out, true
}

// privileged reports whether the process runs with elevated rights: euid 0 on
// Unix. Windows exposes euid -1, so this is reported false there.
func (c *Collection) privileged() bool {
	return c.env.euid() == 0
}

// numericID returns the integer form of a uid/gid when it is numeric (Unix), or
// the original string (a Windows SID).
func numericID(id string) any {
	if n, err := strconv.Atoi(id); err == nil {
		return n
	}
	return id
}
