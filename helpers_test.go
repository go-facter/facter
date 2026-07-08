// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// fakeEnv is a controllable operating-system environment for driving collectors
// against fixture data. Zero values are inert; set only the fields a test needs.
type fakeEnv struct {
	goos    string
	goarch  string
	numCPU  int
	files   map[string]string      // path -> contents (present)
	dirs    map[string][]dirEntry  // path -> entries
	dirErr  map[string]bool        // path -> readDir fails
	modes   map[string]os.FileMode // path -> stat mode (present)
	cmds    map[string]string      // "name arg1 arg2" -> stdout (success)
	host    string
	hostErr bool
	envv    map[string]string
	user    userInfo
	userErr bool
	now     time.Time
	euid    int
	ifaces  []ifaceData
	ifErr   bool
}

// env materialises a *env whose seams are backed by the fakeEnv maps.
func (f fakeEnv) env() *env {
	goos := f.goos
	if goos == "" {
		goos = "linux"
	}
	goarch := f.goarch
	if goarch == "" {
		goarch = "amd64"
	}
	numCPU := f.numCPU
	if numCPU == 0 {
		numCPU = 4
	}
	now := f.now
	if now.IsZero() {
		now = time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	}
	return &env{
		goos:   goos,
		goarch: goarch,
		numCPU: numCPU,
		readFile: func(p string) ([]byte, error) {
			if s, ok := f.files[p]; ok {
				return []byte(s), nil
			}
			return nil, fmt.Errorf("no such file: %s", p)
		},
		readDir: func(p string) ([]dirEntry, error) {
			if f.dirErr[p] {
				return nil, fmt.Errorf("readdir failed: %s", p)
			}
			return f.dirs[p], nil
		},
		statMode: func(p string) (os.FileMode, bool) {
			if m, ok := f.modes[p]; ok {
				return m, true
			}
			return 0, false
		},
		runCmd: func(name string, args ...string) (string, error) {
			key := strings.Join(append([]string{name}, args...), " ")
			if out, ok := f.cmds[key]; ok {
				return out, nil
			}
			return "", fmt.Errorf("command failed: %s", key)
		},
		hostname: func() (string, error) {
			if f.hostErr {
				return "", fmt.Errorf("hostname failed")
			}
			return f.host, nil
		},
		lookupEnv: func(k string) (string, bool) {
			v, ok := f.envv[k]
			return v, ok
		},
		interfaces: func() ([]ifaceData, error) {
			if f.ifErr {
				return nil, fmt.Errorf("interfaces failed")
			}
			return f.ifaces, nil
		},
		curUser: func() (userInfo, error) {
			if f.userErr {
				return userInfo{}, fmt.Errorf("user failed")
			}
			return f.user, nil
		},
		now:  func() time.Time { return now },
		euid: func() int { return f.euid },
	}
}

// collection wires a fakeEnv into a fresh Collection with the core facts.
func (f fakeEnv) collection() *Collection { return newWithEnv(f.env()) }
