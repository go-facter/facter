// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "testing"

func TestSELinuxEnforcing(t *testing.T) {
	c := (fakeEnv{goos: "linux", files: map[string]string{
		"/sys/fs/selinux/enforce":    "1",
		"/sys/fs/selinux/policyvers": "31\n",
		"/etc/selinux/config":        "# comment\n\nSELINUX=enforcing\nSELINUXTYPE=targeted\nbogusline\n",
	}}).collection()
	cases := map[string]any{
		"selinux.enabled":        true,
		"selinux.enforced":       true,
		"selinux.current_mode":   "enforcing",
		"selinux.policy_version": "31",
		"selinux.config_mode":    "enforcing",
		"selinux.config_policy":  "targeted",
	}
	for path, want := range cases {
		if v, ok := c.Value(path); !ok || v != want {
			t.Errorf("%s = %v (ok=%v), want %v", path, v, ok, want)
		}
	}
}

func TestSELinuxPermissive(t *testing.T) {
	c := (fakeEnv{goos: "linux", files: map[string]string{
		"/sys/fs/selinux/enforce":    "0",
		"/sys/fs/selinux/policyvers": "   \n", // whitespace: policy_version omitted
	}}).collection()
	if v, _ := c.Value("selinux.enforced"); v != false {
		t.Errorf("enforced = %v", v)
	}
	if v, _ := c.Value("selinux.current_mode"); v != "permissive" {
		t.Errorf("current_mode = %v", v)
	}
	if _, ok := c.Value("selinux.policy_version"); ok {
		t.Error("blank policyvers should be omitted")
	}
	if _, ok := c.Value("selinux.config_mode"); ok {
		t.Error("no config file -> config_mode omitted")
	}
}

func TestSELinuxDisabled(t *testing.T) {
	// selinuxfs not mounted: enabled=false only.
	c := (fakeEnv{goos: "linux"}).collection()
	if v, _ := c.Value("selinux.enabled"); v != false {
		t.Errorf("enabled = %v", v)
	}
	if _, ok := c.Value("selinux.enforced"); ok {
		t.Error("enforced should be absent when disabled")
	}
}

func TestSELinuxNonLinux(t *testing.T) {
	if _, ok := (fakeEnv{goos: "darwin"}).collection().Value("selinux"); ok {
		t.Fatal("selinux absent off Linux")
	}
}

func TestParseSELinuxConfig(t *testing.T) {
	mode, policy := parseSELinuxConfig("SELINUX=permissive\nSELINUXTYPE=mls\n")
	if mode != "permissive" || policy != "mls" {
		t.Fatalf("parsed = %q %q", mode, policy)
	}
}

func TestModeLabel(t *testing.T) {
	if modeLabel(true) != "enforcing" || modeLabel(false) != "permissive" {
		t.Fatal("modeLabel")
	}
}
