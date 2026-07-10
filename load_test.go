// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "testing"

func TestLoadAveragesLinux(t *testing.T) {
	c := (fakeEnv{goos: "linux", files: map[string]string{
		"/proc/loadavg": "0.15 0.25 0.35 1/234 5678\n",
	}}).collection()
	if v, _ := c.Value("load_averages.1m"); v != 0.15 {
		t.Errorf("1m = %v", v)
	}
	if v, _ := c.Value("load_averages.5m"); v != 0.25 {
		t.Errorf("5m = %v", v)
	}
	if v, _ := c.Value("load_averages.15m"); v != 0.35 {
		t.Errorf("15m = %v", v)
	}
}

func TestLoadAveragesLinuxAbsent(t *testing.T) {
	if _, ok := (fakeEnv{goos: "linux"}).collection().Value("load_averages"); ok {
		t.Fatal("no /proc/loadavg -> absent")
	}
}

func TestLoadAveragesLinuxMalformed(t *testing.T) {
	c := (fakeEnv{goos: "linux", files: map[string]string{"/proc/loadavg": "only two\n"}}).collection()
	if _, ok := c.Value("load_averages"); ok {
		t.Fatal("malformed loadavg -> absent")
	}
}

func TestLoadAveragesDarwin(t *testing.T) {
	c := (fakeEnv{goos: "darwin", cmds: map[string]string{
		"sysctl -n vm.loadavg": "{ 1.50 1.20 0.90 }\n",
	}}).collection()
	if v, _ := c.Value("load_averages.1m"); v != 1.50 {
		t.Errorf("darwin 1m = %v", v)
	}
	if v, _ := c.Value("load_averages.15m"); v != 0.90 {
		t.Errorf("darwin 15m = %v", v)
	}
}

func TestLoadAveragesDarwinAbsent(t *testing.T) {
	if _, ok := (fakeEnv{goos: "darwin"}).collection().Value("load_averages"); ok {
		t.Fatal("no sysctl -> absent")
	}
}

func TestLoadAveragesUnsupported(t *testing.T) {
	if _, ok := (fakeEnv{goos: "windows"}).collection().Value("load_averages"); ok {
		t.Fatal("windows -> absent")
	}
}

func TestParseLoadAvg(t *testing.T) {
	if _, _, _, ok := parseLoadAvg("1 2"); ok {
		t.Error("too few fields")
	}
	if _, _, _, ok := parseLoadAvg("x 2 3"); ok {
		t.Error("bad 1m")
	}
	if _, _, _, ok := parseLoadAvg("1 x 3"); ok {
		t.Error("bad 5m")
	}
	if _, _, _, ok := parseLoadAvg("1 2 x"); ok {
		t.Error("bad 15m")
	}
	if one, five, fifteen, ok := parseLoadAvg("1 2 3"); !ok || one != 1 || five != 2 || fifteen != 3 {
		t.Errorf("good parse = %v %v %v %v", one, five, fifteen, ok)
	}
}
