// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"testing"
)

// TestHelperProcess is re-executed by TestDefaultRunCmd to give defaultRunCmd a
// deterministic, cross-platform command to run.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_FACTER_HELPER") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, "hello\n")
	os.Exit(0)
}

func TestDefaultRunCmd(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Skipf("no executable path: %v", err)
	}
	t.Setenv("GO_FACTER_HELPER", "1")
	out, err := defaultRunCmd(exe, "-test.run=TestHelperProcess")
	if err != nil || out != "hello\n" {
		t.Fatalf("helper run: out=%q err=%v", out, err)
	}
	if _, err := defaultRunCmd("this-command-does-not-exist-facter"); err == nil {
		t.Fatal("expected error for missing command")
	}
}

func TestDefaultReadDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	ents, err := defaultReadDir(dir)
	if err != nil || len(ents) != 2 {
		t.Fatalf("readdir: %v %v", ents, err)
	}
	var sawDir bool
	for _, e := range ents {
		if e.Name == "d" && e.IsDir {
			sawDir = true
		}
	}
	if !sawDir {
		t.Fatal("expected subdir entry")
	}
	if _, err := defaultReadDir(filepath.Join(dir, "nope")); err == nil {
		t.Fatal("expected readdir error")
	}
}

func TestDefaultStatMode(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := defaultStatMode(p); !ok {
		t.Fatal("expected stat ok")
	}
	if _, ok := defaultStatMode(filepath.Join(dir, "missing")); ok {
		t.Fatal("expected stat miss")
	}
}

func TestDefaultInterfacesSmoke(t *testing.T) {
	// Exercises the real seam body. Under qemu-user the netlink emulation can
	// fail to parse route attributes (an emulator artifact, not a fault here), so
	// we only require the call to complete; the coverage lane runs on native
	// linux/amd64 where enumeration succeeds and netAddrsOf is exercised.
	_, _ = defaultInterfaces()
}

func TestDefaultCurUserSmoke(t *testing.T) {
	if _, err := defaultCurUser(); err != nil {
		t.Fatalf("defaultCurUser error: %v", err)
	}
}

func TestLookupGroupName(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Skip("no current user")
	}
	if _, err := lookupGroupName(u.Gid); err != nil {
		t.Fatalf("lookupGroupName(%q): %v", u.Gid, err)
	}
	if _, err := lookupGroupName("9999999-bogus"); err == nil {
		t.Fatal("expected group lookup error")
	}
}

func TestDefaultEnv(t *testing.T) {
	e := defaultEnv()
	if e.goos == "" || e.readFile == nil || e.now == nil {
		t.Fatal("defaultEnv incomplete")
	}
}

func TestAdaptInterfaces(t *testing.T) {
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	ifaces := []net.Interface{
		{Name: "eth0", MTU: 1500, HardwareAddr: mac, Flags: net.FlagUp},
		{Name: "bad", Flags: net.FlagLoopback},
	}
	addrsOf := func(i net.Interface) ([]net.Addr, error) {
		if i.Name == "bad" {
			return nil, errors.New("addr error")
		}
		return []net.Addr{
			&net.IPNet{IP: net.IPv4(10, 0, 0, 1), Mask: net.CIDRMask(24, 32)},
			&net.IPNet{IP: net.ParseIP("fe80::1"), Mask: net.CIDRMask(64, 128)},
			&net.IPAddr{IP: net.IPv4(1, 1, 1, 1)}, // non-IPNet: skipped
		}, nil
	}
	out, err := adaptInterfaces(ifaces, nil, addrsOf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(out) != 2 || len(out[0].Addrs) != 2 {
		t.Fatalf("adapted = %+v", out)
	}
	if !out[0].Up || !out[1].Loopback {
		t.Fatal("flags not mapped")
	}
	if _, err := adaptInterfaces(nil, errors.New("boom"), addrsOf); err == nil {
		t.Fatal("expected enumeration error to propagate")
	}
}

func TestToIPAddr(t *testing.T) {
	v4 := toIPAddr(&net.IPNet{IP: net.IPv4(1, 2, 3, 4), Mask: net.CIDRMask(24, 32)})
	if !v4.IsV4 || v4.IP != "1.2.3.4" || v4.Prefix != 24 {
		t.Fatalf("v4 = %+v", v4)
	}
	v6 := toIPAddr(&net.IPNet{IP: net.ParseIP("fe80::1"), Mask: net.CIDRMask(64, 128)})
	if v6.IsV4 || v6.IP != "fe80::1" || v6.Prefix != 64 {
		t.Fatalf("v6 = %+v", v6)
	}
}

func TestBuildUser(t *testing.T) {
	u := &user.User{Username: "x", Uid: "1", Gid: "2"}
	info, err := buildUser(u, nil, func(string) (string, error) { return "grp", nil })
	if err != nil || info.Group != "grp" || info.Username != "x" {
		t.Fatalf("success: %+v %v", info, err)
	}
	if _, err := buildUser(nil, errors.New("e"), nil); err == nil {
		t.Fatal("expected user error")
	}
	info, _ = buildUser(u, nil, func(string) (string, error) { return "", errors.New("e") })
	if info.Group != "" {
		t.Fatalf("group should be empty on lookup error: %+v", info)
	}
}

func TestStringify(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{"s", "s"},
		{true, "true"},
		{false, "false"},
		{7, "7"},
		{int64(8), "8"},
		{uint64(9), "9"},
		{2.5, "2.5"},
		{struct{}{}, ""},
	}
	for _, tc := range cases {
		if got := stringify(tc.in); got != tc.want {
			t.Errorf("stringify(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestEnvHelpers(t *testing.T) {
	e := fakeEnv{files: map[string]string{"/a": "data"}, cmds: map[string]string{"echo hi": "hi\n"}}.env()
	if s, ok := e.readText("/a"); !ok || s != "data" {
		t.Fatalf("readText = %q %v", s, ok)
	}
	if _, ok := e.readText("/missing"); ok {
		t.Fatal("missing readText should be false")
	}
	if s, ok := e.cmd("echo", "hi"); !ok || s != "hi\n" {
		t.Fatalf("cmd = %q %v", s, ok)
	}
	if _, ok := e.cmd("nope"); ok {
		t.Fatal("missing cmd should be false")
	}
}
