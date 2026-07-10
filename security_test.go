// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "testing"

func TestFIPSEnabled(t *testing.T) {
	on := (fakeEnv{goos: "linux", files: map[string]string{"/proc/sys/crypto/fips_enabled": "1\n"}}).collection()
	if v, _ := on.Value("fips_enabled"); v != true {
		t.Errorf("fips on = %v", v)
	}
	off := (fakeEnv{goos: "linux", files: map[string]string{"/proc/sys/crypto/fips_enabled": "0\n"}}).collection()
	if v, _ := off.Value("fips_enabled"); v != false {
		t.Errorf("fips off = %v", v)
	}
	// No source file: reported false, still present.
	noFile := (fakeEnv{goos: "linux"}).collection()
	if v, ok := noFile.Value("fips_enabled"); !ok || v != false {
		t.Errorf("fips no-file = %v (ok=%v)", v, ok)
	}
	// Non-Linux: false.
	if v, _ := (fakeEnv{goos: "darwin"}).collection().Value("fips_enabled"); v != false {
		t.Errorf("fips darwin = %v", v)
	}
}

func TestAIOAgentVersion(t *testing.T) {
	c := (fakeEnv{files: map[string]string{"/opt/puppetlabs/puppet/VERSION": "8.4.0\n"}}).collection()
	if v, _ := c.Value("aio_agent_version"); v != "8.4.0" {
		t.Errorf("aio = %v", v)
	}
	// Blank file -> absent.
	blank := (fakeEnv{files: map[string]string{"/opt/puppetlabs/puppet/VERSION": "  \n"}}).collection()
	if _, ok := blank.Value("aio_agent_version"); ok {
		t.Error("blank VERSION -> absent")
	}
	// Missing -> absent.
	if _, ok := (fakeEnv{}).collection().Value("aio_agent_version"); ok {
		t.Error("missing VERSION -> absent")
	}
}

func TestAugeasVersion(t *testing.T) {
	c := (fakeEnv{cmds: map[string]string{"augparse --version": "augparse 1.12.0 <http://augeas.net/>\n"}}).collection()
	if v, _ := c.Value("augeasversion"); v != "1.12.0" {
		t.Errorf("augeas = %v", v)
	}
	// Output without a version token -> absent.
	noVer := (fakeEnv{cmds: map[string]string{"augparse --version": "augparse unknown\n"}}).collection()
	if _, ok := noVer.Value("augeasversion"); ok {
		t.Error("no version token -> absent")
	}
	// Command unavailable -> absent.
	if _, ok := (fakeEnv{}).collection().Value("augeasversion"); ok {
		t.Error("no augparse -> absent")
	}
}

func TestEnvWindowsInstalldir(t *testing.T) {
	c := (fakeEnv{goos: "windows", envv: map[string]string{"FACTER_env_windows_installdir": `C:\Program Files\Puppet Labs\Puppet`}}).collection()
	if v, _ := c.Value("env_windows_installdir"); v != `C:\Program Files\Puppet Labs\Puppet` {
		t.Errorf("installdir = %v", v)
	}
	// Set but empty -> absent.
	empty := (fakeEnv{goos: "windows", envv: map[string]string{"FACTER_env_windows_installdir": ""}}).collection()
	if _, ok := empty.Value("env_windows_installdir"); ok {
		t.Error("empty installdir -> absent")
	}
	// Unset -> absent.
	if _, ok := (fakeEnv{goos: "windows"}).collection().Value("env_windows_installdir"); ok {
		t.Error("unset installdir -> absent")
	}
	// Non-Windows -> absent.
	if _, ok := (fakeEnv{goos: "linux"}).collection().Value("env_windows_installdir"); ok {
		t.Error("non-windows -> absent")
	}
}

func TestVersionToken(t *testing.T) {
	if got := versionToken("tool 2.3.4 extra"); got != "2.3.4" {
		t.Errorf("token = %q", got)
	}
	if got := versionToken("no digits here"); got != "" {
		t.Errorf("no token = %q", got)
	}
}
