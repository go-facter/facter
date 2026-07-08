// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"strings"
	"testing"
)

func TestMarshalJSON(t *testing.T) {
	out, err := MarshalJSON(map[string]any{"a": 1})
	if err != nil || !strings.Contains(out, `"a": 1`) || !strings.HasSuffix(out, "\n") {
		t.Fatalf("json = %q err %v", out, err)
	}
	if _, err := MarshalJSON(make(chan int)); err == nil {
		t.Fatal("unserializable value should error")
	}
}

func TestMarshalYAMLEmptyMap(t *testing.T) {
	if got := MarshalYAML(map[string]any{}); got != "{}\n" {
		t.Fatalf("empty map = %q", got)
	}
}

func TestMarshalYAMLScalarTop(t *testing.T) {
	if got := MarshalYAML("hi"); got != "hi\n" {
		t.Fatalf("scalar top = %q", got)
	}
}

func TestMarshalYAMLNested(t *testing.T) {
	v := map[string]any{
		"os": map[string]any{
			"name":   "Ubuntu",
			"family": "Debian",
		},
		"tags":  []any{"a", "b"},
		"count": 3,
		"empty": map[string]any{},
	}
	out := MarshalYAML(v)
	want := []string{
		"count: 3\n",
		"empty: {}\n",
		"os:\n",
		"  family: Debian\n",
		"  name: Ubuntu\n",
		"tags:\n",
		"  - a\n",
		"  - b\n",
	}
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in:\n%s", w, out)
		}
	}
}

func TestYAMLScalarTypes(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{nil, "null"},
		{"plain", "plain"},
		{true, "true"},
		{false, "false"},
		{7, "7"},
		{int64(8), "8"},
		{uint64(9), "9"},
		{1.5, "1.5"},
		{map[string]any{}, "{}"},
		{[]any{}, "[]"},
		{struct{ X int }{1}, "{1}"},
	}
	for _, tc := range cases {
		if got := yamlScalar(tc.in); got != tc.want {
			t.Errorf("yamlScalar(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestQuoteYAMLString(t *testing.T) {
	if got := quoteYAMLString(""); got != `""` {
		t.Errorf("empty = %q", got)
	}
	if got := quoteYAMLString("true"); got != `"true"` {
		t.Errorf("bool-like = %q", got)
	}
	if got := quoteYAMLString("a:b"); got != `"a:b"` {
		t.Errorf("colon = %q", got)
	}
	if got := quoteYAMLString(`he said "hi"`); got != `"he said \"hi\""` {
		t.Errorf("quotes = %q", got)
	}
	if got := quoteYAMLString("plain"); got != "plain" {
		t.Errorf("plain = %q", got)
	}
}

func TestNeedsYAMLQuote(t *testing.T) {
	for _, s := range []string{"true", "NULL", "yes", "no", "12", "3.5", "a:b", "x#y", "line\nbreak"} {
		if !needsYAMLQuote(s) {
			t.Errorf("needsYAMLQuote(%q) = false, want true", s)
		}
	}
	if needsYAMLQuote("hello") {
		t.Error("plain word should not need quoting")
	}
}

func TestIsCompound(t *testing.T) {
	if isCompound(map[string]any{}) || isCompound([]any{}) || isCompound(5) {
		t.Error("empty/scalar should not be compound")
	}
	if !isCompound(map[string]any{"a": 1}) || !isCompound([]any{1}) {
		t.Error("non-empty map/slice should be compound")
	}
}
