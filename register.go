// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"sort"
	"strings"
)

// registerCore wires every built-in fact into the Collection: the structured
// (aggregate) facts first, then the flat legacy aliases Puppet manifests still
// reference. Aliases resolve lazily through the structured facts, so the
// underlying probe runs at most once regardless of how a value is queried.
func registerCore(c *Collection) {
	// Structured / aggregate facts.
	c.AddFunc("os", (*Collection).collectOS)
	c.AddFunc("networking", (*Collection).collectNetworking)
	c.AddFunc("processors", (*Collection).collectProcessors)
	c.AddFunc("memory", (*Collection).collectMemory)
	c.AddFunc("identity", (*Collection).collectIdentity)
	c.AddFunc("system_uptime", (*Collection).collectSystemUptime)
	c.AddFunc("mountpoints", (*Collection).collectMountpoints)
	c.AddFunc("disks", (*Collection).collectDisks)
	c.AddFunc("filesystems", (*Collection).collectFilesystems)
	c.AddFunc("virtual", (*Collection).collectVirtual)
	c.AddFunc("is_virtual", (*Collection).collectIsVirtual)
	c.AddFunc("timezone", (*Collection).collectTimezone)
	c.AddFunc("path", (*Collection).collectPath)
	c.AddFunc("facterversion", (*Collection).collectFacterVersion)

	// Kernel facts share one memoised probe.
	c.AddFunc("kernel", func(cc *Collection) (any, bool) { return cc.kernelData().kernel, true })
	c.AddFunc("kernelrelease", func(cc *Collection) (any, bool) { return cc.kernelData().release, true })
	c.AddFunc("kernelversion", func(cc *Collection) (any, bool) { return cc.kernelData().version, true })
	c.AddFunc("kernelmajversion", func(cc *Collection) (any, bool) { return cc.kernelData().majorVersion, true })

	// Legacy flat aliases onto the structured facts.
	aliases := map[string]string{
		"operatingsystem":           "os.name",
		"osfamily":                  "os.family",
		"operatingsystemrelease":    "os.release.full",
		"operatingsystemmajrelease": "os.release.major",
		"architecture":              "os.architecture",
		"hardwaremodel":             "os.hardware",
		"hardwareisa":               "processors.isa",
		"hostname":                  "networking.hostname",
		"domain":                    "networking.domain",
		"fqdn":                      "networking.fqdn",
		"ipaddress":                 "networking.ip",
		"ipaddress6":                "networking.ip6",
		"macaddress":                "networking.mac",
		"netmask":                   "networking.netmask",
		"processorcount":            "processors.count",
		"physicalprocessorcount":    "processors.physicalcount",
		"memorysize":                "memory.system.total",
		"memoryfree":                "memory.system.available",
		"swapsize":                  "memory.swap.total",
		"swapfree":                  "memory.swap.available",
		"uptime":                    "system_uptime.uptime",
		"uptime_seconds":            "system_uptime.seconds",
		"uptime_days":               "system_uptime.days",
		"uptime_hours":              "system_uptime.hours",
		"id":                        "identity.user",
		"gid":                       "identity.gid",
	}
	names := make([]string, 0, len(aliases))
	for name := range aliases {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		c.alias(name, aliases[name])
	}

	// The interfaces legacy fact is the comma-joined list of interface names.
	c.AddFunc("interfaces", (*Collection).collectInterfacesList)
}

// alias registers name as a flat fact resolving to the value at a structured
// dotted path.
func (c *Collection) alias(name, path string) {
	c.AddFunc(name, func(cc *Collection) (any, bool) { return cc.Value(path) })
}

// collectInterfacesList reports the sorted, comma-separated interface names, the
// shape Facter's legacy interfaces fact takes.
func (c *Collection) collectInterfacesList() (any, bool) {
	v, ok := c.Value("networking.interfaces")
	if !ok {
		return nil, false
	}
	// networking only publishes interfaces as a non-empty map, so the assertion
	// is total here.
	m := v.(map[string]any)
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ","), true
}
