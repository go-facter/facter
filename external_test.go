// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExternalFactsAllFormats(t *testing.T) {
	dir := "/facts.d"
	j := filepath.Join(dir, "a.json")
	y := filepath.Join(dir, "b.yaml")
	txt := filepath.Join(dir, "c.txt")
	exe := filepath.Join(dir, "d.sh")
	f := fakeEnv{
		dirs: map[string][]dirEntry{dir: {
			{Name: "a.json"}, {Name: "b.yaml"}, {Name: "c.txt"},
			{Name: "d.sh"}, {Name: "e.conf"}, {Name: "sub", IsDir: true},
		}},
		files: map[string]string{
			j:   `{"role":"web","port":8080}`,
			y:   "team: sre\n",
			txt: "env=prod\n",
		},
		modes: map[string]os.FileMode{exe: 0o755},
		cmds:  map[string]string{exe: "custom=yes\n"},
	}
	c := f.collection()
	if err := c.LoadExternalFacts(dir); err != nil {
		t.Fatalf("LoadExternalFacts error: %v", err)
	}
	if v, _ := c.Value("role"); v != "web" {
		t.Fatalf("role = %v", v)
	}
	if v, _ := c.Value("port"); v != float64(8080) {
		t.Fatalf("port = %v (%T)", v, v)
	}
	if v, _ := c.Value("team"); v != "sre" {
		t.Fatalf("team = %v", v)
	}
	if v, _ := c.Value("env"); v != "prod" {
		t.Fatalf("env = %v", v)
	}
	if v, _ := c.Value("custom"); v != "yes" {
		t.Fatalf("custom = %v", v)
	}
	if _, ok := c.Value("e"); ok {
		t.Fatal("unknown-format file must be ignored")
	}
}

func TestLoadExternalReadDirError(t *testing.T) {
	f := fakeEnv{dirErr: map[string]bool{"/missing": true}}
	if err := f.collection().LoadExternalFacts("/missing"); err == nil {
		t.Fatal("expected readDir error")
	}
}

func TestLoadExternalFileErrors(t *testing.T) {
	dir := "/facts.d"
	exe := filepath.Join(dir, "run.sh")
	f := fakeEnv{
		dirs: map[string][]dirEntry{dir: {
			{Name: "a.json"}, {Name: "b.yaml"}, {Name: "c.txt"}, {Name: "run.sh"},
		}},
		// No file contents -> json/yaml/txt reads fail.
		modes: map[string]os.FileMode{exe: 0o755},
		// No cmd for run.sh -> exec fails.
	}
	if err := f.collection().LoadExternalFacts(dir); err == nil {
		t.Fatal("expected the first file error")
	}
}

func TestLoadExternalInvalidJSON(t *testing.T) {
	dir := "/f"
	j := filepath.Join(dir, "x.json")
	f := fakeEnv{
		dirs:  map[string][]dirEntry{dir: {{Name: "x.json"}}},
		files: map[string]string{j: "{not json"},
	}
	if err := f.collection().LoadExternalFacts(dir); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestLoadExternalNonExecutableIgnored(t *testing.T) {
	dir := "/f"
	f := fakeEnv{dirs: map[string][]dirEntry{dir: {{Name: "note.conf"}}}}
	c := f.collection()
	if err := c.LoadExternalFacts(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseJSONFacts(t *testing.T) {
	m, err := parseJSONFacts(`{"a":1}`)
	if err != nil || m["a"] != float64(1) {
		t.Fatalf("valid json: %v %v", m, err)
	}
	if _, err := parseJSONFacts("nope"); err == nil {
		t.Fatal("invalid json should error")
	}
}

func TestParseKeyValue(t *testing.T) {
	m := parseKeyValue("# comment\n\nfoo=bar\nnum=5\nnoequals\n")
	if m["foo"] != "bar" || m["num"] != 5 {
		t.Fatalf("parseKeyValue = %v", m)
	}
	if _, ok := m["noequals"]; ok {
		t.Fatal("line without = must be skipped")
	}
}

func TestParseSimpleYAML(t *testing.T) {
	text := `---
# comment
db:
  host: localhost
  port: 5432
tags:
  - a
  - b
flag: true
empty:
`
	m, err := parseSimpleYAML(text)
	if err != nil {
		t.Fatalf("yaml error: %v", err)
	}
	db, ok := m["db"].(map[string]any)
	if !ok || db["host"] != "localhost" || db["port"] != 5432 {
		t.Fatalf("db = %v", m["db"])
	}
	tags, ok := m["tags"].([]any)
	if !ok || len(tags) != 2 || tags[0] != "a" {
		t.Fatalf("tags = %v", m["tags"])
	}
	if m["flag"] != true {
		t.Fatalf("flag = %v", m["flag"])
	}
	if m["empty"] != nil {
		t.Fatalf("empty = %v", m["empty"])
	}
}

func TestParseSimpleYAMLEmpty(t *testing.T) {
	m, err := parseSimpleYAML("\n\n")
	if err != nil || len(m) != 0 {
		t.Fatalf("empty yaml = %v %v", m, err)
	}
}

func TestParseSimpleYAMLErrors(t *testing.T) {
	if _, err := parseSimpleYAML("- a\n- b\n"); err == nil {
		t.Fatal("top-level sequence should error")
	}
	if _, err := parseSimpleYAML("nocolonhere\n"); err == nil {
		t.Fatal("malformed mapping line should error")
	}
	// A malformed line inside a nested block propagates the error upward.
	if _, err := parseSimpleYAML("a:\n  nocolon\n"); err == nil {
		t.Fatal("nested malformed line should error")
	}
}

func TestParseSimpleYAMLKeyNoNested(t *testing.T) {
	// A bare "key:" followed by a same-or-lower indent line -> nil value.
	m, err := parseSimpleYAML("a:\nb: 1\n")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if m["a"] != nil || m["b"] != 1 {
		t.Fatalf("m = %v", m)
	}
}

func TestSplitYAMLKV(t *testing.T) {
	if k, v, ok := splitYAMLKV(`"quoted": val`); !ok || k != "quoted" || v != "val" {
		t.Fatalf("kv = %q %q %v", k, v, ok)
	}
	if _, _, ok := splitYAMLKV("noseparator"); ok {
		t.Fatal("no separator should be false")
	}
}

func TestScalarValue(t *testing.T) {
	cases := map[string]any{
		`"quoted"`: "quoted",
		`'single'`: "single",
		"true":     true,
		"FALSE":    false,
		"42":       42,
		"3.14":     3.14,
		"plain":    "plain",
	}
	for in, want := range cases {
		if got := scalarValue(in); got != want {
			t.Errorf("scalarValue(%q) = %v (%T), want %v", in, got, got, want)
		}
	}
}
