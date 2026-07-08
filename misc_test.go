// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"testing"
	"time"
)

func TestTimezone(t *testing.T) {
	f := fakeEnv{now: time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)}
	if v, _ := f.collection().Value("timezone"); v != "UTC" {
		t.Fatalf("timezone = %v", v)
	}
}

func TestTimezoneEmpty(t *testing.T) {
	// A zone with no abbreviation yields no fact.
	f := fakeEnv{now: time.Date(2026, 7, 7, 12, 0, 0, 0, time.FixedZone("", 0))}
	if _, ok := f.collection().Value("timezone"); ok {
		t.Fatal("empty zone -> absent")
	}
}

func TestPath(t *testing.T) {
	f := fakeEnv{envv: map[string]string{"PATH": "/usr/bin:/bin"}}
	if v, _ := f.collection().Value("path"); v != "/usr/bin:/bin" {
		t.Fatalf("path = %v", v)
	}
	if _, ok := (fakeEnv{}).collection().Value("path"); ok {
		t.Fatal("no PATH -> absent")
	}
}

func TestFacterVersion(t *testing.T) {
	if v, _ := (fakeEnv{}).collection().Value("facterversion"); v != Version {
		t.Fatalf("facterversion = %v", v)
	}
}
