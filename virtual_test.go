// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "testing"

func TestVirtualDocker(t *testing.T) {
	f := fakeEnv{goos: "linux", files: map[string]string{"/.dockerenv": ""}}
	c := f.collection()
	if v, _ := c.Value("virtual"); v != "docker" {
		t.Fatalf("virtual = %v", v)
	}
	if v, _ := c.Value("is_virtual"); v != true {
		t.Fatalf("is_virtual = %v", v)
	}
}

func TestVirtualPodman(t *testing.T) {
	f := fakeEnv{goos: "linux", files: map[string]string{"/run/.containerenv": ""}}
	if v, _ := f.collection().Value("virtual"); v != "podman" {
		t.Fatalf("virtual = %v", v)
	}
}

func TestVirtualCgroup(t *testing.T) {
	cases := map[string]string{
		"11:devices:/docker/abc": "docker",
		"5:memory:/kubepods/pod": "kubernetes",
		"3:cpu:/lxc/ct1":         "lxc",
		"1:name=systemd:/":       "", // falls through to DMI/physical
	}
	for cg, want := range cases {
		f := fakeEnv{goos: "linux", files: map[string]string{"/proc/1/cgroup": cg}}
		got, _ := f.collection().Value("virtual")
		if want == "" {
			if got != "physical" {
				t.Errorf("cgroup %q -> %v, want physical", cg, got)
			}
			continue
		}
		if got != want {
			t.Errorf("cgroup %q -> %v, want %v", cg, got, want)
		}
	}
}

func TestVirtualDMI(t *testing.T) {
	f := fakeEnv{goos: "linux", files: map[string]string{"/sys/class/dmi/id/product_name": "VMware Virtual Platform\n"}}
	if v, _ := f.collection().Value("virtual"); v != "vmware" {
		t.Fatalf("dmi vmware = %v", v)
	}
}

func TestDetectHypervisor(t *testing.T) {
	cases := map[string]string{
		"vmware":     "vmware",
		"virtualbox": "virtualbox",
		"oracle":     "virtualbox",
		"kvm":        "kvm",
		"qemu":       "kvm",
		"bochs":      "kvm",
		"xen":        "xen",
		"microsoft":  "hyperv",
		"google":     "gce",
		"amazon":     "kvm",
		"ec2":        "kvm",
		"parallels":  "parallels",
		"dell inc.":  "",
	}
	for in, want := range cases {
		if got := detectHypervisor(in, "", ""); got != want {
			t.Errorf("detectHypervisor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestVirtualCPUFlag(t *testing.T) {
	f := fakeEnv{goos: "linux", files: map[string]string{"/proc/cpuinfo": "flags\t: fpu vme hypervisor\n"}}
	if v, _ := f.collection().Value("virtual"); v != "kvm" {
		t.Fatalf("cpu hypervisor = %v", v)
	}
}

func TestVirtualPhysical(t *testing.T) {
	f := fakeEnv{goos: "linux", files: map[string]string{"/proc/cpuinfo": "flags\t: fpu vme\n"}}
	c := f.collection()
	if v, _ := c.Value("virtual"); v != "physical" {
		t.Fatalf("physical = %v", v)
	}
	if v, _ := c.Value("is_virtual"); v != false {
		t.Fatalf("is_virtual = %v", v)
	}
}

func TestCPUHasHypervisor(t *testing.T) {
	if cpuHasHypervisor("model\t: x\n") {
		t.Error("no flags line")
	}
	if cpuHasHypervisor("flags : a b\n") {
		t.Error("flags without hypervisor")
	}
	if !cpuHasHypervisor("flags : a hypervisor b\n") {
		t.Error("flags with hypervisor")
	}
}

func TestVirtualDarwin(t *testing.T) {
	f := fakeEnv{goos: "darwin", cmds: map[string]string{"sysctl -n kern.hv_vmm_present": "1\n"}}
	if v, _ := f.collection().Value("virtual"); v != "vmware" {
		t.Fatalf("darwin vm = %v", v)
	}
	// Not a guest.
	f2 := fakeEnv{goos: "darwin", cmds: map[string]string{"sysctl -n kern.hv_vmm_present": "0\n"}}
	if v, _ := f2.collection().Value("virtual"); v != "physical" {
		t.Fatalf("darwin physical = %v", v)
	}
	// sysctl unavailable.
	if v, _ := (fakeEnv{goos: "darwin"}).collection().Value("virtual"); v != "physical" {
		t.Fatalf("darwin no sysctl = %v", v)
	}
}

func TestVirtualGeneric(t *testing.T) {
	if v, _ := (fakeEnv{goos: "plan9"}).collection().Value("virtual"); v != "physical" {
		t.Fatalf("generic = %v", v)
	}
}
