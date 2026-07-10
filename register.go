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
	c.AddFunc("dmi", (*Collection).collectDMI)
	c.AddFunc("load_averages", (*Collection).collectLoadAverages)
	c.AddFunc("ssh", (*Collection).collectSSH)
	c.AddFunc("selinux", (*Collection).collectSELinux)
	c.AddFunc("cloud", (*Collection).collectCloud)
	c.AddFunc("ec2_metadata", (*Collection).collectEC2Metadata)
	c.AddFunc("ruby", (*Collection).collectRuby)
	c.AddFunc("fips_enabled", (*Collection).collectFIPSEnabled)
	c.AddFunc("aio_agent_version", (*Collection).collectAIOAgentVersion)
	c.AddFunc("augeasversion", (*Collection).collectAugeasVersion)
	c.AddFunc("env_windows_installdir", (*Collection).collectEnvWindowsInstalldir)

	// SSH legacy flat facts (sshXXXkey and sshfp_XXX) derive from the structured
	// ssh fact, so the host keys are read and fingerprinted at most once.
	for _, algo := range []string{"dsa", "rsa", "ecdsa", "ed25519"} {
		algo := algo
		c.AddFunc("ssh"+algo+"key", func(cc *Collection) (any, bool) { return cc.collectSSHLegacyKey(algo) })
		c.AddFunc("sshfp_"+algo, func(cc *Collection) (any, bool) { return cc.collectSSHFP(algo) })
	}

	// Memory sizes in mebibytes, the legacy *_mb facts manifests still read.
	memMB := map[string]string{
		"memorysize_mb": "memory.system.total_bytes",
		"memoryfree_mb": "memory.system.available_bytes",
		"swapsize_mb":   "memory.swap.total_bytes",
		"swapfree_mb":   "memory.swap.available_bytes",
	}
	mbNames := make([]string, 0, len(memMB))
	for name := range memMB {
		mbNames = append(mbNames, name)
	}
	sort.Strings(mbNames)
	for _, name := range mbNames {
		path := memMB[name]
		c.AddFunc(name, func(cc *Collection) (any, bool) { return cc.memoryMB(path) })
	}

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
		"bios_vendor":               "dmi.bios.vendor",
		"bios_version":              "dmi.bios.version",
		"bios_release_date":         "dmi.bios.release_date",
		"boardmanufacturer":         "dmi.board.manufacturer",
		"boardproductname":          "dmi.board.product",
		"boardserialnumber":         "dmi.board.serial_number",
		"boardassettag":             "dmi.board.asset_tag",
		"chassistype":               "dmi.chassis.type",
		"chassisassettag":           "dmi.chassis.asset_tag",
		"manufacturer":              "dmi.manufacturer",
		"productname":               "dmi.product.name",
		"serialnumber":              "dmi.product.serial_number",
		"uuid":                      "dmi.product.uuid",
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
