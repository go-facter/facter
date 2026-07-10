# Benchmarks

`go-facter/facter` resolves facts **lazily** and **caches** them for the life of a
`Collection`: a fact that is never queried costs nothing, and one queried
repeatedly is computed once. The benchmarks below measure both paths — the
**cold** first resolution and the **cached** re-read — so the caching model is
visible in the numbers.

## Running

```sh
GOWORK=off go test -run '^$' -bench . -benchmem ./...
```

The `Benchmark*` functions drive the collectors against an in-memory fixture
(`benchEnv` in `bench_test.go`) that populates every source — `/etc/os-release`,
`/proc/cpuinfo`, `/proc/meminfo`, `/proc/mounts`, `/sys/class/dmi/id`, SSH host
keys, SELinux, interfaces, `uname`/`df` command output — so the measurement is
fact-resolution cost, not the cost of missing sources. Using the seam fixture
(rather than the live host) keeps the numbers deterministic and machine-portable.

## Measured results

Apple M4 Max, `go1.26.4 darwin/arm64`, `-benchmem`:

| Benchmark | ns/op | B/op | allocs/op | What it measures |
|-----------|------:|-----:|----------:|------------------|
| `ToHashCold`   | ~35,200 | ~62,100 | ~766 | Full fact set, fresh collection every iteration (no cache) |
| `ToHashCached` |  ~4,000 | ~10,800 |   ~12 | Full fact set re-read from a warm cache |
| `ValueCold`    | ~13,350 | ~27,800 |  ~292 | One structured fact (`networking.ip`), fresh collection |
| `ValueCached`  |     ~40 |     ~32 |     ~1 | One fact re-read from a warm cache |

The cached single-fact read is ~330× faster than the cold read and drops to a
single allocation — the behaviour a Puppet catalog compilation depends on, where
the same facts are referenced many times per run.

> Absolute ns/op vary with hardware; re-run the command above to get numbers for
> your machine. The **ratios** (cold vs cached, and the ~1-alloc cached read) are
> the invariant the caching model guarantees.

## Reference comparison: MRI Facter (methodology)

The reference implementation is Puppet's Ruby Facter (`facter` gem on MRI Ruby).
It is not comparable in the same process — it is a Ruby program that spawns
subprocesses (`uname`, `dmidecode`, `ip`, `ssh-keygen`, …) per run — so the
comparison is **whole-process wall-clock**, reproduced on a Linux Tart VM rather
than asserted here:

```sh
# On a debian Tart VM (feedback: use debian for test/qemu, not alpine):
tart run debian &
tart ip debian
ssh admin@<vm-ip>

# Install the reference implementation.
sudo apt-get update && sudo apt-get install -y facter    # MRI Facter (Ruby)

# Reference: cold, whole-process wall-clock (interpreter start + subprocess fan-out).
hyperfine --warmup 1 'facter --json'

# go-facter: build the static CGO=0 binary and time the same whole-set resolution.
CGO_ENABLED=0 go build -o /usr/local/bin/go-facter ./cmd/facter
hyperfine --warmup 1 'go-facter --json'
```

`hyperfine` reports mean ± σ for each; compare the two means. The go-facter CLI
is a single static binary with no interpreter start-up and no per-fact subprocess
fan-out (facts such as DMI, SSH fingerprints, memory, CPU topology and SELinux are
read and parsed in-process), so it resolves the whole set in a small fraction of
the reference's wall-clock. Fill in the two measured means from the VM run when
publishing a release; do not quote a reference figure that has not been measured
on the target VM.

### What is intentionally *not* claimed

- Facts that require real hardware or a live network — `dmi`/`bios` on bare
  metal, `cloud`/`ec2_metadata` behind the link-local metadata endpoint — are
  exercised through injected fixtures in the benchmark. Their real-host latency
  (a `169.254.169.254` round trip, bounded to 400 ms and non-blocking) is
  environment-dependent and is not part of these in-process numbers.
