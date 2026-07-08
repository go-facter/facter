// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"sort"
	"strings"
)

// mountEntry is one parsed row of /proc/mounts.
type mountEntry struct {
	device     string
	mountpoint string
	fstype     string
	options    []string
}

// dfEntry is the size accounting df reports for a mountpoint, in 1 KiB blocks.
type dfEntry struct {
	sizeKB, usedKB, availKB uint64
	capacity                string
}

// collectMountpoints builds the structured mountpoints fact from /proc/mounts
// (device, filesystem type, options) joined with df's size accounting. It is a
// Linux fact; other platforms report none.
func (c *Collection) collectMountpoints() (any, bool) {
	if c.env.goos != "linux" {
		return nil, false
	}
	text, ok := c.env.readText("/proc/mounts")
	if !ok {
		return nil, false
	}
	mounts := parseMounts(text)
	if len(mounts) == 0 {
		return nil, false
	}
	sizes := map[string]dfEntry{}
	if out, ok := c.env.cmd("df", "-P", "-k"); ok {
		sizes = parseDF(out)
	}

	out := map[string]any{}
	for _, m := range mounts {
		entry := map[string]any{
			"device":     m.device,
			"filesystem": m.fstype,
			"options":    toAnySlice(m.options),
		}
		if df, ok := sizes[m.mountpoint]; ok {
			size := df.sizeKB * 1024
			used := df.usedKB * 1024
			avail := df.availKB * 1024
			entry["size_bytes"] = size
			entry["size"] = humanBytes(size)
			entry["used_bytes"] = used
			entry["used"] = humanBytes(used)
			entry["available_bytes"] = avail
			entry["available"] = humanBytes(avail)
			entry["capacity"] = df.capacity
		}
		out[m.mountpoint] = entry
	}
	return out, true
}

// parseMounts parses /proc/mounts rows into mountEntry records.
func parseMounts(text string) []mountEntry {
	var out []mountEntry
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		out = append(out, mountEntry{
			device:     unescapeMount(fields[0]),
			mountpoint: unescapeMount(fields[1]),
			fstype:     fields[2],
			options:    strings.Split(fields[3], ","),
		})
	}
	return out
}

// unescapeMount decodes the octal \040 style escapes the kernel uses for spaces
// and other whitespace in /proc/mounts paths.
func unescapeMount(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) && isOctal(s[i+1]) && isOctal(s[i+2]) && isOctal(s[i+3]) {
			v := (int(s[i+1]-'0') << 6) | (int(s[i+2]-'0') << 3) | int(s[i+3]-'0')
			b.WriteByte(byte(v))
			i += 3
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// isOctal reports whether ch is an octal digit.
func isOctal(ch byte) bool { return ch >= '0' && ch <= '7' }

// parseDF parses the POSIX df -P -k table keyed by mountpoint.
func parseDF(out string) map[string]dfEntry {
	res := map[string]dfEntry{}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i, line := range lines {
		if i == 0 {
			continue // header
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		mount := strings.Join(fields[5:], " ")
		res[mount] = dfEntry{
			sizeKB:   atou64(fields[1]),
			usedKB:   atou64(fields[2]),
			availKB:  atou64(fields[3]),
			capacity: fields[4],
		}
	}
	return res
}

// collectFilesystems reports the comma-separated list of on-disk filesystem
// types the kernel supports, from /proc/filesystems. Linux only.
func (c *Collection) collectFilesystems() (any, bool) {
	if c.env.goos != "linux" {
		return nil, false
	}
	text, ok := c.env.readText("/proc/filesystems")
	if !ok {
		return nil, false
	}
	list := parseFilesystems(text)
	if list == "" {
		return nil, false
	}
	return list, true
}

// parseFilesystems keeps the block-device (non-"nodev") filesystem names from
// /proc/filesystems, sorted and comma-joined.
func parseFilesystems(text string) string {
	seen := map[string]struct{}{}
	var names []string
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "nodev" {
			continue
		}
		name := fields[len(fields)-1]
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}

// collectDisks builds the structured disks fact from /sys/block, skipping the
// synthetic loop and ram devices. Linux only.
func (c *Collection) collectDisks() (any, bool) {
	if c.env.goos != "linux" {
		return nil, false
	}
	ents, err := c.env.readDir("/sys/block")
	if err != nil {
		return nil, false
	}
	out := map[string]any{}
	for _, e := range ents {
		name := e.Name
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") {
			continue
		}
		entry := map[string]any{}
		if s, ok := c.env.readText("/sys/block/" + name + "/size"); ok {
			bytes := atou64(firstField(s)) * 512
			entry["size_bytes"] = bytes
			entry["size"] = humanBytes(bytes)
		}
		if v, ok := c.env.readText("/sys/block/" + name + "/device/model"); ok {
			entry["model"] = strings.TrimSpace(v)
		}
		if v, ok := c.env.readText("/sys/block/" + name + "/device/vendor"); ok {
			entry["vendor"] = strings.TrimSpace(v)
		}
		out[name] = entry
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}
