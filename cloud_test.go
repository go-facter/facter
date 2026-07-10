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

func TestCloudAWSFromMetadata(t *testing.T) {
	// No DMI hints; the EC2 IMDS answers.
	c := (fakeEnv{goos: "linux", https: map[string]string{
		ec2MetadataBase: "ami-id\n",
	}}).collection()
	if v, _ := c.Value("cloud.provider"); v != "aws" {
		t.Errorf("provider = %v", v)
	}
}

func TestCloudAzureFromMetadata(t *testing.T) {
	c := (fakeEnv{goos: "linux", https: map[string]string{
		azureMetadata: `{"compute":{}}`,
	}}).collection()
	if v, _ := c.Value("cloud.provider"); v != "azure" {
		t.Errorf("provider = %v", v)
	}
}

func TestCloudAbsent(t *testing.T) {
	if _, ok := (fakeEnv{goos: "linux"}).collection().Value("cloud"); ok {
		t.Fatal("no cloud hints -> absent")
	}
}

func TestEC2Metadata(t *testing.T) {
	c := (fakeEnv{goos: "linux", https: map[string]string{
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
	if _, ok := (fakeEnv{goos: "linux"}).collection().Value("ec2_metadata"); ok {
		t.Fatal("no IMDS -> absent")
	}
}

func TestEC2MetadataEmpty(t *testing.T) {
	// Index reachable but only contains a sub-tree: no leaf keys -> absent.
	c := (fakeEnv{goos: "linux", https: map[string]string{
		ec2MetadataBase: "iam/\n",
	}}).collection()
	if _, ok := c.Value("ec2_metadata"); ok {
		t.Fatal("index of only subtrees -> absent")
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
