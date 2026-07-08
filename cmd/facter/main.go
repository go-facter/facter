// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

// Command facter is a pure-Go drop-in for Puppet's facter(8): with no argument
// it prints the full fact set, and with a dotted path it prints a single fact.
// Output is JSON by default and YAML with --yaml.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/go-facter/facter"
)

// osExit is the process-exit seam, swapped in tests so run's exit codes are
// observable without terminating the test binary.
var osExit = os.Exit

func main() { osExit(run(os.Args[1:], os.Stdout, os.Stderr)) }

// run is the testable CLI core. It parses flags, resolves either the full fact
// set or a single dotted-path query, and writes the chosen encoding to stdout.
func run(args []string, stdout, stderr io.Writer) int {
	format := "json"
	var query string
	for _, a := range args {
		switch a {
		case "--json", "-j":
			format = "json"
		case "--yaml", "-y":
			format = "yaml"
		case "-h", "--help":
			fmt.Fprintln(stdout, usage)
			return 0
		default:
			if len(a) > 0 && a[0] == '-' {
				fmt.Fprintf(stderr, "facter: unknown flag %q\n", a)
				return 2
			}
			if query != "" {
				fmt.Fprintln(stderr, "facter: only one fact query is supported")
				return 2
			}
			query = a
		}
	}

	c := facter.New()

	var value any
	if query == "" {
		value = c.ToHash()
	} else {
		v, ok := c.Value(query)
		if !ok {
			return 1 // absent fact: no output, non-zero status, like facter(8)
		}
		value = v
	}

	return emit(value, format, stdout, stderr)
}

// emit writes value in the requested format.
func emit(value any, format string, stdout, stderr io.Writer) int {
	switch format {
	case "yaml":
		fmt.Fprint(stdout, facter.MarshalYAML(value))
	default:
		out, err := facter.MarshalJSON(value)
		if err != nil {
			fmt.Fprintf(stderr, "facter: %v\n", err)
			return 1
		}
		fmt.Fprint(stdout, out)
	}
	return 0
}

const usage = `usage: facter [--json|--yaml] [fact.path]

  no argument   print all facts
  fact.path     print a single fact by dotted path, e.g. os.name
  --json, -j    JSON output (default)
  --yaml, -y    YAML output`
