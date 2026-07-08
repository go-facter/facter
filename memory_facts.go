// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"fmt"
	"strconv"
	"strings"
)

// memStat is a total/used/available byte triple for either system RAM or swap.
type memStat struct {
	total, used, available uint64
	known                  bool
}

// collectMemory builds the structured memory fact with system and swap
// sub-maps. Platforms that cannot supply memory data return no fact.
func (c *Collection) collectMemory() (any, bool) {
	var system, swap memStat
	switch c.env.goos {
	case "linux":
		system, swap = c.memoryLinux()
	case "darwin":
		system, swap = c.memoryDarwin()
	default:
		return nil, false
	}
	if !system.known {
		return nil, false
	}
	out := map[string]any{"system": memStatMap(system)}
	if swap.known {
		out["swap"] = memStatMap(swap)
	}
	return out, true
}

// memStatMap renders a memStat as Facter's total/available/used byte counts,
// human-readable strings and capacity percentage.
func memStatMap(m memStat) map[string]any {
	return map[string]any{
		"total":           humanBytes(m.total),
		"total_bytes":     m.total,
		"available":       humanBytes(m.available),
		"available_bytes": m.available,
		"used":            humanBytes(m.used),
		"used_bytes":      m.used,
		"capacity":        capacity(m.used, m.total),
	}
}

// memoryLinux derives system and swap statistics from /proc/meminfo.
func (c *Collection) memoryLinux() (system, swap memStat) {
	text, _ := c.env.readText("/proc/meminfo")
	kv := parseMemInfo(text)

	total := kv["MemTotal"]
	avail, ok := kv["MemAvailable"]
	if !ok {
		avail = kv["MemFree"]
	}
	if total > 0 {
		system = newMemStat(total, avail)
	}

	swapTotal := kv["SwapTotal"]
	swapFree := kv["SwapFree"]
	if swapTotal > 0 {
		swap = newMemStat(swapTotal, swapFree)
	}
	return system, swap
}

// memoryDarwin derives RAM from hw.memsize and page statistics from vm_stat, and
// swap usage from vm.swapusage.
func (c *Collection) memoryDarwin() (system, swap memStat) {
	var total uint64
	if out, ok := c.env.cmd("sysctl", "-n", "hw.memsize"); ok {
		total = atou64(firstField(out))
	}
	if total > 0 {
		avail := total
		if out, ok := c.env.cmd("vm_stat"); ok {
			avail = parseVMStatFree(out)
		}
		if avail > total {
			avail = total
		}
		system = newMemStat(total, avail)
	}
	if out, ok := c.env.cmd("sysctl", "-n", "vm.swapusage"); ok {
		if st, ok := parseSwapUsage(out); ok {
			swap = st
		}
	}
	return system, swap
}

// newMemStat builds a memStat from total and available bytes, deriving used.
func newMemStat(total, available uint64) memStat {
	if available > total {
		available = total
	}
	return memStat{total: total, used: total - available, available: available, known: true}
}

// parseMemInfo parses /proc/meminfo into a map of byte counts (its kB values are
// converted to bytes).
func parseMemInfo(text string) map[string]uint64 {
	out := map[string]uint64{}
	for _, line := range strings.Split(text, "\n") {
		key, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}
		val := atou64(fields[0])
		if len(fields) >= 2 && fields[1] == "kB" {
			val *= 1024
		}
		out[strings.TrimSpace(key)] = val
	}
	return out
}

// parseVMStatFree sums the free and inactive pages reported by vm_stat into an
// available-bytes figure.
func parseVMStatFree(out string) uint64 {
	pageSize := uint64(4096)
	var freePages, inactivePages, specPages uint64
	for _, line := range strings.Split(out, "\n") {
		if i := strings.Index(line, "page size of "); i >= 0 {
			rest := line[i+len("page size of "):]
			pageSize = atou64(firstField(rest))
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		n := atou64(strings.TrimRight(strings.TrimSpace(val), "."))
		switch strings.TrimSpace(key) {
		case "Pages free":
			freePages = n
		case "Pages inactive":
			inactivePages = n
		case "Pages speculative":
			specPages = n
		}
	}
	return (freePages + inactivePages + specPages) * pageSize
}

// parseSwapUsage parses macOS "total = 1024.00M used = 0.00M free = 1024.00M".
func parseSwapUsage(out string) (memStat, bool) {
	fields := strings.Fields(out)
	vals := map[string]uint64{}
	for i := 0; i+2 < len(fields); i++ {
		if fields[i+1] == "=" {
			vals[fields[i]] = parseHumanSize(fields[i+2])
		}
	}
	total, ok := vals["total"]
	if !ok {
		return memStat{}, false
	}
	return newMemStat(total, vals["free"]), true
}

// parseHumanSize converts "1024.00M" / "2.00G" to bytes.
func parseHumanSize(s string) uint64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	mult := uint64(1)
	switch s[len(s)-1] {
	case 'K', 'k':
		mult, s = 1024, s[:len(s)-1]
	case 'M', 'm':
		mult, s = 1024*1024, s[:len(s)-1]
	case 'G', 'g':
		mult, s = 1024*1024*1024, s[:len(s)-1]
	case 'T', 't':
		mult, s = 1024*1024*1024*1024, s[:len(s)-1]
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return uint64(f * float64(mult))
}

// humanBytes formats a byte count the way Facter prints memory sizes: an exact
// "N bytes" below 1 KiB, otherwise a two-decimal binary-unit string.
func humanBytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d bytes", n)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	val := float64(n)
	idx := -1
	for val >= unit && idx < len(units)-1 {
		val /= unit
		idx++
	}
	return fmt.Sprintf("%.2f %s", val, units[idx])
}

// capacity renders used/total as Facter's "NN.NN%" utilisation string.
func capacity(used, total uint64) string {
	if total == 0 {
		return "0.00%"
	}
	return fmt.Sprintf("%.2f%%", float64(used)/float64(total)*100)
}

// atou64 parses an unsigned integer, returning 0 on any error.
func atou64(s string) uint64 {
	n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return n
}
