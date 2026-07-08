// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func runArgs(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestRunFullDumpJSON(t *testing.T) {
	code, out, _ := runArgs(t)
	if code != 0 || !strings.Contains(out, `"facterversion"`) {
		t.Fatalf("full dump: code=%d out=%s", code, out)
	}
}

func TestRunSingleQuery(t *testing.T) {
	code, out, _ := runArgs(t, "facterversion")
	if code != 0 || !strings.Contains(out, "4.0.0") {
		t.Fatalf("query: code=%d out=%s", code, out)
	}
}

func TestRunYAML(t *testing.T) {
	code, out, _ := runArgs(t, "--yaml", "os")
	if code != 0 || !strings.Contains(out, "name:") {
		t.Fatalf("yaml: code=%d out=%s", code, out)
	}
}

func TestRunJSONFlag(t *testing.T) {
	code, out, _ := runArgs(t, "-j", "os.family")
	if code != 0 || out == "" {
		t.Fatalf("json flag: code=%d out=%s", code, out)
	}
}

func TestRunAbsentFact(t *testing.T) {
	code, out, _ := runArgs(t, "no.such.fact.here")
	if code != 1 || out != "" {
		t.Fatalf("absent: code=%d out=%q", code, out)
	}
}

func TestRunHelp(t *testing.T) {
	code, out, _ := runArgs(t, "-h")
	if code != 0 || !strings.Contains(out, "usage:") {
		t.Fatalf("help: code=%d out=%s", code, out)
	}
}

func TestRunUnknownFlag(t *testing.T) {
	code, _, errb := runArgs(t, "--bogus")
	if code != 2 || !strings.Contains(errb, "unknown flag") {
		t.Fatalf("unknown flag: code=%d err=%s", code, errb)
	}
}

func TestRunTwoQueries(t *testing.T) {
	code, _, errb := runArgs(t, "os.name", "kernel")
	if code != 2 || !strings.Contains(errb, "only one") {
		t.Fatalf("two queries: code=%d err=%s", code, errb)
	}
}

func TestEmitJSONError(t *testing.T) {
	var out, errb bytes.Buffer
	// A channel cannot be JSON-encoded: exercises emit's error branch.
	code := emit(make(chan int), "json", &out, &errb)
	if code != 1 || !strings.Contains(errb.String(), "facter:") {
		t.Fatalf("emit error: code=%d err=%s", code, errb.String())
	}
}

func TestMain_ExitSeam(t *testing.T) {
	oldExit, oldArgs := osExit, os.Args
	defer func() { osExit, os.Args = oldExit, oldArgs }()
	got := -1
	osExit = func(code int) { got = code }
	os.Args = []string{"facter", "facterversion"}
	main()
	if got != 0 {
		t.Fatalf("main exit code = %d", got)
	}
}
