// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// MarshalJSON renders a fact value (or a whole ToHash map) as indented JSON with
// a trailing newline. It is exported so a consumer such as go-ruby-facter can
// reproduce Facter's `--json` output without re-deriving it.
func MarshalJSON(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}

// MarshalYAML renders a fact value as a small, deterministic subset of YAML
// (sorted block mappings, block sequences and scalars) — the counterpart of the
// external-fact YAML reader, and Facter's `--yaml` output shape.
func MarshalYAML(v any) string {
	var b strings.Builder
	if m, ok := v.(map[string]any); ok && len(m) == 0 {
		return "{}\n"
	}
	encodeYAML(&b, v, 0)
	return b.String()
}

// encodeYAML writes v at the given indentation depth.
func encodeYAML(b *strings.Builder, v any, depth int) {
	switch t := v.(type) {
	case map[string]any:
		encodeYAMLMap(b, t, depth)
	case []any:
		encodeYAMLSeq(b, t, depth)
	default:
		b.WriteString(strings.Repeat("  ", depth))
		b.WriteString(yamlScalar(v))
		b.WriteByte('\n')
	}
}

// encodeYAMLMap writes a block mapping with sorted keys.
func encodeYAMLMap(b *strings.Builder, m map[string]any, depth int) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pad := strings.Repeat("  ", depth)
	for _, k := range keys {
		val := m[k]
		if isCompound(val) {
			b.WriteString(pad)
			b.WriteString(k)
			b.WriteString(":\n")
			encodeYAML(b, val, depth+1)
			continue
		}
		b.WriteString(pad)
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(yamlScalar(val))
		b.WriteByte('\n')
	}
}

// encodeYAMLSeq writes a block sequence.
func encodeYAMLSeq(b *strings.Builder, s []any, depth int) {
	pad := strings.Repeat("  ", depth)
	for _, item := range s {
		b.WriteString(pad)
		b.WriteString("- ")
		b.WriteString(yamlScalar(item))
		b.WriteByte('\n')
	}
}

// isCompound reports whether v is a non-empty map or slice needing a nested block.
func isCompound(v any) bool {
	switch t := v.(type) {
	case map[string]any:
		return len(t) > 0
	case []any:
		return len(t) > 0
	default:
		return false
	}
}

// yamlScalar renders a leaf value; strings that could be mistaken for another
// type are quoted.
func yamlScalar(v any) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case string:
		return quoteYAMLString(t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case map[string]any:
		return "{}"
	case []any:
		return "[]"
	default:
		return fmt.Sprintf("%v", t)
	}
}

// quoteYAMLString quotes a string when it is empty or would otherwise parse as a
// non-string scalar or disturb the block layout.
func quoteYAMLString(s string) string {
	if s == "" {
		return `""`
	}
	if needsYAMLQuote(s) {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return s
}

// needsYAMLQuote reports whether s must be quoted to round-trip as a string.
func needsYAMLQuote(s string) bool {
	switch strings.ToLower(s) {
	case "true", "false", "null", "yes", "no":
		return true
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}
	return strings.ContainsAny(s, ":#\"'\n")
}
