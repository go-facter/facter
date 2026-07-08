// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

// Package facter is a pure-Go (no cgo) reimplementation of Puppet's Facter
// system-inventory engine. It discovers structured facts about the host —
// operating system, kernel, networking, processors, memory, filesystems,
// virtualization, uptime, identity and more — and exposes them through a small,
// stable API designed to be bound onto Ruby's Facter interface (Facter.value,
// Facter[], Facter.add, Facter.to_hash) by a consumer such as go-ruby-facter.
//
// Facts are resolved lazily and cached for the lifetime of a Collection, so a
// fact that is never queried costs nothing and one that is queried repeatedly is
// computed once. Structured facts are nested maps (for example os is a map with
// name/family/release members); the same underlying data is also surfaced through
// flat "legacy" aliases (operatingsystem, osfamily, ...) exactly as Facter does,
// so it is a drop-in for Puppet manifests that reference either shape.
//
// Every interaction with the operating system — reading a file, running a
// command, enumerating interfaces — flows through an injectable seam, so the
// per-OS collectors and their error branches are exercised deterministically in
// tests, without root and independent of the host the tests run on.
package facter

import (
	"sort"
	"strings"
	"sync"
)

// Version is the reported facterversion. It tracks the Facter schema this engine
// aims to be a drop-in for, not the Facter gem's own release cadence.
const Version = "4.0.0"

// Resolver produces the value of a fact. It returns (value, true) when the fact
// resolves and (nil, false) when it is not available on this host. The value may
// be any JSON-serialisable Go value, including a nested map[string]any for a
// structured fact.
type Resolver interface {
	Resolve(c *Collection) (any, bool)
}

// ResolverFunc adapts an ordinary function to the Resolver interface. It is the
// registration shape a Ruby custom-fact block maps onto.
type ResolverFunc func(c *Collection) (any, bool)

// Resolve implements Resolver.
func (f ResolverFunc) Resolve(c *Collection) (any, bool) { return f(c) }

// fact is a registered top-level fact together with its memoised result.
type fact struct {
	name     string
	resolver Resolver
	done     bool
	value    any
	present  bool
}

// Collection is a set of registered facts sharing one per-run cache and one set
// of operating-system seams. It is the engine object a Ruby binding wraps: query
// with Value, enumerate with ToHash, extend with Add, and load out-of-process
// facts with LoadExternalFacts.
type Collection struct {
	env   *env
	mu    sync.Mutex
	facts map[string]*fact
	order []string
	memos map[string]any
}

// memo computes a shared, un-exported sub-result once and caches it for the life
// of the Collection. Several legacy facts derive from the same expensive probe
// (the kernel version triple, the primary interface); memo lets them share it
// without exposing an internal fact in ToHash. fn must not call memo for the same
// key re-entrantly.
func (c *Collection) memo(key string, fn func() any) any {
	c.mu.Lock()
	if c.memos == nil {
		c.memos = map[string]any{}
	}
	if v, ok := c.memos[key]; ok {
		c.mu.Unlock()
		return v
	}
	c.mu.Unlock()

	v := fn()

	c.mu.Lock()
	c.memos[key] = v
	c.mu.Unlock()
	return v
}

// New returns a Collection with every built-in fact group registered against the
// real operating system. It is the entry point for production use.
func New() *Collection {
	return newWithEnv(defaultEnv())
}

// newWithEnv builds a Collection over an explicit seam set. Tests use it to drive
// the collectors against fixture data.
func newWithEnv(e *env) *Collection {
	c := &Collection{env: e, facts: map[string]*fact{}}
	registerCore(c)
	return c
}

// Add registers (or replaces) a top-level fact resolved by r. It is the Go
// surface behind Ruby's Facter.add. Registering a fact invalidates any cached
// value already computed for that name.
func (c *Collection) Add(name string, r Resolver) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.addLocked(name, r)
}

// AddFunc is Add for a plain resolver function.
func (c *Collection) AddFunc(name string, fn func(c *Collection) (any, bool)) {
	c.Add(name, ResolverFunc(fn))
}

// AddValue registers a fact with a constant value. It is the degenerate custom
// fact (Facter.add(:x) { setcode { v } } with no logic).
func (c *Collection) AddValue(name string, value any) {
	c.AddFunc(name, func(*Collection) (any, bool) { return value, true })
}

func (c *Collection) addLocked(name string, r Resolver) {
	name = strings.ToLower(name)
	f, ok := c.facts[name]
	if !ok {
		f = &fact{name: name}
		c.facts[name] = f
		c.order = append(c.order, name)
	}
	f.resolver = r
	f.done = false
	f.value = nil
	f.present = false
}

// resolveTop resolves a single top-level fact by name, memoising the result.
func (c *Collection) resolveTop(name string) (any, bool) {
	c.mu.Lock()
	f, ok := c.facts[name]
	if !ok {
		c.mu.Unlock()
		return nil, false
	}
	if f.done {
		v, p := f.value, f.present
		c.mu.Unlock()
		return v, p
	}
	resolver := f.resolver
	c.mu.Unlock()

	v, ok := resolver.Resolve(c)

	c.mu.Lock()
	f.done = true
	f.value = v
	f.present = ok
	c.mu.Unlock()
	return v, ok
}

// Value resolves a fact by dotted path. The first segment names a top-level fact
// and any further segments descend into a structured fact's nested maps, so
// Value("os.name") and Value("networking.interfaces.eth0.ip") both work. It
// returns (value, true) on success and (nil, false) when the fact or a path
// segment is absent. Value is the surface behind Ruby's Facter.value / Facter[].
func (c *Collection) Value(path string) (any, bool) {
	if path == "" {
		return nil, false
	}
	segs := strings.Split(path, ".")
	top, ok := c.resolveTop(strings.ToLower(segs[0]))
	if !ok {
		return nil, false
	}
	return descend(top, segs[1:])
}

// ValueString is Value coerced to its string form, the common case for a Ruby
// caller. The bool reports presence, not emptiness.
func (c *Collection) ValueString(path string) (string, bool) {
	v, ok := c.Value(path)
	if !ok {
		return "", false
	}
	return stringify(v), true
}

// descend walks the remaining dotted-path segments into nested maps.
func descend(v any, segs []string) (any, bool) {
	for _, s := range segs {
		m, ok := v.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok = m[s]
		if !ok {
			return nil, false
		}
	}
	return v, true
}

// ToHash resolves every registered fact and returns them as one nested map, the
// surface behind Ruby's Facter.to_hash. Facts that do not resolve on this host
// are omitted.
func (c *Collection) ToHash() map[string]any {
	c.mu.Lock()
	names := append([]string(nil), c.order...)
	c.mu.Unlock()

	out := map[string]any{}
	for _, name := range names {
		if v, ok := c.resolveTop(name); ok {
			out[name] = v
		}
	}
	return out
}

// Names returns the registered top-level fact names in registration order. It
// does not resolve anything.
func (c *Collection) Names() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := append([]string(nil), c.order...)
	sort.Strings(out)
	return out
}
