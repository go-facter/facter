// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "strings"

// collectVirtual reports the virtualization technology the host runs under
// ("physical" when bare metal). Container runtimes take precedence over the
// underlying hypervisor, matching Facter's innermost-wins behaviour.
func (c *Collection) collectVirtual() (any, bool) {
	switch c.env.goos {
	case "linux":
		return c.virtualLinux(), true
	case "darwin":
		return c.virtualDarwin(), true
	default:
		return "physical", true
	}
}

// collectIsVirtual is the boolean companion of the virtual fact.
func (c *Collection) collectIsVirtual() (any, bool) {
	v, _ := c.Value("virtual")
	return stringify(v) != "physical", true
}

// virtualLinux detects containers via well-known markers first, then the
// hypervisor via DMI, then the CPU hypervisor flag.
func (c *Collection) virtualLinux() string {
	if ct := c.containerType(); ct != "" {
		return ct
	}
	product, _ := c.env.readText("/sys/class/dmi/id/product_name")
	sysVendor, _ := c.env.readText("/sys/class/dmi/id/sys_vendor")
	biosVendor, _ := c.env.readText("/sys/class/dmi/id/bios_vendor")
	if hv := detectHypervisor(product, sysVendor, biosVendor); hv != "" {
		return hv
	}
	if cpu, ok := c.env.readText("/proc/cpuinfo"); ok && cpuHasHypervisor(cpu) {
		return "kvm"
	}
	return "physical"
}

// containerType inspects the container markers a Linux guest exposes.
func (c *Collection) containerType() string {
	if _, ok := c.env.readText("/.dockerenv"); ok {
		return "docker"
	}
	if _, ok := c.env.readText("/run/.containerenv"); ok {
		return "podman"
	}
	if cg, ok := c.env.readText("/proc/1/cgroup"); ok {
		return cgroupContainer(cg)
	}
	return ""
}

// cgroupContainer classifies a container runtime from a /proc/1/cgroup dump.
func cgroupContainer(cgroup string) string {
	switch {
	case strings.Contains(cgroup, "docker"):
		return "docker"
	case strings.Contains(cgroup, "kubepods"):
		return "kubernetes"
	case strings.Contains(cgroup, "lxc"):
		return "lxc"
	default:
		return ""
	}
}

// detectHypervisor maps DMI product/vendor strings to a Facter hypervisor label.
func detectHypervisor(product, sysVendor, biosVendor string) string {
	hay := strings.ToLower(product + " " + sysVendor + " " + biosVendor)
	switch {
	case strings.Contains(hay, "vmware"):
		return "vmware"
	case strings.Contains(hay, "virtualbox"), strings.Contains(hay, "oracle"):
		return "virtualbox"
	case strings.Contains(hay, "kvm"), strings.Contains(hay, "qemu"), strings.Contains(hay, "bochs"):
		return "kvm"
	case strings.Contains(hay, "xen"):
		return "xen"
	case strings.Contains(hay, "microsoft"):
		return "hyperv"
	case strings.Contains(hay, "google"):
		return "gce"
	case strings.Contains(hay, "amazon"), strings.Contains(hay, "ec2"):
		return "kvm"
	case strings.Contains(hay, "parallels"):
		return "parallels"
	default:
		return ""
	}
}

// cpuHasHypervisor reports whether /proc/cpuinfo advertises the hypervisor flag.
func cpuHasHypervisor(cpuinfo string) bool {
	for _, line := range strings.Split(cpuinfo, "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(key) != "flags" {
			continue
		}
		for _, f := range strings.Fields(val) {
			if f == "hypervisor" {
				return true
			}
		}
	}
	return false
}

// virtualDarwin uses the kern.hv_vmm_present sysctl, which macOS sets to 1 when
// running as a guest.
func (c *Collection) virtualDarwin() string {
	if out, ok := c.env.cmd("sysctl", "-n", "kern.hv_vmm_present"); ok {
		if firstField(out) == "1" {
			return "vmware"
		}
	}
	return "physical"
}
