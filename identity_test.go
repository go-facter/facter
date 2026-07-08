// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "testing"

func TestIdentity(t *testing.T) {
	f := fakeEnv{
		goos: "linux",
		user: userInfo{Username: "deploy", UID: "1001", GID: "1001", Group: "deploy"},
		euid: 0,
	}
	c := f.collection()
	if v, _ := c.Value("identity.user"); v != "deploy" {
		t.Fatalf("user = %v", v)
	}
	if v, _ := c.Value("identity.uid"); v != 1001 {
		t.Fatalf("uid = %v (%T)", v, v)
	}
	if v, _ := c.Value("identity.privileged"); v != true {
		t.Fatalf("privileged = %v", v)
	}
	if v, _ := c.Value("id"); v != "deploy" {
		t.Fatalf("legacy id = %v", v)
	}
}

func TestIdentityUnprivileged(t *testing.T) {
	f := fakeEnv{user: userInfo{Username: "u", UID: "1000", GID: "1000"}, euid: 1000}
	if v, _ := f.collection().Value("identity.privileged"); v != false {
		t.Fatalf("privileged = %v", v)
	}
}

func TestIdentityWindowsSID(t *testing.T) {
	f := fakeEnv{goos: "windows", user: userInfo{Username: "Admin", UID: "S-1-5-21", GID: "S-1-5-32", Group: "Administrators"}, euid: -1}
	c := f.collection()
	if v, _ := c.Value("identity.uid"); v != "S-1-5-21" {
		t.Fatalf("sid uid = %v", v)
	}
	if v, _ := c.Value("identity.privileged"); v != false {
		t.Fatalf("windows privileged = %v", v)
	}
}

func TestIdentityError(t *testing.T) {
	if _, ok := (fakeEnv{userErr: true}).collection().Value("identity"); ok {
		t.Fatal("user error -> identity absent")
	}
}

func TestNumericID(t *testing.T) {
	if v := numericID("42"); v != 42 {
		t.Errorf("numeric = %v", v)
	}
	if v := numericID("S-1-5"); v != "S-1-5" {
		t.Errorf("sid = %v", v)
	}
}
