// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"strconv"
	"strings"
)

// collectProcessors builds the structured processors fact: the logical count,
// the physical socket count, one model string per logical CPU, and the ISA.
func (c *Collection) collectProcessors() (any, bool) {
	var count, physical int
	var models []string
	isa := archName(c.env.goarch, c.unameMachine())

	switch c.env.goos {
	case "linux":
		text, _ := c.env.readText("/proc/cpuinfo")
		count, models, physical = parseCPUInfo(text)
	case "darwin":
		count, physical, models = c.processorsDarwin()
	case "windows":
		count, physical, models = c.processorsWindows()
	}

	if count == 0 {
		count = c.env.numCPU
	}
	if physical == 0 {
		physical = 1
	}
	if len(models) == 0 {
		models = repeatModel("unknown", count)
	}

	return map[string]any{
		"count":         count,
		"physicalcount": physical,
		"models":        toAnySlice(models),
		"isa":           isa,
	}, true
}

// parseCPUInfo reads Linux /proc/cpuinfo: the logical count is the number of
// "processor" records, models come from each "model name", and the physical
// count is the number of distinct "physical id" values (at least one).
func parseCPUInfo(text string) (count int, models []string, physical int) {
	sockets := map[string]struct{}{}
	for _, line := range strings.Split(text, "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "processor":
			count++
		case "model name":
			models = append(models, val)
		case "physical id":
			sockets[val] = struct{}{}
		}
	}
	physical = len(sockets)
	if physical == 0 && count > 0 {
		physical = 1
	}
	return count, models, physical
}

// processorsDarwin reads the logical/physical counts and CPU brand via sysctl.
func (c *Collection) processorsDarwin() (count, physical int, models []string) {
	if out, ok := c.env.cmd("sysctl", "-n", "hw.logicalcpu"); ok {
		count = atoiSafe(firstField(out))
	}
	if out, ok := c.env.cmd("sysctl", "-n", "hw.physicalcpu"); ok {
		physical = atoiSafe(firstField(out))
	}
	brand := "unknown"
	if out, ok := c.env.cmd("sysctl", "-n", "machdep.cpu.brand_string"); ok {
		if b := strings.TrimSpace(out); b != "" {
			brand = b
		}
	}
	if count == 0 {
		count = c.env.numCPU
	}
	models = repeatModel(brand, count)
	return count, physical, models
}

// processorsWindows reads the count and CPU identifier from the environment.
func (c *Collection) processorsWindows() (count, physical int, models []string) {
	if v, ok := c.env.lookupEnv("NUMBER_OF_PROCESSORS"); ok {
		count = atoiSafe(v)
	}
	if count == 0 {
		count = c.env.numCPU
	}
	brand := "unknown"
	if v, ok := c.env.lookupEnv("PROCESSOR_IDENTIFIER"); ok {
		if v = strings.TrimSpace(v); v != "" {
			brand = v
		}
	}
	models = repeatModel(brand, count)
	return count, physical, models
}

// repeatModel returns n copies of model, matching Facter's one-entry-per-logical
// -CPU models array. A zero or negative n yields a single entry.
func repeatModel(model string, n int) []string {
	if n < 1 {
		n = 1
	}
	out := make([]string, n)
	for i := range out {
		out[i] = model
	}
	return out
}

// toAnySlice widens a []string to []any for embedding in a fact map.
func toAnySlice(in []string) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}

// atoiSafe parses an integer, returning 0 on any error.
func atoiSafe(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}
