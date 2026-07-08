// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"reflect"
	"testing"
)

func TestAddAndValue(t *testing.T) {
	c := fakeEnv{}.collection()
	c.AddValue("custom", "hello")
	if v, ok := c.Value("custom"); !ok || v != "hello" {
		t.Fatalf("custom = %v, %v", v, ok)
	}
	// Re-register replaces and invalidates the cached value.
	c.AddValue("custom", "world")
	if v, _ := c.Value("custom"); v != "world" {
		t.Fatalf("re-add: %v", v)
	}
}

func TestAddFuncCollectionArg(t *testing.T) {
	c := fakeEnv{}.collection()
	c.AddFunc("derived", func(cc *Collection) (any, bool) {
		v, _ := cc.Value("facterversion")
		return v, true
	})
	if v, _ := c.Value("derived"); v != Version {
		t.Fatalf("derived = %v", v)
	}
}

func TestAddResolverInterface(t *testing.T) {
	c := fakeEnv{}.collection()
	c.Add("r", ResolverFunc(func(*Collection) (any, bool) { return 42, true }))
	if v, _ := c.Value("r"); v != 42 {
		t.Fatalf("r = %v", v)
	}
}

func TestValueMissing(t *testing.T) {
	c := fakeEnv{}.collection()
	if _, ok := c.Value(""); ok {
		t.Fatal("empty path should be absent")
	}
	if _, ok := c.Value("nope"); ok {
		t.Fatal("unknown fact should be absent")
	}
}

func TestValueDescend(t *testing.T) {
	c := fakeEnv{}.collection()
	c.AddValue("s", map[string]any{"a": map[string]any{"b": "deep"}})
	if v, ok := c.Value("s.a.b"); !ok || v != "deep" {
		t.Fatalf("s.a.b = %v %v", v, ok)
	}
	if _, ok := c.Value("s.a.missing"); ok {
		t.Fatal("missing leaf should be absent")
	}
	if _, ok := c.Value("s.a.b.c"); ok {
		t.Fatal("descend into scalar should be absent")
	}
}

func TestValueStringAndPresence(t *testing.T) {
	c := fakeEnv{}.collection()
	c.AddValue("n", 7)
	if s, ok := c.ValueString("n"); !ok || s != "7" {
		t.Fatalf("ValueString n = %q %v", s, ok)
	}
	if _, ok := c.ValueString("absent"); ok {
		t.Fatal("absent ValueString should report false")
	}
}

func TestResolveCaching(t *testing.T) {
	c := fakeEnv{}.collection()
	calls := 0
	c.AddFunc("counter", func(*Collection) (any, bool) {
		calls++
		return calls, true
	})
	_, _ = c.Value("counter")
	_, _ = c.Value("counter")
	if calls != 1 {
		t.Fatalf("resolver called %d times, want 1 (cached)", calls)
	}
}

func TestResolveFalseCaching(t *testing.T) {
	c := fakeEnv{}.collection()
	calls := 0
	c.AddFunc("maybe", func(*Collection) (any, bool) {
		calls++
		return nil, false
	})
	if _, ok := c.Value("maybe"); ok {
		t.Fatal("want absent")
	}
	_, _ = c.Value("maybe")
	if calls != 1 {
		t.Fatalf("absent resolver called %d times, want 1", calls)
	}
}

func TestToHashOmitsAbsent(t *testing.T) {
	c := fakeEnv{}.collection()
	c.AddValue("present", "x")
	c.AddFunc("gone", func(*Collection) (any, bool) { return nil, false })
	h := c.ToHash()
	if h["present"] != "x" {
		t.Fatalf("present missing from hash")
	}
	if _, ok := h["gone"]; ok {
		t.Fatal("absent fact must be omitted from hash")
	}
}

func TestNames(t *testing.T) {
	c := fakeEnv{}.collection()
	names := c.Names()
	found := false
	for i, n := range names {
		if i > 0 && names[i-1] > n {
			t.Fatal("Names not sorted")
		}
		if n == "os" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected core fact os in Names")
	}
}

func TestMemoComputesOnce(t *testing.T) {
	c := fakeEnv{}.collection()
	calls := 0
	fn := func() any { calls++; return "v" }
	if c.memo("k", fn) != "v" || c.memo("k", fn) != "v" {
		t.Fatal("memo value")
	}
	if calls != 1 {
		t.Fatalf("memo fn called %d times, want 1", calls)
	}
}

func TestNewRealCollection(t *testing.T) {
	// Exercise the production constructor and a full ToHash against the real
	// host; values are host-dependent so we only assert structural invariants.
	c := New()
	if _, ok := c.Value("facterversion"); !ok {
		t.Fatal("facterversion must resolve on the real host")
	}
	h := c.ToHash()
	if !reflect.DeepEqual(h["facterversion"], Version) {
		t.Fatalf("facterversion = %v", h["facterversion"])
	}
}
