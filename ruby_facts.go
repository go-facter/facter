// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "strings"

// rubyProbe is the one-liner run to read the host Ruby's identity: version,
// sitedir and platform, one per line. A consumer that embeds a Ruby runtime (for
// example go-ruby-facter under rbgo) can override this fact with its own values.
const rubyProbe = `print RUBY_VERSION,"\n",RbConfig::CONFIG["sitedir"],"\n",RUBY_PLATFORM`

// collectRuby builds the structured ruby fact (platform / sitedir / version) for
// the host Ruby interpreter, if one is on PATH. It is best-effort: with no ruby
// available the fact is simply absent, exactly as Facter reports it on a host
// without Ruby.
func (c *Collection) collectRuby() (any, bool) {
	out, ok := c.env.cmd("ruby", "-rrbconfig", "-e", rubyProbe)
	if !ok {
		return nil, false
	}
	version, sitedir, platform := parseRubyProbe(out)
	if version == "" {
		return nil, false
	}
	m := map[string]any{"version": version}
	putNonEmpty(m, "sitedir", sitedir)
	putNonEmpty(m, "platform", platform)
	return m, true
}

// parseRubyProbe reads the three newline-separated fields the ruby probe prints.
func parseRubyProbe(out string) (version, sitedir, platform string) {
	lines := strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n")
	get := func(i int) string {
		if i < len(lines) {
			return strings.TrimSpace(lines[i])
		}
		return ""
	}
	return get(0), get(1), get(2)
}
