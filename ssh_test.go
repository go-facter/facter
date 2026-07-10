// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "testing"

// Real OpenSSH public keys with fingerprints computed independently (ssh-keygen +
// shasum), so these tests differentially verify the pure-Go fingerprinting.
const (
	edPub = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJT33HI2yr593JgHITVzZYLNv6oIw+bLYm22iwyX7wn7 root@host\n"
	edKey = "AAAAC3NzaC1lZDI1NTE5AAAAIJT33HI2yr593JgHITVzZYLNv6oIw+bLYm22iwyX7wn7"
	ecPub = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBH9e1qZEg4cobsCNvWFKOma9SPmSkhNnJ+bjAxghNEui1m/XH/UXeADLbT46C8PWKISQucmb9f9rVfNWvOelHjY= root@host\n"
)

func TestSSHStructured(t *testing.T) {
	c := (fakeEnv{goos: "linux", files: map[string]string{
		"/etc/ssh/ssh_host_ed25519_key.pub": edPub,
		"/etc/ssh/ssh_host_ecdsa_key.pub":   ecPub,
	}}).collection()

	if v, _ := c.Value("ssh.ed25519.type"); v != "ssh-ed25519" {
		t.Errorf("ed25519 type = %v", v)
	}
	if v, _ := c.Value("ssh.ed25519.key"); v != edKey {
		t.Errorf("ed25519 key = %v", v)
	}
	if v, _ := c.Value("ssh.ed25519.fingerprints.sha1"); v != "SSHFP 4 1 c884b501486f7ef9389f3d9b2d09770f007a9bd1" {
		t.Errorf("ed25519 sha1 = %v", v)
	}
	if v, _ := c.Value("ssh.ed25519.fingerprints.sha256"); v != "SSHFP 4 2 159f2815d9e55757f79721d640bc9d329ec182973e71b3cf97359c668a4a3ce9" {
		t.Errorf("ed25519 sha256 = %v", v)
	}
	if v, _ := c.Value("ssh.ecdsa.type"); v != "ecdsa-sha2-nistp256" {
		t.Errorf("ecdsa type = %v", v)
	}
	if v, _ := c.Value("ssh.ecdsa.fingerprints.sha1"); v != "SSHFP 3 1 df1f66cd10f8504f346921db4ed1005c2bfd2da9" {
		t.Errorf("ecdsa sha1 = %v", v)
	}
}

func TestSSHLegacyFacts(t *testing.T) {
	c := (fakeEnv{goos: "linux", files: map[string]string{
		"/etc/ssh/ssh_host_ed25519_key.pub": edPub,
	}}).collection()
	if v, _ := c.Value("sshed25519key"); v != edKey {
		t.Errorf("sshed25519key = %v", v)
	}
	want := "SSHFP 4 1 c884b501486f7ef9389f3d9b2d09770f007a9bd1\nSSHFP 4 2 159f2815d9e55757f79721d640bc9d329ec182973e71b3cf97359c668a4a3ce9"
	if v, _ := c.Value("sshfp_ed25519"); v != want {
		t.Errorf("sshfp_ed25519 = %v", v)
	}
	// A key type that is not present yields absent legacy facts.
	if _, ok := c.Value("sshrsakey"); ok {
		t.Error("sshrsakey should be absent")
	}
	if _, ok := c.Value("sshfp_rsa"); ok {
		t.Error("sshfp_rsa should be absent")
	}
}

func TestSSHAbsent(t *testing.T) {
	if _, ok := (fakeEnv{goos: "linux"}).collection().Value("ssh"); ok {
		t.Fatal("no host keys -> ssh absent")
	}
}

func TestSSHMalformedAndBadBase64(t *testing.T) {
	c := (fakeEnv{goos: "linux", files: map[string]string{
		"/etc/ssh/ssh_host_dsa_key.pub": "onlyonefield\n",               // parsePubKey fails
		"/etc/ssh/ssh_host_rsa_key.pub": "ssh-rsa !!!not-base64!!! x\n", // base64 decode fails
	}}).collection()
	if _, ok := c.Value("ssh"); ok {
		t.Fatal("all keys unparseable -> ssh absent")
	}
}

func TestParsePubKey(t *testing.T) {
	// Leading blank/short lines are skipped; the first full line wins.
	typ, blob, ok := parsePubKey("\nshort\nssh-rsa AAAA comment\n")
	if !ok || typ != "ssh-rsa" || blob != "AAAA" {
		t.Fatalf("parsePubKey = %q %q %v", typ, blob, ok)
	}
	if _, _, ok := parsePubKey("nofields\n"); ok {
		t.Error("single-field line should not parse")
	}
}
