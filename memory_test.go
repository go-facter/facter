// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "testing"

const meminfoSample = `MemTotal:       16000 kB
MemAvailable:    8000 kB
SwapTotal:       4000 kB
SwapFree:        1000 kB
Bogus 5
Empty:
NoColonLine
`

func TestMemoryLinux(t *testing.T) {
	f := fakeEnv{goos: "linux", files: map[string]string{"/proc/meminfo": meminfoSample}}
	c := f.collection()
	if v, _ := c.Value("memory.system.total_bytes"); v != uint64(16000*1024) {
		t.Fatalf("total_bytes = %v", v)
	}
	if v, _ := c.Value("memory.system.used_bytes"); v != uint64(8000*1024) {
		t.Fatalf("used_bytes = %v", v)
	}
	if v, _ := c.Value("memory.swap.total_bytes"); v != uint64(4000*1024) {
		t.Fatalf("swap total = %v", v)
	}
	// legacy human strings
	if v, _ := c.Value("memorysize"); v == "" {
		t.Fatalf("memorysize empty")
	}
	if v, _ := c.Value("swapfree"); v == "" {
		t.Fatalf("swapfree empty")
	}
}

func TestMemoryLinuxMemFreeFallbackNoSwap(t *testing.T) {
	f := fakeEnv{goos: "linux", files: map[string]string{"/proc/meminfo": "MemTotal: 100 kB\nMemFree: 40 kB\n"}}
	c := f.collection()
	if v, _ := c.Value("memory.system.available_bytes"); v != uint64(40*1024) {
		t.Fatalf("available via MemFree = %v", v)
	}
	if _, ok := c.Value("memory.swap"); ok {
		t.Fatal("no swap expected")
	}
}

func TestMemoryLinuxNoTotal(t *testing.T) {
	f := fakeEnv{goos: "linux", files: map[string]string{"/proc/meminfo": "Foo: 1 kB\n"}}
	if _, ok := f.collection().Value("memory"); ok {
		t.Fatal("memory should be absent without MemTotal")
	}
}

func TestMemoryUnsupportedOS(t *testing.T) {
	if _, ok := (fakeEnv{goos: "windows"}).collection().Value("memory"); ok {
		t.Fatal("windows memory should be absent")
	}
	if _, ok := (fakeEnv{goos: "plan9"}).collection().Value("memory"); ok {
		t.Fatal("generic memory should be absent")
	}
}

func TestMemoryDarwin(t *testing.T) {
	vmstat := `Mach Virtual Memory Statistics: (page size of 16384 bytes)
Pages free:                          100.
Pages inactive:                       50.
Pages speculative:                    10.
`
	f := fakeEnv{goos: "darwin", cmds: map[string]string{
		"sysctl -n hw.memsize":   "17179869184\n",
		"vm_stat":                vmstat,
		"sysctl -n vm.swapusage": "total = 2048.00M  used = 512.00M  free = 1536.00M\n",
	}}
	c := f.collection()
	if v, _ := c.Value("memory.system.total_bytes"); v != uint64(17179869184) {
		t.Fatalf("darwin total = %v", v)
	}
	// available = (100+50+10)*16384
	if v, _ := c.Value("memory.system.available_bytes"); v != uint64(160*16384) {
		t.Fatalf("darwin available = %v", v)
	}
	if v, _ := c.Value("memory.swap.total_bytes"); v != uint64(2048*1024*1024) {
		t.Fatalf("darwin swap = %v", v)
	}
}

func TestMemoryDarwinNoVMStatNoSwap(t *testing.T) {
	// No vm_stat: available defaults to total. No swapusage: no swap.
	f := fakeEnv{goos: "darwin", cmds: map[string]string{"sysctl -n hw.memsize": "1024\n"}}
	c := f.collection()
	if v, _ := c.Value("memory.system.available_bytes"); v != uint64(1024) {
		t.Fatalf("available default = %v", v)
	}
	if _, ok := c.Value("memory.swap"); ok {
		t.Fatal("no swap")
	}
}

func TestMemoryDarwinNoMemsize(t *testing.T) {
	if _, ok := (fakeEnv{goos: "darwin"}).collection().Value("memory"); ok {
		t.Fatal("darwin no memsize -> absent")
	}
}

func TestMemoryDarwinAvailClamp(t *testing.T) {
	// vm_stat reports more free than total -> clamped to total.
	vmstat := "page size of 4096 bytes\nPages free: 1000000.\n"
	f := fakeEnv{goos: "darwin", cmds: map[string]string{
		"sysctl -n hw.memsize": "4096\n",
		"vm_stat":              vmstat,
	}}
	c := f.collection()
	if v, _ := c.Value("memory.system.available_bytes"); v != uint64(4096) {
		t.Fatalf("clamped available = %v", v)
	}
}

func TestParseVMStatDefaultPageSize(t *testing.T) {
	// No "page size of" line: default 4096 used.
	got := parseVMStatFree("Pages free: 2.\n")
	if got != uint64(2*4096) {
		t.Fatalf("default page size = %d", got)
	}
}

func TestParseSwapUsageMissingTotal(t *testing.T) {
	if _, ok := parseSwapUsage("used = 1.00M"); ok {
		t.Fatal("missing total should fail")
	}
}

func TestParseHumanSize(t *testing.T) {
	cases := map[string]uint64{
		"1024.00M": 1024 * 1024 * 1024,
		"2.00G":    2 * 1024 * 1024 * 1024,
		"512K":     512 * 1024,
		"1T":       1024 * 1024 * 1024 * 1024,
		"100":      100,
		"":         0,
		"bad":      0,
	}
	for in, want := range cases {
		if got := parseHumanSize(in); got != want {
			t.Errorf("parseHumanSize(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestHumanBytes(t *testing.T) {
	if got := humanBytes(512); got != "512 bytes" {
		t.Errorf("bytes = %q", got)
	}
	if got := humanBytes(1536); got != "1.50 KiB" {
		t.Errorf("kib = %q", got)
	}
	if got := humanBytes(3 * 1024 * 1024 * 1024); got != "3.00 GiB" {
		t.Errorf("gib = %q", got)
	}
}

func TestCapacity(t *testing.T) {
	if got := capacity(1, 0); got != "0.00%" {
		t.Errorf("zero total = %q", got)
	}
	if got := capacity(1, 2); got != "50.00%" {
		t.Errorf("half = %q", got)
	}
}

func TestNewMemStatClamp(t *testing.T) {
	m := newMemStat(100, 200)
	if m.available != 100 || m.used != 0 {
		t.Fatalf("clamp: %+v", m)
	}
}

func TestAtou64(t *testing.T) {
	if atou64("42") != 42 || atou64("x") != 0 {
		t.Fatal("atou64")
	}
}
