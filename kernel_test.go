// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "testing"

func TestKernelLinux(t *testing.T) {
	f := fakeEnv{
		goos: "linux",
		cmds: map[string]string{"uname -s": "Linux\n", "uname -r": "6.5.0-27-generic\n"},
	}
	c := f.collection()
	if v, _ := c.Value("kernel"); v != "Linux" {
		t.Fatalf("kernel = %v", v)
	}
	if v, _ := c.Value("kernelrelease"); v != "6.5.0-27-generic" {
		t.Fatalf("kernelrelease = %v", v)
	}
	if v, _ := c.Value("kernelversion"); v != "6.5.0" {
		t.Fatalf("kernelversion = %v", v)
	}
	if v, _ := c.Value("kernelmajversion"); v != "6.5" {
		t.Fatalf("kernelmajversion = %v", v)
	}
}

func TestKernelLinuxNoUname(t *testing.T) {
	c := fakeEnv{goos: "linux"}.collection()
	if v, _ := c.Value("kernel"); v != "unknown" {
		t.Fatalf("kernel default = %v", v)
	}
	if v, _ := c.Value("kernelrelease"); v != "" {
		t.Fatalf("kernelrelease default = %v", v)
	}
}

func TestKernelWindows(t *testing.T) {
	f := fakeEnv{
		goos: "windows",
		cmds: map[string]string{"cmd /c ver": "Microsoft Windows [Version 10.0.19045.0]\n"},
	}
	c := f.collection()
	if v, _ := c.Value("kernel"); v != "windows" {
		t.Fatalf("win kernel = %v", v)
	}
	if v, _ := c.Value("kernelrelease"); v != "10.0.19045.0" {
		t.Fatalf("win kernelrelease = %v", v)
	}
	if v, _ := c.Value("kernelmajversion"); v != "10.0" {
		t.Fatalf("win kernelmajversion = %v", v)
	}
}

func TestNumericPrefix(t *testing.T) {
	if numericPrefix("6.5.0-27-generic") != "6.5.0" {
		t.Error("numericPrefix suffix strip")
	}
	if numericPrefix("") != "" {
		t.Error("numericPrefix empty")
	}
	if numericPrefix("abc") != "" {
		t.Error("numericPrefix non-numeric")
	}
}

func TestMajorMinor(t *testing.T) {
	if majorMinor("6") != "6" {
		t.Error("majorMinor single")
	}
	if majorMinor("6.5.0") != "6.5" {
		t.Error("majorMinor triple")
	}
}

func TestNestedStringMissing(t *testing.T) {
	if nestedString(map[string]any{}, "a", "b") != "" {
		t.Error("nestedString missing")
	}
	if nestedString("scalar", "a") != "" {
		t.Error("nestedString into scalar")
	}
	if nestedString(map[string]any{"a": 3}, "a") != "" {
		t.Error("nestedString non-string leaf")
	}
}
