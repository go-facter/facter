<p align="center"><img src="https://raw.githubusercontent.com/go-facter/brand/main/social/go-facter-facter.png" alt="go-facter/facter" width="720"></p>

# facter — go-facter

[![CI](https://github.com/go-facter/facter/actions/workflows/ci.yml/badge.svg)](https://github.com/go-facter/facter/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of Puppet's [Facter](https://www.puppet.com/docs/puppet/latest/facter.html)
system-inventory engine.** It discovers structured facts about the host — the
operating system, kernel, networking, processors, memory, filesystems,
virtualization, uptime, identity and more — and exposes them through a small,
stable Go API that a Ruby binding maps directly onto `Facter.value` / `Facter[]`
/ `Facter.add` / `Facter.to_hash`.

It is the engine layer for **[go-facter/go-ruby-facter](https://github.com/go-facter)**
(the Ruby-facing binding that lets Puppet manifests and Ruby code running under
`rbgo` resolve facts), but it is a **standalone, dependency-free** module — it
imports only the Go standard library.

> **Fidelity to Facter's schema.** Fact names and structure follow Facter's
> aggregate schema (`os`, `networking`, `processors`, `memory`, …) with the flat
> legacy aliases (`operatingsystem`, `osfamily`, `ipaddress`, …) alongside, so it
> is a drop-in for manifests that reference either shape.

## Design

Facts resolve **lazily** and are **cached** for the life of a `Collection`, so a
fact that is never queried costs nothing and one queried repeatedly is computed
once. Every interaction with the operating system — reading a file, running a
command, enumerating interfaces — goes through an **injectable seam**, so the
per-OS collectors and their error branches are covered deterministically in
tests, without root and independent of the host the tests run on. The collectors
branch on a runtime `GOOS` string rather than build tags, which is what lets the
per-OS 100%-coverage gate hold identically on Linux, macOS and Windows and lets
every fact group's logic run on all six 64-bit architectures.

## Fact groups

`os` (name/family/release/distro, `os.macosx` on Darwin) · `kernel` /
`kernelrelease` / `kernelversion` / `kernelmajversion` · `networking`
(hostname/fqdn/domain, per-interface ip/ip6/mac/mtu/netmask/network + bindings,
primary) · `processors` (count/physicalcount/models/isa) · `memory`
(system + swap, bytes and human sizes) · `mountpoints` · `filesystems` · `disks`
· `virtual` / `is_virtual` (hypervisor and container detection) · `system_uptime`
(+ `uptime*` aliases) · `timezone` · `identity` (user/group/uid/gid/privileged) ·
`path` · `facterversion`, plus the flat legacy aliases.

Where a platform cannot provide a fact in pure Go (for example memory on Windows),
the fact is skipped gracefully rather than reported wrong.

## Go API

```go
c := facter.New()

v, ok := c.Value("os.name")               // dotted-path query -> Facter.value / Facter[]
s, ok := c.ValueString("networking.fqdn") // string-coerced query
all   := c.ToHash()                       // nested map of every fact -> Facter.to_hash

c.AddValue("role", "web")                 // custom facts -> Facter.add
c.AddFunc("answer", func(cc *facter.Collection) (any, bool) { return 42, true })

// External facts from facts.d directories: .json / .yaml / .txt + executables.
err := c.LoadExternalFacts("/etc/facter/facts.d")

// Deterministic encoders (Facter's --json / --yaml output shapes).
j, _ := facter.MarshalJSON(all)
y    := facter.MarshalYAML(all)
```

## CLI

```sh
facter                # print all facts (JSON)
facter os.name        # print a single fact by dotted path
facter --yaml os      # YAML output of a structured fact
```

## Tests & coverage

Every collector is driven against fixture data through the OS seams, so the suite
proves **100% statement coverage including error branches** with no root and no
host dependence:

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

CI additionally builds and tests on the six supported 64-bit architectures
(amd64, arm64, riscv64, loong64, ppc64le, s390x) and on Linux, macOS and Windows.

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-facter/facter authors.
