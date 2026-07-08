// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "strings"

// collectOS builds the structured os fact: name, family, release, distro,
// architecture and hardware, dispatched per operating system. It is the anchor
// fact many legacy aliases derive from.
func (c *Collection) collectOS() (any, bool) {
	switch c.env.goos {
	case "linux":
		return c.osLinux(), true
	case "darwin":
		return c.osDarwin(), true
	case "windows":
		return c.osWindows(), true
	default:
		return c.osGeneric(), true
	}
}

// archName maps the Go architecture to Facter's architecture vocabulary, unless
// the host reported a uname machine string, which Facter prefers verbatim.
func archName(goarch, unameMachine string) string {
	if m := strings.TrimSpace(unameMachine); m != "" {
		return m
	}
	switch goarch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	case "386":
		return "i386"
	case "ppc64le":
		return "ppc64le"
	case "s390x":
		return "s390x"
	case "riscv64":
		return "riscv64"
	case "loong64":
		return "loongarch64"
	default:
		return goarch
	}
}

// unameMachine returns uname -m, or "" when the command is unavailable.
func (c *Collection) unameMachine() string {
	if out, ok := c.env.cmd("uname", "-m"); ok {
		return firstField(out)
	}
	return ""
}

// osLinux derives the os fact from /etc/os-release (falling back to
// /usr/lib/os-release), the modern cross-distribution source.
func (c *Collection) osLinux() map[string]any {
	text, ok := c.env.readText("/etc/os-release")
	if !ok {
		text, _ = c.env.readText("/usr/lib/os-release")
	}
	kv := parseOSRelease(text)

	name := osReleaseName(kv)
	family := osFamily(kv["ID"], kv["ID_LIKE"])
	full := kv["VERSION_ID"]
	major, minor := splitVersion(full)

	arch := archName(c.env.goarch, c.unameMachine())

	distro := map[string]any{
		"id":          firstNonEmpty(kv["ID"], "linux"),
		"description": kv["PRETTY_NAME"],
		"codename":    kv["VERSION_CODENAME"],
		"release": map[string]any{
			"full":  full,
			"major": major,
			"minor": minor,
		},
	}

	return map[string]any{
		"name":         name,
		"family":       family,
		"architecture": arch,
		"hardware":     arch,
		"distro":       distro,
		"release": map[string]any{
			"full":  full,
			"major": major,
			"minor": minor,
		},
	}
}

// osReleaseName picks the human OS name Facter reports, preferring the
// capitalised NAME field but normalising a few well-known ids.
func osReleaseName(kv map[string]string) string {
	if n := kv["NAME"]; n != "" {
		return n
	}
	switch kv["ID"] {
	case "":
		return "Linux"
	default:
		return capitalize(kv["ID"])
	}
}

// osFamily maps a distribution id (and ID_LIKE) to Facter's os.family value.
func osFamily(id, idLike string) string {
	candidates := append([]string{id}, strings.Fields(idLike)...)
	for _, c := range candidates {
		switch strings.ToLower(c) {
		case "debian", "ubuntu", "raspbian", "linuxmint":
			return "Debian"
		case "rhel", "fedora", "centos", "rocky", "almalinux", "amzn", "ol", "redhat":
			return "RedHat"
		case "suse", "opensuse", "sles", "opensuse-leap", "opensuse-tumbleweed":
			return "Suse"
		case "arch", "archlinux", "manjaro":
			return "Archlinux"
		case "gentoo":
			return "Gentoo"
		case "alpine":
			return "Alpine"
		}
	}
	if id != "" {
		return capitalize(id)
	}
	return "Linux"
}

