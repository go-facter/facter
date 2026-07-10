// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "testing"

// benchEnv is a richly-populated Linux fixture that makes every collector do
// representative parsing work, so the benchmarks measure fact-resolution cost
// rather than the cost of missing sources.
func benchEnv() fakeEnv {
	files := map[string]string{
		"/etc/os-release":               "NAME=\"Ubuntu\"\nID=ubuntu\nID_LIKE=debian\nVERSION_ID=\"22.04\"\nVERSION_CODENAME=jammy\nPRETTY_NAME=\"Ubuntu 22.04 LTS\"\n",
		"/proc/cpuinfo":                 cpuinfoExtras,
		"/proc/meminfo":                 meminfoSample,
		"/proc/uptime":                  "123456.78 987654.32\n",
		"/proc/loadavg":                 "0.15 0.25 0.35 1/234 5678\n",
		"/proc/mounts":                  "/dev/sda1 / ext4 rw,relatime 0 0\n/dev/sda2 /home xfs rw 0 0\n",
		"/proc/filesystems":             "nodev\tsysfs\n\text4\n\txfs\n",
		"/proc/sys/crypto/fips_enabled": "0\n",
		"/etc/resolv.conf":              "domain example.com\nnameserver 1.1.1.1\n",
		"/sys/fs/selinux/enforce":       "1",
		"/sys/fs/selinux/policyvers":    "31\n",
		"/etc/selinux/config":           "SELINUX=enforcing\nSELINUXTYPE=targeted\n",
		"/etc/ssh/ssh_host_ed25519_key.pub": edPub,
		"/etc/ssh/ssh_host_ecdsa_key.pub":   ecPub,
		"/opt/puppetlabs/puppet/VERSION":    "8.4.0\n",
	}
	for k, v := range dmiLinuxFiles {
		files[k] = v
	}
	dirs := map[string][]dirEntry{
		"/sys/block": {{Name: "sda"}, {Name: "loop0"}},
	}
	files["/sys/block/sda/size"] = "1000215216\n"
	files["/sys/block/sda/device/model"] = "Samsung SSD\n"
	files["/sys/block/sda/device/vendor"] = "ATA\n"
	return fakeEnv{
		goos:   "linux",
		host:   "web01",
		files:  files,
		dirs:   dirs,
		ifaces: sampleIfaces(),
		cmds: map[string]string{
			"uname -m": "x86_64\n",
			"uname -s": "Linux\n",
			"uname -r": "5.15.0-generic\n",
			"df -P -k": "Filesystem 1024-blocks Used Available Capacity Mounted\n/dev/sda1 100 40 60 40% /\n",
		},
	}
}

// BenchmarkToHashCold resolves the entire fact set from a fresh collection each
// iteration: the cold path, with no cache reuse.
func BenchmarkToHashCold(b *testing.B) {
	e := benchEnv()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.collection().ToHash()
	}
}

// BenchmarkToHashCached resolves the whole fact set once, then re-reads it from
// the per-collection cache each iteration: the warm path.
func BenchmarkToHashCached(b *testing.B) {
	c := benchEnv().collection()
	_ = c.ToHash() // prime the cache
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.ToHash()
	}
}

// BenchmarkValueCold resolves a single structured fact from a fresh collection,
// the cost a manifest pays for its first reference to a fact.
func BenchmarkValueCold(b *testing.B) {
	e := benchEnv()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.collection().Value("networking.ip")
	}
}

// BenchmarkValueCached resolves a single fact repeatedly against a warm cache,
// the cost of every reference after the first.
func BenchmarkValueCached(b *testing.B) {
	c := benchEnv().collection()
	_, _ = c.Value("networking.ip") // prime
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Value("networking.ip")
	}
}
