// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "testing"

func TestCloudAWSFromDMI(t *testing.T) {
	c := (fakeEnv{goos: "linux", files: map[string]string{
		"/sys/class/dmi/id/sys_vendor": "Amazon EC2\n",
	}}).collection()
	if v, _ := c.Value("cloud.provider"); v != "aws" {
		t.Errorf("provider = %v", v)
	}
}

func TestCloudGCEFromDMI(t *testing.T) {
	c := (fakeEnv{goos: "linux", files: map[string]string{
		"/sys/class/dmi/id/product_name": "Google Compute Engine\n",
	}}).collection()
	if v, _ := c.Value("cloud.provider"); v != "gce" {
		t.Errorf("provider = %v", v)
	}
}

func TestCloudAzureFromAssetTag(t *testing.T) {
	c := (fakeEnv{goos: "linux", files: map[string]string{
		"/sys/class/dmi/id/chassis_asset_tag": azureAssetTag + "\n",
		"/sys/class/dmi/id/sys_vendor":        "Microsoft Corporation\n",
	}}).collection()
	if v, _ := c.Value("cloud.provider"); v != "azure" {
		t.Errorf("provider = %v", v)
	}
}

// hypervisorCPU is a /proc/cpuinfo fixture whose hypervisor flag marks the host
// virtual (is_virtual true) without leaving any provider-specific DMI hint, so a
// metadata probe is gated on but the provider still comes from the IMDS answer.
const hypervisorCPU = "flags\t: fpu vme hypervisor\n"

func TestCloudAWSFromMetadata(t *testing.T) {
	// No DMI cloud hint, but the host is virtual and the EC2 IMDS answers.
	c := (fakeEnv{goos: "linux",
		files: map[string]string{"/proc/cpuinfo": hypervisorCPU},
		https: map[string]string{ec2MetadataBase: "ami-id\n"},
	}).collection()
	if v, _ := c.Value("cloud.provider"); v != "aws" {
		t.Errorf("provider = %v", v)
	}
}

func TestCloudAzureFromMetadata(t *testing.T) {
	c := (fakeEnv{goos: "linux",
		files: map[string]string{"/proc/cpuinfo": hypervisorCPU},
		https: map[string]string{azureMetadata: `{"compute":{}}`},
	}).collection()
	if v, _ := c.Value("cloud.provider"); v != "azure" {
		t.Errorf("provider = %v", v)
	}
}

func TestCloudAbsent(t *testing.T) {
	if _, ok := (fakeEnv{goos: "linux"}).collection().Value("cloud"); ok {
		t.Fatal("no cloud hints -> absent")
	}
}

// TestCloudVirtualNoProviderAbsent covers a virtual, non-cloud host: it is
// plausibly cloud (so it probes) but neither metadata endpoint answers, so the
// provider stays unidentified and the cloud fact is absent.
func TestCloudVirtualNoProviderAbsent(t *testing.T) {
	c := (fakeEnv{goos: "linux",
		files: map[string]string{"/proc/cpuinfo": hypervisorCPU},
	}).collection()
	if _, ok := c.Value("cloud"); ok {
		t.Fatal("virtual host with no metadata answer -> cloud absent")
	}
}

func TestEC2Metadata(t *testing.T) {
	c := (fakeEnv{goos: "linux",
		files: map[string]string{"/proc/cpuinfo": hypervisorCPU},
		https: map[string]string{
			ec2MetadataBase:              "ami-id\nhostname\niam/\nlocal-ipv4\n",
			ec2MetadataBase + "ami-id":   "ami-0abc\n",
			ec2MetadataBase + "hostname": "ip-10-0-0-1\n",
			// local-ipv4 deliberately unreachable -> that leaf is skipped.
		}}).collection()
	v, ok := c.Value("ec2_metadata")
	if !ok {
		t.Fatal("ec2_metadata absent")
	}
	m := v.(map[string]any)
	if m["ami-id"] != "ami-0abc" || m["hostname"] != "ip-10-0-0-1" {
		t.Errorf("metadata = %v", m)
	}
	if _, present := m["iam/"]; present {
		t.Error("subtree entry should be skipped")
	}
	if _, present := m["local-ipv4"]; present {
		t.Error("unreachable leaf should be skipped")
	}
}

func TestEC2MetadataAbsent(t *testing.T) {
	// Virtual host (so the crawl is not gated out), but the IMDS index is
	// unreachable, so the fact is absent.
	c := (fakeEnv{goos: "linux",
		files: map[string]string{"/proc/cpuinfo": hypervisorCPU},
	}).collection()
	if _, ok := c.Value("ec2_metadata"); ok {
		t.Fatal("no IMDS -> absent")
	}
}

func TestEC2MetadataEmpty(t *testing.T) {
	// Index reachable but only contains a sub-tree: no leaf keys -> absent.
	c := (fakeEnv{goos: "linux",
		files: map[string]string{"/proc/cpuinfo": hypervisorCPU},
		https: map[string]string{ec2MetadataBase: "iam/\n"},
	}).collection()
	if _, ok := c.Value("ec2_metadata"); ok {
		t.Fatal("index of only subtrees -> absent")
	}
}

// TestCloudNoNetworkOnNonCloudHost is the regression guard for the probe-gating
// fix: a bare-metal, non-virtual host must resolve cloud and ec2_metadata to
// absent while making ZERO metadata HTTP calls, so it never pays the ~1.2s of
// link-local timeouts that dominated the old run.
func TestCloudNoNetworkOnNonCloudHost(t *testing.T) {
	e := (fakeEnv{goos: "linux"}).env()
	calls := 0
	inner := e.httpGet
	e.httpGet = func(url string, h map[string]string) (string, bool) {
		calls++
		return inner(url, h)
	}
	c := newWithEnv(e)
	if _, ok := c.Value("cloud"); ok {
		t.Error("cloud should be absent on a non-cloud host")
	}
	if _, ok := c.Value("ec2_metadata"); ok {
		t.Error("ec2_metadata should be absent on a non-cloud host")
	}
	if calls != 0 {
		t.Errorf("non-cloud host made %d metadata HTTP calls, want 0", calls)
	}
}

// TestCloudProbesWhenVirtual is the companion guard: a plausibly-cloud host
// still probes the metadata endpoint through the injectable HTTP seam.
func TestCloudProbesWhenVirtual(t *testing.T) {
	e := (fakeEnv{goos: "linux",
		files: map[string]string{"/proc/cpuinfo": hypervisorCPU},
		https: map[string]string{ec2MetadataBase: "ami-id\n"},
	}).env()
	calls := 0
	inner := e.httpGet
	e.httpGet = func(url string, h map[string]string) (string, bool) {
		calls++
		return inner(url, h)
	}
	c := newWithEnv(e)
	if v, _ := c.Value("cloud.provider"); v != "aws" {
		t.Errorf("provider = %v", v)
	}
	if calls == 0 {
		t.Error("a plausibly-cloud host should probe the metadata endpoint")
	}
}

func TestMapString(t *testing.T) {
	m := map[string]any{"s": "x", "n": 3}
	if mapString(m, "s") != "x" {
		t.Error("string member")
	}
	if mapString(m, "n") != "" {
		t.Error("non-string member should be empty")
	}
	if mapString(m, "absent") != "" {
		t.Error("absent member should be empty")
	}
}