// osDarwin derives the os fact from sw_vers and the Darwin uname release.
func (c *Collection) osDarwin() map[string]any {
	product, version, build := "", "", ""
	if out, ok := c.env.cmd("sw_vers"); ok {
		product, version, build = parseSwVers(out)
	}
	kernRel := ""
	if out, ok := c.env.cmd("uname", "-r"); ok {
		kernRel = firstField(out)
	}
	kMajor, kMinor := splitVersion(kernRel)
	vMajor, vMinor := splitVersion(version)
	arch := archName(c.env.goarch, c.unameMachine())

	macosx := map[string]any{
		"product": firstNonEmpty(product, "macOS"),
		"build":   build,
		"version": map[string]any{
			"full":  version,
			"major": vMajor,
			"minor": vMinor,
		},
	}

	return map[string]any{
		"name":         "Darwin",
		"family":       "Darwin",
		"architecture": arch,
		"hardware":     arch,
		"macosx":       macosx,
		"release": map[string]any{
			"full":  kernRel,
			"major": kMajor,
			"minor": kMinor,
		},
	}
}

// osWindows derives the os fact from the Windows version string and environment.
func (c *Collection) osWindows() map[string]any {
	full := ""
	if out, ok := c.env.cmd("cmd", "/c", "ver"); ok {
		full = parseWindowsVer(out)
	}
	major, minor := splitVersion(full)
	arch := windowsArch(c.env)

	return map[string]any{
		"name":         "windows",
		"family":       "windows",
		"architecture": arch,
		"hardware":     arch,
		"release": map[string]any{
			"full":  full,
			"major": major,
			"minor": minor,
		},
	}
}

// windowsArch reads the processor architecture from the environment, the
// registry-free source Windows exposes.
func windowsArch(e *env) string {
	if v, ok := e.lookupEnv("PROCESSOR_ARCHITECTURE"); ok {
		switch strings.ToUpper(v) {
		case "AMD64":
			return "x64"
		case "ARM64":
			return "arm64"
		case "X86":
			return "x86"
		default:
			return strings.ToLower(v)
		}
	}
	return archName(e.goarch, "")
}

// osGeneric is the portable fallback for operating systems without a dedicated
// collector: it reports what the runtime knows.
func (c *Collection) osGeneric() map[string]any {
	arch := archName(c.env.goarch, c.unameMachine())
	title := capitalize(c.env.goos)
	return map[string]any{
		"name":         title,
		"family":       title,
		"architecture": arch,
		"hardware":     arch,
		"release": map[string]any{
			"full":  "",
			"major": "",
			"minor": "",
		},
	}
}

// parseOSRelease parses the os-release KEY=VALUE format, stripping surrounding
// single or double quotes and ignoring blanks and comments.
func parseOSRelease(text string) map[string]string {
	kv := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		kv[strings.TrimSpace(k)] = unquote(strings.TrimSpace(v))
	}
	return kv
}

// unquote removes a single matching pair of single or double quotes.
func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// parseSwVers extracts ProductName, ProductVersion and BuildVersion from sw_vers.
func parseSwVers(out string) (product, version, build string) {
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch strings.TrimSpace(k) {
		case "ProductName":
			product = v
		case "ProductVersion":
			version = v
		case "BuildVersion":
			build = v
		}
	}
	return product, version, build
}

// parseWindowsVer pulls the dotted version out of a "Microsoft Windows [Version
// 10.0.19045.0]" banner.
func parseWindowsVer(out string) string {
	i := strings.LastIndex(out, "Version ")
	if i < 0 {
		return ""
	}
	rest := out[i+len("Version "):]
	if j := strings.IndexByte(rest, ']'); j >= 0 {
		rest = rest[:j]
	}
	return strings.TrimSpace(rest)
}

// splitVersion breaks "14.5.1" into major "14" and minor "5".
func splitVersion(full string) (major, minor string) {
	full = strings.TrimSpace(full)
	if full == "" {
		return "", ""
	}
	parts := strings.Split(full, ".")
	major = parts[0]
	if len(parts) > 1 {
		minor = parts[1]
	}
	return major, minor
}

// capitalize upper-cases the first rune of s, leaving the rest unchanged.
func capitalize(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// firstNonEmpty returns the first non-empty argument, or "".
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
