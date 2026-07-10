// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "testing"

const cpuinfo2x2 = `processor	: 0
model name	: Intel(R) Core(TM) i7
physical id	: 0
processor	: 1
model name	: Intel(R) Core(TM) i7
physical id	: 0
bogus
`

func TestProcessorsLinux(t *testing.T) {
	f := fakeEnv{goos: "linux", files: map[string]string{"/proc/cpuinfo": cpuinfo2x2}, cmds: map[string]string{"uname -m": "x86_64\n"}}
	c := f.collection()
	if v, _ := c.Value("processors.count"); v != 2 {
		t.Fatalf("count = %v", v)
	}
	if v, _ := c.Value("processors.physicalcount"); v != 1 {
		t.Fatalf("physical = %v", v)
	}
	if v, _ := c.Value("processors.isa"); v != "x86_64" {
		t.Fatalf("isa = %v", v)
	}
	if v, _ := c.Value("processorcount"); v != 2 {
		t.Fatalf("legacy processorcount = %v", v)
	}
	models, _ := c.Value("processors.models")
	if ms, ok := models.([]any); !ok || len(ms) != 2 {
		t.Fatalf("models = %v", models)
	}
}

func TestProcessorsLinuxEmpty(t *testing.T) {
	// No cpuinfo: falls back to numCPU and default model/physical.
	f := fakeEnv{goos: "linux", numCPU: 8}
	c := f.collection()
	if v, _ := c.Value("processors.count"); v != 8 {
		t.Fatalf("fallback count = %v", v)
	}
	if v, _ := c.Value("processors.physicalcount"); v != 1 {
		t.Fatalf("fallback physical = %v", v)
	}
	models, _ := c.Value("processors.models")
	if ms := models.([]any); len(ms) != 8 || ms[0] != "unknown" {
		t.Fatalf("fallback models = %v", models)
	}
}

func TestParseCPUInfoNoPhysical(t *testing.T) {
	count, models, physical := parseCPUInfo("processor : 0\nmodel name : X\n")
	if count != 1 || len(models) != 1 || physical != 1 {
		t.Fatalf("got %d %v %d", count, models, physical)
	}
	// Fully empty input.
	count, _, physical = parseCPUInfo("")
	if count != 0 || physical != 0 {
		t.Fatalf("empty cpuinfo got %d %d", count, physical)
	}
}

func TestProcessorsDarwin(t *testing.T) {
	f := fakeEnv{goos: "darwin", goarch: "arm64", cmds: map[string]string{
		"sysctl -n hw.logicalcpu":            "10\n",
		"sysctl -n hw.physicalcpu":           "10\n",
		"sysctl -n machdep.cpu.brand_string": "Apple M1 Max\n",
		"uname -m":                           "arm64\n",
	}}
	c := f.collection()
	if v, _ := c.Value("processors.count"); v != 10 {
		t.Fatalf("darwin count = %v", v)
	}
	models, _ := c.Value("processors.models")
	if ms := models.([]any); ms[0] != "Apple M1 Max" {
		t.Fatalf("darwin model = %v", ms[0])
	}
}

func TestProcessorsDarwinDefaults(t *testing.T) {
	// No sysctl output: count falls back to numCPU, brand unknown.
	f := fakeEnv{goos: "darwin", goarch: "arm64", numCPU: 6}
	c := f.collection()
	if v, _ := c.Value("processors.count"); v != 6 {
		t.Fatalf("darwin fallback count = %v", v)
	}
	models, _ := c.Value("processors.models")
	if ms := models.([]any); ms[0] != "unknown" {
		t.Fatalf("darwin fallback model = %v", ms[0])
	}
}

func TestProcessorsDarwinBrandBlank(t *testing.T) {
	// brand command returns whitespace -> stays "unknown".
	f := fakeEnv{goos: "darwin", cmds: map[string]string{
		"sysctl -n hw.logicalcpu":            "2\n",
		"sysctl -n machdep.cpu.brand_string": "   \n",
	}}
	c := f.collection()
	models, _ := c.Value("processors.models")
	if ms := models.([]any); ms[0] != "unknown" {
		t.Fatalf("blank brand = %v", ms[0])
	}
}

func TestProcessorsWindows(t *testing.T) {
	f := fakeEnv{goos: "windows", envv: map[string]string{
		"NUMBER_OF_PROCESSORS": "16",
		"PROCESSOR_IDENTIFIER": "Intel64 Family 6",
	}}
	c := f.collection()
	if v, _ := c.Value("processors.count"); v != 16 {
		t.Fatalf("win count = %v", v)
	}
	models, _ := c.Value("processors.models")
	if ms := models.([]any); ms[0] != "Intel64 Family 6" {
		t.Fatalf("win model = %v", ms[0])
	}
}

