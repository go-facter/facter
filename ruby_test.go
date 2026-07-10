// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"strings"
	"testing"
)

// rubyCmdKey is the fakeEnv command key the ruby probe resolves to.
var rubyCmdKey = strings.Join([]string{"ruby", "-rrbconfig", "-e", rubyProbe}, " ")

func TestRubyPresent(t *testing.T) {
	c := (fakeEnv{cmds: map[string]string{
		rubyCmdKey: "3.1.2\n/usr/lib/ruby/site_ruby/3.1.0\nx86_64-linux\n",
	}}).collection()
	if v, _ := c.Value("ruby.version"); v != "3.1.2" {
		t.Errorf("version = %v", v)
	}
	if v, _ := c.Value("ruby.sitedir"); v != "/usr/lib/ruby/site_ruby/3.1.0" {
		t.Errorf("sitedir = %v", v)
	}
	if v, _ := c.Value("ruby.platform"); v != "x86_64-linux" {
		t.Errorf("platform = %v", v)
	}
}

func TestRubyAbsent(t *testing.T) {
	if _, ok := (fakeEnv{}).collection().Value("ruby"); ok {
		t.Fatal("no ruby command -> absent")
	}
}

func TestRubyEmptyVersion(t *testing.T) {
	// Command runs but prints nothing usable: absent.
	c := (fakeEnv{cmds: map[string]string{rubyCmdKey: "\n\n\n"}}).collection()
	if _, ok := c.Value("ruby"); ok {
		t.Fatal("blank version -> absent")
	}
}

func TestRubyVersionOnly(t *testing.T) {
	// Only the version line: sitedir/platform omitted, fact still present.
	c := (fakeEnv{cmds: map[string]string{rubyCmdKey: "2.7.0\n"}}).collection()
	if v, _ := c.Value("ruby.version"); v != "2.7.0" {
		t.Errorf("version = %v", v)
	}
	if _, ok := c.Value("ruby.sitedir"); ok {
		t.Error("sitedir should be omitted")
	}
}

func TestParseRubyProbe(t *testing.T) {
	v, s, p := parseRubyProbe("1\r\n2\r\n3")
	if v != "1" || s != "2" || p != "3" {
		t.Fatalf("parsed = %q %q %q", v, s, p)
	}
}
