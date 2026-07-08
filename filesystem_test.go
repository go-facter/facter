// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "testing"

const procMounts = `/dev/sda1 / ext4 rw,relatime 0 0
tmpfs /run\040dir tmpfs rw 0 0
short line
`

const dfOutput = `Filesystem 1024-blocks Used Available Capacity Mounted on
/dev/sda1 1000 400 600 40% /
tmpfs 200 10 190 5% /run dir
badrow
`

func TestMountpointsLinux(t *testing.T) {
	f := fakeEnv{goos: "linux",
		files: map[string]string{"/proc/mounts": procMounts},
		cmds:  map[string]string{"df -P -k": dfOutput},
	}
	c := f.collection()
	if v, _ := c.Value("mountpoints./.device"); v != "/dev/sda1" {
		t.Fatalf("device = %v", v)
	}
	if v, _ := c.Value("mountpoints./.filesystem"); v != "ext4" {
		t.Fatalf("filesystem = %v", v)
	}
	if v, _ := c.Value("mountpoints./.size_bytes"); v != uint64(1000*1024) {
		t.Fatalf("size_bytes = %v", v)
	}
	if v, _ := c.Value("mountpoints./.capacity"); v != "40%" {
		t.Fatalf("capacity = %v", v)
	}
	// Escaped mountpoint with a space is decoded.
	if v, _ := c.Value(`mountpoints./run dir.device`); v != "tmpfs" {
		t.Fatalf("escaped mount = %v", v)
	}
}

func TestMountpointsNoDF(t *testing.T) {
	// df unavailable: entries exist without size accounting.
	f := fakeEnv{goos: "linux", files: map[string]string{"/proc/mounts": procMounts}}
	c := f.collection()
	if _, ok := c.Value("mountpoints./.device"); !ok {
		t.Fatal("device should still resolve")
	}
	if _, ok := c.Value("mountpoints./.size_bytes"); ok {
		t.Fatal("size absent without df")
	}
}

func TestMountpointsAbsent(t *testing.T) {
	if _, ok := (fakeEnv{goos: "darwin"}).collection().Value("mountpoints"); ok {
		t.Fatal("non-linux mountpoints absent")
	}
	if _, ok := (fakeEnv{goos: "linux"}).collection().Value("mountpoints"); ok {
		t.Fatal("no /proc/mounts -> absent")
	}
	// Present file but no valid rows.
	f := fakeEnv{goos: "linux", files: map[string]string{"/proc/mounts": "x y\n"}}
	if _, ok := f.collection().Value("mountpoints"); ok {
		t.Fatal("no valid mounts -> absent")
	}
}

func TestUnescapeMount(t *testing.T) {
	if got := unescapeMount("/plain"); got != "/plain" {
		t.Errorf("plain = %q", got)
	}
	if got := unescapeMount(`/a\040b`); got != "/a b" {
		t.Errorf("octal = %q", got)
	}
	if got := unescapeMount(`/a\09z`); got != `/a\09z` {
		t.Errorf("bad escape preserved = %q", got)
	}
}

func TestFilesystemsLinux(t *testing.T) {
	procFS := "nodev\tsysfs\nnodev\ttmpfs\n\text4\n\txfs\n\text4\n"
	f := fakeEnv{goos: "linux", files: map[string]string{"/proc/filesystems": procFS}}
	c := f.collection()
	if v, _ := c.Value("filesystems"); v != "ext4,xfs" {
		t.Fatalf("filesystems = %v", v)
	}
}

func TestFilesystemsAbsent(t *testing.T) {
	if _, ok := (fakeEnv{goos: "windows"}).collection().Value("filesystems"); ok {
		t.Fatal("non-linux filesystems absent")
	}
	if _, ok := (fakeEnv{goos: "linux"}).collection().Value("filesystems"); ok {
		t.Fatal("no /proc/filesystems absent")
	}
	// Present but only nodev entries -> empty list -> absent.
	f := fakeEnv{goos: "linux", files: map[string]string{"/proc/filesystems": "nodev\tsysfs\n\n"}}
	if _, ok := f.collection().Value("filesystems"); ok {
		t.Fatal("only-nodev -> absent")
	}
}

func TestDisksLinux(t *testing.T) {
	f := fakeEnv{goos: "linux",
		dirs: map[string][]dirEntry{"/sys/block": {{Name: "sda"}, {Name: "loop0"}, {Name: "ram0"}}},
		files: map[string]string{
			"/sys/block/sda/size":          "2048\n",
			"/sys/block/sda/device/model":  "Samsung SSD\n",
			"/sys/block/sda/device/vendor": "ATA\n",
		},
	}
	c := f.collection()
	if v, _ := c.Value("disks.sda.size_bytes"); v != uint64(2048*512) {
		t.Fatalf("disk size = %v", v)
	}
	if v, _ := c.Value("disks.sda.model"); v != "Samsung SSD" {
		t.Fatalf("disk model = %v", v)
	}
	if v, _ := c.Value("disks.sda.vendor"); v != "ATA" {
		t.Fatalf("disk vendor = %v", v)
	}
	// loop/ram devices are excluded.
	if _, ok := c.Value("disks.loop0"); ok {
		t.Fatal("loop0 should be excluded")
	}
}

func TestDisksAbsent(t *testing.T) {
	if _, ok := (fakeEnv{goos: "darwin"}).collection().Value("disks"); ok {
		t.Fatal("non-linux disks absent")
	}
	if _, ok := (fakeEnv{goos: "linux", dirErr: map[string]bool{"/sys/block": true}}).collection().Value("disks"); ok {
		t.Fatal("readDir error -> absent")
	}
	// Only synthetic devices -> empty -> absent.
	f := fakeEnv{goos: "linux", dirs: map[string][]dirEntry{"/sys/block": {{Name: "loop0"}}}}
	if _, ok := f.collection().Value("disks"); ok {
		t.Fatal("only loop -> absent")
	}
}
