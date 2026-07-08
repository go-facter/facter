// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "testing"

const ubuntuOSRelease = `# a comment
NAME="Ubuntu"
ID=ubuntu
ID_LIKE=debian
VERSION_ID="22.04"
VERSION_CODENAME=jammy
PRETTY_NAME="Ubuntu 22.04.3 LTS"
bogusline
EMPTY=
`

func TestOSLinuxEtcOSRelease(t *testing.T) {
	f := fakeEnv{
		goos:  "linux",
		files: map[string]string{"/etc/os-release": ubuntuOSRelease},
		cmds:  map[string]string{"uname -m": "x86_64\n"},
	}
	c := f.collection()
	if v, _ := c.Value("os.name"); v != "Ubuntu" {
		t.Fatalf("os.name = %v", v)
	}
	if v, _ := c.Value("os.family"); v != "Debian" {
		t.Fatalf("os.family = %v", v)
	}
	if v, _ := c.Value("os.release.full"); v != "22.04" {
		t.Fatalf("release.full = %v", v)
	}
	if v, _ := c.Value("os.release.major"); v != "22" {
		t.Fatalf("release.major = %v", v)
	}
	if v, _ := c.Value("os.release.minor"); v != "04" {
		t.Fatalf("release.minor = %v", v)
	}
	if v, _ := c.Value("os.distro.codename"); v != "jammy" {
		t.Fatalf("codename = %v", v)
	}
	if v, _ := c.Value("os.architecture"); v != "x86_64" {
		t.Fatalf("arch = %v", v)
	}
	if v, _ := c.Value("operatingsystem"); v != "Ubuntu" {
		t.Fatalf("legacy operatingsystem = %v", v)
	}
}

func TestOSLinuxFallbackAndDefaults(t *testing.T) {
	// No /etc/os-release; falls back to /usr/lib and, absent NAME, capitalises ID.
	f := fakeEnv{
		goos:  "linux",
		files: map[string]string{"/usr/lib/os-release": "ID=fedora\nVERSION_ID=39\n"},
	}
	c := f.collection()
	if v, _ := c.Value("os.name"); v != "Fedora" {
		t.Fatalf("fallback name = %v", v)
	}
	if v, _ := c.Value("os.family"); v != "RedHat" {
		t.Fatalf("family = %v", v)
	}
	// uname -m unavailable -> arch from GOARCH.
	if v, _ := c.Value("os.architecture"); v != "x86_64" {
		t.Fatalf("arch fallback = %v", v)
	}
}

func TestOSLinuxEmptyRelease(t *testing.T) {
	c := fakeEnv{goos: "linux"}.collection()
	if v, _ := c.Value("os.name"); v != "Linux" {
		t.Fatalf("empty name = %v", v)
	}
	if v, _ := c.Value("os.family"); v != "Linux" {
		t.Fatalf("empty family = %v", v)
	}
	if v, _ := c.Value("os.distro.id"); v != "linux" {
		t.Fatalf("distro id default = %v", v)
	}
}

func TestOSFamilyMapping(t *testing.T) {
	cases := map[string]string{
		"debian": "Debian", "rhel": "RedHat", "suse": "Suse",
		"arch": "Archlinux", "gentoo": "Gentoo", "alpine": "Alpine",
		"weird": "Weird",
	}
	for id, want := range cases {
		if got := osFamily(id, ""); got != want {
			t.Errorf("osFamily(%q) = %q, want %q", id, got, want)
		}
	}
	if got := osFamily("", "ubuntu"); got != "Debian" {
		t.Errorf("ID_LIKE ubuntu -> %q", got)
	}
	if got := osFamily("", ""); got != "Linux" {
		t.Errorf("empty -> %q", got)
	}
}

func TestOSDarwin(t *testing.T) {
	sw := "ProductName:\tmacOS\nProductVersion:\t14.5\nBuildVersion:\t23F79\nbad line\n"
	f := fakeEnv{
		goos:   "darwin",
		goarch: "arm64",
		cmds: map[string]string{
			"sw_vers":  sw,
			"uname -r": "23.5.0\n",
			"uname -m": "arm64\n",
		},
	}
	c := f.collection()
	if v, _ := c.Value("os.name"); v != "Darwin" {
		t.Fatalf("darwin name = %v", v)
	}
	if v, _ := c.Value("os.macosx.version.full"); v != "14.5" {
		t.Fatalf("macosx version = %v", v)
	}
	if v, _ := c.Value("os.macosx.build"); v != "23F79" {
		t.Fatalf("build = %v", v)
	}
	if v, _ := c.Value("os.release.full"); v != "23.5.0" {
		t.Fatalf("release = %v", v)
	}
	if v, _ := c.Value("os.architecture"); v != "arm64" {
		t.Fatalf("arch = %v", v)
	}
}

