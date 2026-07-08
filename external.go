// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// LoadExternalFacts loads Facter external facts from one or more facts.d style
// directories and registers each as a top-level fact. Three data formats are
// understood — .json, .yaml/.yml and .txt (key=value) — and any other file with
// an executable bit is run, its key=value stdout parsed as facts. This is the
// seam a Ruby binding drives to honour Puppet's external-fact contract. Errors
// reading a directory or an individual fact file are collected; the first is
// returned, but every readable fact is still loaded.
func (c *Collection) LoadExternalFacts(dirs ...string) error {
	var firstErr error
	note := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for _, dir := range dirs {
		ents, err := c.env.readDir(dir)
		if err != nil {
			note(err)
			continue
		}
		sort.Slice(ents, func(i, j int) bool { return ents[i].Name < ents[j].Name })
		for _, e := range ents {
			if e.IsDir {
				continue
			}
			facts, err := c.loadExternalFile(filepath.Join(dir, e.Name))
			if err != nil {
				note(err)
				continue
			}
			c.registerExternal(facts)
		}
	}
	return firstErr
}

// registerExternal registers external facts in a deterministic order.
func (c *Collection) registerExternal(facts map[string]any) {
	keys := make([]string, 0, len(facts))
	for k := range facts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		c.AddValue(k, facts[k])
	}
}

// loadExternalFile parses one external-fact file according to its extension, or
// executes it when it is a runnable structured/executable fact.
func (c *Collection) loadExternalFile(path string) (map[string]any, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		text, ok := c.env.readText(path)
		if !ok {
			return nil, fmt.Errorf("facter: cannot read external fact %q", path)
		}
		return parseJSONFacts(text)
	case ".yaml", ".yml":
		text, ok := c.env.readText(path)
		if !ok {
			return nil, fmt.Errorf("facter: cannot read external fact %q", path)
		}
		return parseSimpleYAML(text)
	case ".txt":
		text, ok := c.env.readText(path)
		if !ok {
			return nil, fmt.Errorf("facter: cannot read external fact %q", path)
		}
		return parseKeyValue(text), nil
	default:
		mode, ok := c.env.statMode(path)
		if !ok || mode&0o111 == 0 {
			return map[string]any{}, nil // not executable, not a known format: ignore
		}
		out, ok := c.env.cmd(path)
		if !ok {
			return nil, fmt.Errorf("facter: external fact %q failed to run", path)
		}
		return parseKeyValue(out), nil
	}
}

// parseJSONFacts unmarshals a JSON object of facts.
func parseJSONFacts(text string) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(text), &m); err != nil {
		return nil, fmt.Errorf("facter: invalid external JSON: %w", err)
	}
	return m, nil
}

// parseKeyValue parses key=value lines (external .txt and executable facts),
// inferring scalar types and ignoring blanks and comments.
func parseKeyValue(text string) map[string]any {
	out := map[string]any{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = scalarValue(strings.TrimSpace(v))
	}
	return out
}

// yamlLine is a non-blank YAML source line with its indentation depth.
type yamlLine struct {
	indent int
	text   string
}

// parseSimpleYAML parses the block-mapping subset of YAML that Facter external
// facts use: nested key: value mappings, scalar values with type inference, and
// block sequences of scalars. It intentionally rejects nothing structural beyond
// a top level that is not a mapping.
func parseSimpleYAML(text string) (map[string]any, error) {
	lines := yamlLines(text)
	if len(lines) == 0 {
		return map[string]any{}, nil
	}
	v, _, err := parseYAMLBlock(lines, 0)
	if err != nil {
		return nil, err
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("facter: external YAML top level is not a mapping")
	}
	return m, nil
}

// yamlLines strips blanks, comments and the optional "---" document marker,
// returning each remaining line with its leading-space indentation.
func yamlLines(text string) []yamlLine {
	var out []yamlLine
	for _, raw := range strings.Split(text, "\n") {
		trimmed := strings.TrimRight(raw, "\r")
		content := strings.TrimSpace(trimmed)
		if content == "" || content == "---" || strings.HasPrefix(content, "#") {
			continue
		}
		indent := len(trimmed) - len(strings.TrimLeft(trimmed, " "))
		out = append(out, yamlLine{indent: indent, text: content})
	}
	return out
}

// parseYAMLBlock parses the block starting at line i, consuming every following
// line at the same indentation. It returns the parsed value (a mapping or a
// sequence) and the index of the first unconsumed line.
func parseYAMLBlock(lines []yamlLine, i int) (any, int, error) {
	base := lines[i].indent
	if strings.HasPrefix(lines[i].text, "- ") {
		return parseYAMLSequence(lines, i, base)
	}
	return parseYAMLMapping(lines, i, base)
}

// parseYAMLMapping parses a block mapping at the given indentation.
func parseYAMLMapping(lines []yamlLine, i, base int) (any, int, error) {
	m := map[string]any{}
	for i < len(lines) && lines[i].indent == base {
		key, val, ok := splitYAMLKV(lines[i].text)
		if !ok {
			return nil, i, fmt.Errorf("facter: malformed YAML line %q", lines[i].text)
		}
		if val != "" {
			m[key] = scalarValue(val)
			i++
			continue
		}
		// Value is on the following, more-indented lines (nested block).
		if i+1 < len(lines) && lines[i+1].indent > base {
			child, ni, err := parseYAMLBlock(lines, i+1)
			if err != nil {
				return nil, ni, err
			}
			m[key] = child
			i = ni
		} else {
			m[key] = nil
			i++
		}
	}
	return m, i, nil
}

// parseYAMLSequence parses a block sequence of scalars at the given indentation.
func parseYAMLSequence(lines []yamlLine, i, base int) (any, int, error) {
	var seq []any
	for i < len(lines) && lines[i].indent == base && strings.HasPrefix(lines[i].text, "- ") {
		item := strings.TrimSpace(strings.TrimPrefix(lines[i].text, "-"))
		seq = append(seq, scalarValue(item))
		i++
	}
	return seq, i, nil
}

// splitYAMLKV splits a "key: value" line, tolerating a bare "key:" with the
// value on following lines. The key is unquoted; the value is returned trimmed.
func splitYAMLKV(line string) (key, val string, ok bool) {
	if k, v, found := strings.Cut(line, ":"); found {
		return unquote(strings.TrimSpace(k)), strings.TrimSpace(v), true
	}
	return "", "", false
}

// scalarValue infers a YAML/key=value scalar's Go type: quoted and plain strings,
// booleans, integers and floats.
func scalarValue(s string) any {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1]
	}
	switch strings.ToLower(s) {
	case "true":
		return true
	case "false":
		return false
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}
