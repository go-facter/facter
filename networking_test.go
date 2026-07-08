// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "testing"

func sampleIfaces() []ifaceData {
	return []ifaceData{
		{Name: "lo", Loopback: true, Up: true, Addrs: []ipAddr{{IP: "127.0.0.1", Prefix: 8, IsV4: true}}},
		{Name: "eth0", Up: true, MAC: "aa:bb:cc:dd:ee:ff", MTU: 1500, Addrs: []ipAddr{
			{IP: "10.0.0.5", Prefix: 24, IsV4: true},
			{IP: "10.0.0.6", Prefix: 24, IsV4: true},
			{IP: "fe80::1", Prefix: 64, IsV4: false},
		}},
		{Name: "eth1", Up: true},
	}
}

func TestNetworkingFull(t *testing.T) {
	f := fakeEnv{
		goos:   "linux",
		host:   "web01",
		files:  map[string]string{"/etc/resolv.conf": "domain example.com\nnameserver 1.1.1.1\n"},
		ifaces: sampleIfaces(),
	}
	c := f.collection()
	if v, _ := c.Value("networking.hostname"); v != "web01" {
		t.Fatalf("hostname = %v", v)
	}
	if v, _ := c.Value("networking.domain"); v != "example.com" {
		t.Fatalf("domain = %v", v)
	}
	if v, _ := c.Value("networking.fqdn"); v != "web01.example.com" {
		t.Fatalf("fqdn = %v", v)
	}
	if v, _ := c.Value("networking.primary"); v != "eth0" {
		t.Fatalf("primary = %v", v)
	}
	if v, _ := c.Value("networking.ip"); v != "10.0.0.5" {
		t.Fatalf("ip = %v", v)
	}
	if v, _ := c.Value("networking.interfaces.eth0.netmask"); v != "255.255.255.0" {
		t.Fatalf("netmask = %v", v)
	}
	if v, _ := c.Value("networking.interfaces.eth0.network"); v != "10.0.0.0" {
		t.Fatalf("network = %v", v)
	}
	if v, _ := c.Value("networking.interfaces.eth0.ip6"); v != "fe80::1" {
		t.Fatalf("ip6 = %v", v)
	}
	if v, _ := c.Value("networking.interfaces.eth0.mtu"); v != 1500 {
		t.Fatalf("mtu = %v", v)
	}
	// legacy aliases
	if v, _ := c.Value("ipaddress"); v != "10.0.0.5" {
		t.Fatalf("legacy ipaddress = %v", v)
	}
	if v, _ := c.Value("interfaces"); v != "eth0,eth1,lo" {
		t.Fatalf("legacy interfaces = %v", v)
	}
	// bindings present
	if v, _ := c.Value("networking.interfaces.eth0.bindings"); v == nil {
		t.Fatal("bindings missing")
	}
}

func TestNetworkingHostnameWithDot(t *testing.T) {
	f := fakeEnv{goos: "linux", host: "db.corp.local", ifaces: nil}
	c := f.collection()
	if v, _ := c.Value("networking.hostname"); v != "db" {
		t.Fatalf("short = %v", v)
	}
	if v, _ := c.Value("networking.domain"); v != "corp.local" {
		t.Fatalf("domain = %v", v)
	}
}

func TestNetworkingNoDomain(t *testing.T) {
	f := fakeEnv{goos: "linux", host: "solo"}
	c := f.collection()
	if v, _ := c.Value("networking.fqdn"); v != "solo" {
		t.Fatalf("fqdn without domain = %v", v)
	}
	if _, ok := c.Value("networking.domain"); ok {
		t.Fatal("domain should be absent")
	}
}

func TestNetworkingHostnameError(t *testing.T) {
	f := fakeEnv{goos: "linux", hostErr: true}
	c := f.collection()
	if v, _ := c.Value("networking.hostname"); v != "" {
		t.Fatalf("hostname on error = %v", v)
	}
}

func TestNetworkingInterfacesError(t *testing.T) {
	f := fakeEnv{goos: "linux", host: "h", ifErr: true}
	c := f.collection()
	// Hostname facts survive; interface-derived facts are absent.
	if v, _ := c.Value("networking.hostname"); v != "h" {
		t.Fatalf("hostname = %v", v)
	}
	if _, ok := c.Value("networking.ip"); ok {
		t.Fatal("ip should be absent without interfaces")
	}
	if _, ok := c.Value("interfaces"); ok {
		t.Fatal("legacy interfaces should be absent")
	}
}

func TestNetworkingNoPrimary(t *testing.T) {
	// Only a down interface and loopback: no primary summary.
	f := fakeEnv{goos: "linux", host: "h", ifaces: []ifaceData{
		{Name: "lo", Loopback: true, Up: true, Addrs: []ipAddr{{IP: "127.0.0.1", Prefix: 8, IsV4: true}}},
		{Name: "eth0", Up: false},
	}}
	c := f.collection()
	if _, ok := c.Value("networking.primary"); ok {
		t.Fatal("no primary expected")
	}
}

func TestInterfaceFactMinimal(t *testing.T) {
	// No MAC, MTU 0, no addresses: an empty-ish entry.
	entry := interfaceFact(ifaceData{Name: "x"})
	if len(entry) != 0 {
		t.Fatalf("expected empty entry, got %v", entry)
	}
}

func TestParseResolvDomain(t *testing.T) {
	if got := parseResolvDomain("search a.com b.com\n"); got != "a.com" {
		t.Errorf("search fallback = %q", got)
	}
	if got := parseResolvDomain("domain d.com\nsearch s.com\n"); got != "d.com" {
		t.Errorf("domain precedence = %q", got)
	}
	if got := parseResolvDomain("nameserver 1.1.1.1\n"); got != "" {
		t.Errorf("no domain = %q", got)
	}
	if got := parseResolvDomain("junk\n"); got != "" {
		t.Errorf("short line = %q", got)
	}
}

func TestDNSDomainWindows(t *testing.T) {
	f := fakeEnv{goos: "windows", envv: map[string]string{"USERDNSDOMAIN": "CORP.LOCAL"}}
	if got := f.collection().dnsDomain(); got != "corp.local" {
		t.Errorf("windows domain = %q", got)
	}
	// absent env
	if got := (fakeEnv{goos: "windows"}).collection().dnsDomain(); got != "" {
		t.Errorf("windows no domain = %q", got)
	}
}

func TestDNSDomainNoResolv(t *testing.T) {
	if got := (fakeEnv{goos: "linux"}).collection().dnsDomain(); got != "" {
		t.Errorf("no resolv.conf = %q", got)
	}
}

func TestNetmaskAndNetwork(t *testing.T) {
	if got := netmaskV4(24); got != "255.255.255.0" {
		t.Errorf("netmaskV4 = %q", got)
	}
	if got := netmaskV4(33); got != "" {
		t.Errorf("invalid prefix = %q", got)
	}
	if got := networkAddr("10.1.2.3", 16); got != "10.1.0.0" {
		t.Errorf("networkAddr v4 = %q", got)
	}
	if got := networkAddr("fe80::1", 64); got != "fe80::" {
		t.Errorf("networkAddr v6 = %q", got)
	}
	if got := networkAddr("not-an-ip", 24); got != "" {
		t.Errorf("networkAddr invalid = %q", got)
	}
	if got := prefix6(64); got != "/64" {
		t.Errorf("prefix6 = %q", got)
	}
}