func TestOSDarwinNoCommands(t *testing.T) {
	c := fakeEnv{goos: "darwin", goarch: "arm64"}.collection()
	if v, _ := c.Value("os.macosx.product"); v != "macOS" {
		t.Fatalf("default product = %v", v)
	}
	if v, _ := c.Value("os.architecture"); v != "aarch64" {
		t.Fatalf("arch fallback = %v", v)
	}
}

func TestOSWindows(t *testing.T) {
	f := fakeEnv{
		goos: "windows",
		cmds: map[string]string{"cmd /c ver": "\nMicrosoft Windows [Version 10.0.19045.0]\n"},
		envv: map[string]string{"PROCESSOR_ARCHITECTURE": "AMD64"},
	}
	c := f.collection()
	if v, _ := c.Value("os.name"); v != "windows" {
		t.Fatalf("win name = %v", v)
	}
	if v, _ := c.Value("os.release.full"); v != "10.0.19045.0" {
		t.Fatalf("win release = %v", v)
	}
	if v, _ := c.Value("os.architecture"); v != "x64" {
		t.Fatalf("win arch = %v", v)
	}
}

func TestWindowsArchVariants(t *testing.T) {
	mk := func(v string) *env {
		return fakeEnv{goos: "windows", envv: map[string]string{"PROCESSOR_ARCHITECTURE": v}}.env()
	}
	if got := windowsArch(mk("ARM64")); got != "arm64" {
		t.Errorf("ARM64 -> %q", got)
	}
	if got := windowsArch(mk("X86")); got != "x86" {
		t.Errorf("X86 -> %q", got)
	}
	if got := windowsArch(mk("IA64")); got != "ia64" {
		t.Errorf("IA64 -> %q", got)
	}
	// No env var -> GOARCH mapping.
	if got := windowsArch(fakeEnv{goos: "windows", goarch: "amd64"}.env()); got != "x86_64" {
		t.Errorf("no env -> %q", got)
	}
}

func TestOSGeneric(t *testing.T) {
	c := fakeEnv{goos: "freebsd", goarch: "riscv64"}.collection()
	if v, _ := c.Value("os.name"); v != "Freebsd" {
		t.Fatalf("generic name = %v", v)
	}
	if v, _ := c.Value("os.architecture"); v != "riscv64" {
		t.Fatalf("generic arch = %v", v)
	}
}

func TestArchName(t *testing.T) {
	cases := map[string]string{
		"amd64": "x86_64", "arm64": "aarch64", "386": "i386",
		"ppc64le": "ppc64le", "s390x": "s390x", "riscv64": "riscv64",
		"loong64": "loongarch64", "mips": "mips",
	}
	for in, want := range cases {
		if got := archName(in, ""); got != want {
			t.Errorf("archName(%q) = %q, want %q", in, got, want)
		}
	}
	if got := archName("amd64", " ppc64 "); got != "ppc64" {
		t.Errorf("uname override = %q", got)
	}
}

func TestParseWindowsVer(t *testing.T) {
	if got := parseWindowsVer("no version here"); got != "" {
		t.Errorf("missing version -> %q", got)
	}
	if got := parseWindowsVer("Version 6.1"); got != "6.1" {
		t.Errorf("unterminated -> %q", got)
	}
}

func TestSplitVersion(t *testing.T) {
	if m, n := splitVersion(""); m != "" || n != "" {
		t.Errorf("empty -> %q %q", m, n)
	}
	if m, n := splitVersion("10"); m != "10" || n != "" {
		t.Errorf("single -> %q %q", m, n)
	}
	if m, n := splitVersion("10.5.1"); m != "10" || n != "5" {
		t.Errorf("triple -> %q %q", m, n)
	}
}

func TestCapitalizeAndFirstNonEmpty(t *testing.T) {
	if capitalize("") != "" {
		t.Error("capitalize empty")
	}
	if capitalize("go") != "Go" {
		t.Error("capitalize go")
	}
	if firstNonEmpty("", "", "x") != "x" {
		t.Error("firstNonEmpty")
	}
	if firstNonEmpty("", "") != "" {
		t.Error("firstNonEmpty all empty")
	}
}

func TestFirstField(t *testing.T) {
	if firstField("  a b ") != "a" {
		t.Error("firstField value")
	}
	if firstField("   ") != "" {
		t.Error("firstField blank")
	}
}