func TestProcessorsWindowsDefaults(t *testing.T) {
	f := fakeEnv{goos: "windows", numCPU: 3, envv: map[string]string{"PROCESSOR_IDENTIFIER": "  "}}
	c := f.collection()
	if v, _ := c.Value("processors.count"); v != 3 {
		t.Fatalf("win fallback count = %v", v)
	}
	models, _ := c.Value("processors.models")
	if ms := models.([]any); ms[0] != "unknown" {
		t.Fatalf("win fallback model = %v", ms[0])
	}
}

const cpuinfoExtras = `processor	: 0
model name	: Intel(R) Xeon
physical id	: 0
cpu cores	: 4
siblings	: 8
cpu MHz		: 2400.000
processor	: 1
model name	: Intel(R) Xeon
physical id	: 0
cpu cores	: 4
siblings	: 8
cpu MHz		: 2400.000
`

func TestProcessorsExtrasLinux(t *testing.T) {
	c := (fakeEnv{goos: "linux", files: map[string]string{"/proc/cpuinfo": cpuinfoExtras}}).collection()
	if v, _ := c.Value("processors.cores_per_socket"); v != 4 {
		t.Errorf("cores_per_socket = %v", v)
	}
	if v, _ := c.Value("processors.threads_per_core"); v != 2 {
		t.Errorf("threads_per_core = %v", v)
	}
	if v, _ := c.Value("processors.speed"); v != "2.40 GHz" {
		t.Errorf("speed = %v", v)
	}
}

func TestProcessorsExtrasAbsent(t *testing.T) {
	// cpuinfo without cores/siblings/MHz: the refinements are omitted.
	c := (fakeEnv{goos: "linux", files: map[string]string{"/proc/cpuinfo": cpuinfo2x2}}).collection()
	if _, ok := c.Value("processors.cores_per_socket"); ok {
		t.Error("cores_per_socket should be omitted")
	}
	if _, ok := c.Value("processors.speed"); ok {
		t.Error("speed should be omitted")
	}
}

func TestParseCPUExtra(t *testing.T) {
	// siblings < cores: threads_per_core not derivable.
	ex := parseCPUExtra("cpu cores\t: 8\nsiblings\t: 4\n")
	if ex.coresPerSocket != 8 || ex.threadsPerCore != 0 {
		t.Fatalf("mismatched topology = %+v", ex)
	}
	// no fields at all.
	if ex := parseCPUExtra("model\t: x\n"); ex.coresPerSocket != 0 || ex.speed != "" {
		t.Fatalf("empty = %+v", ex)
	}
}

func TestCPUExtraDarwin(t *testing.T) {
	c := (fakeEnv{goos: "darwin", cmds: map[string]string{
		"sysctl -n hw.physicalcpu":  "8\n",
		"sysctl -n hw.logicalcpu":   "16\n",
		"sysctl -n hw.cpufrequency": "2400000000\n",
	}}).collection()
	if v, _ := c.Value("processors.cores_per_socket"); v != 8 {
		t.Errorf("darwin cores = %v", v)
	}
	if v, _ := c.Value("processors.threads_per_core"); v != 2 {
		t.Errorf("darwin threads = %v", v)
	}
	if v, _ := c.Value("processors.speed"); v != "2.40 GHz" {
		t.Errorf("darwin speed = %v", v)
	}
}

func TestCPUExtraDarwinAppleSilicon(t *testing.T) {
	// No hw.cpufrequency (Apple Silicon): speed omitted, topology still derived.
	c := (fakeEnv{goos: "darwin", cmds: map[string]string{
		"sysctl -n hw.physicalcpu": "10\n",
		"sysctl -n hw.logicalcpu":  "10\n",
	}}).collection()
	if v, _ := c.Value("processors.threads_per_core"); v != 1 {
		t.Errorf("threads = %v", v)
	}
	if _, ok := c.Value("processors.speed"); ok {
		t.Error("speed should be omitted without hw.cpufrequency")
	}
}

func TestFormatHz(t *testing.T) {
	cases := map[uint64]string{
		2_400_000_000: "2.40 GHz",
		800_000_000:   "800 MHz",
		500:           "",
	}
	for hz, want := range cases {
		if got := formatHz(hz); got != want {
			t.Errorf("formatHz(%d) = %q, want %q", hz, got, want)
		}
	}
}

func TestRepeatModelFloor(t *testing.T) {
	if got := repeatModel("x", 0); len(got) != 1 {
		t.Fatalf("repeatModel(0) = %v", got)
	}
}

func TestAtoiSafe(t *testing.T) {
	if atoiSafe("42") != 42 || atoiSafe("bad") != 0 {
		t.Fatal("atoiSafe")
	}
}
