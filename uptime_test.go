// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"strconv"
	"testing"
	"time"
)

func TestUptimeLinux(t *testing.T) {
	f := fakeEnv{goos: "linux", files: map[string]string{"/proc/uptime": "90061.50 12345.6\n"}}
	c := f.collection()
	if v, _ := c.Value("system_uptime.seconds"); v != int64(90061) {
		t.Fatalf("seconds = %v", v)
	}
	if v, _ := c.Value("system_uptime.days"); v != int64(1) {
		t.Fatalf("days = %v", v)
	}
	if v, _ := c.Value("system_uptime.uptime"); v != "1 day" {
		t.Fatalf("uptime = %v", v)
	}
	if v, _ := c.Value("uptime_seconds"); v != int64(90061) {
		t.Fatalf("legacy uptime_seconds = %v", v)
	}
}

func TestUptimeLinuxAbsentAndBad(t *testing.T) {
	if _, ok := (fakeEnv{goos: "linux"}).collection().Value("system_uptime"); ok {
		t.Fatal("no /proc/uptime -> absent")
	}
	f := fakeEnv{goos: "linux", files: map[string]string{"/proc/uptime": "notanumber\n"}}
	if _, ok := f.collection().Value("system_uptime"); ok {
		t.Fatal("bad uptime -> absent")
	}
	f2 := fakeEnv{goos: "linux", files: map[string]string{"/proc/uptime": "\n"}}
	if _, ok := f2.collection().Value("system_uptime"); ok {
		t.Fatal("empty uptime -> absent")
	}
}

func TestUptimeDarwin(t *testing.T) {
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	boot := now.Add(-3 * time.Hour).Unix()
	f := fakeEnv{goos: "darwin", now: now, cmds: map[string]string{
		"sysctl -n kern.boottime": "{ sec = " + itoaTest(boot) + ", usec = 0 } " + now.String(),
	}}
	c := f.collection()
	if v, _ := c.Value("system_uptime.seconds"); v != int64(3*3600) {
		t.Fatalf("darwin uptime seconds = %v", v)
	}
	if v, _ := c.Value("system_uptime.uptime"); v != "3:00 hours" {
		t.Fatalf("darwin uptime str = %v", v)
	}
}

func TestUptimeDarwinAbsent(t *testing.T) {
	// No sysctl.
	if _, ok := (fakeEnv{goos: "darwin"}).collection().Value("system_uptime"); ok {
		t.Fatal("darwin no boottime -> absent")
	}
	// Unparseable boottime.
	f := fakeEnv{goos: "darwin", cmds: map[string]string{"sysctl -n kern.boottime": "garbage"}}
	if _, ok := f.collection().Value("system_uptime"); ok {
		t.Fatal("darwin bad boottime -> absent")
	}
	// boottime in the future -> absent.
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour).Unix()
	f2 := fakeEnv{goos: "darwin", now: now, cmds: map[string]string{
		"sysctl -n kern.boottime": "{ sec = " + itoaTest(future) + ", usec = 0 }",
	}}
	if _, ok := f2.collection().Value("system_uptime"); ok {
		t.Fatal("future boottime -> absent")
	}
}

func TestUptimeUnsupportedOS(t *testing.T) {
	if _, ok := (fakeEnv{goos: "windows"}).collection().Value("system_uptime"); ok {
		t.Fatal("windows uptime -> absent")
	}
}

func TestParseBoottimeEdge(t *testing.T) {
	if _, ok := parseBoottime("no sec here"); ok {
		t.Error("missing sec")
	}
	if _, ok := parseBoottime("sec = "); ok {
		t.Error("empty sec")
	}
	if v, ok := parseBoottime("{ sec = 42, usec = 0 }"); !ok || v != 42 {
		t.Errorf("valid = %d %v", v, ok)
	}
	if _, ok := parseBoottime("{ sec = 999999999999999999999, usec = 0 }"); ok {
		t.Error("overflowing sec should fail")
	}
}

func TestFormatUptime(t *testing.T) {
	cases := map[int64]string{
		2 * 86400:     "2 days",
		1 * 86400:     "1 day",
		3*3600 + 4*60: "3:04 hours",
		1 * 3600:      "1:00 hours",
		5 * 60:        "5 minutes",
		1 * 60:        "1 minute",
		30:            "0 minutes",
	}
	for secs, want := range cases {
		if got := formatUptime(secs); got != want {
			t.Errorf("formatUptime(%d) = %q, want %q", secs, got, want)
		}
	}
}

// itoaTest formats an int64 for building sysctl boottime fixtures.
func itoaTest(n int64) string { return strconv.FormatInt(n, 10) }

func TestPad2(t *testing.T) {
	if pad2(3) != "03" {
		t.Errorf("pad2(3) = %q", pad2(3))
	}
	if pad2(42) != "42" {
		t.Errorf("pad2(42) = %q", pad2(42))
	}
}
