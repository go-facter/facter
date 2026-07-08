// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"context"
	"net"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// env is the set of operating-system seams every collector draws on. Grouping
// them here — rather than calling os/exec/net directly from the collectors —
// makes each collector deterministic in tests: a test builds an env of fakes,
// wires it into a Collection with newWithEnv, and drives the real assembly and
// parsing logic against fixture bytes. The production seams are thin wrappers
// whose own branches are exercised by focused smoke tests, so the collectors'
// logic — including every error branch — is what the coverage gate measures.
type env struct {
	goos   string
	goarch string
	numCPU int

	readFile   func(string) ([]byte, error)
	readDir    func(string) ([]dirEntry, error)
	statMode   func(string) (os.FileMode, bool)
	runCmd     func(name string, args ...string) (string, error)
	hostname   func() (string, error)
	lookupEnv  func(string) (string, bool)
	interfaces func() ([]ifaceData, error)
	curUser    func() (userInfo, error)
	now        func() time.Time
	euid       func() int
}

// dirEntry is the OS-neutral projection of a directory entry the collectors need.
type dirEntry struct {
	Name  string
	IsDir bool
}

// ifaceData is the OS-neutral projection of a network interface.
type ifaceData struct {
	Name     string
	MAC      string
	MTU      int
	Up       bool
	Loopback bool
	Addrs    []ipAddr
}

// ipAddr is one address bound to an interface, in neutral textual form.
type ipAddr struct {
	IP     string
	Prefix int
	IsV4   bool
}

// userInfo is the OS-neutral projection of the current user.
type userInfo struct {
	Username string
	UID      string
	GID      string
	Group    string
}

// defaultEnv wires the production seams. Direct standard-library references
// (readFile, hostname, lookupEnv, now, euid) carry no coverage cost; the thin
// adapters carry their logic in neutral, fully tested helpers.
func defaultEnv() *env {
	return &env{
		goos:       runtime.GOOS,
		goarch:     runtime.GOARCH,
		numCPU:     runtime.NumCPU(),
		readFile:   os.ReadFile,
		readDir:    defaultReadDir,
		statMode:   defaultStatMode,
		runCmd:     defaultRunCmd,
		hostname:   os.Hostname,
		lookupEnv:  os.LookupEnv,
		interfaces: defaultInterfaces,
		curUser:    defaultCurUser,
		now:        time.Now,
		euid:       os.Geteuid,
	}
}

// commandTimeout bounds every external command a collector runs.
const commandTimeout = 5 * time.Second

// defaultRunCmd runs an external command and returns its standard output.
func defaultRunCmd(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// defaultReadDir adapts os.ReadDir to the neutral dirEntry projection.
func defaultReadDir(path string) ([]dirEntry, error) {
	ents, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := make([]dirEntry, 0, len(ents))
	for _, e := range ents {
		out = append(out, dirEntry{Name: e.Name(), IsDir: e.IsDir()})
	}
	return out, nil
}

// defaultStatMode reports a file's mode, and whether it could be stat'd.
func defaultStatMode(path string) (os.FileMode, bool) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, false
	}
	return fi.Mode(), true
}

// defaultInterfaces enumerates the host interfaces and adapts them to ifaceData.
func defaultInterfaces() ([]ifaceData, error) {
	ifaces, err := net.Interfaces()
	return adaptInterfaces(ifaces, err, netAddrsOf)
}

// netAddrsOf fetches the addresses of one interface.
func netAddrsOf(ifi net.Interface) ([]net.Addr, error) { return ifi.Addrs() }

// adaptInterfaces converts standard-library interfaces (and any enumeration
// error) to the neutral ifaceData slice, resolving each interface's addresses
// through addrsOf. It is neutral so its every branch is testable with fakes.
func adaptInterfaces(ifaces []net.Interface, err error, addrsOf func(net.Interface) ([]net.Addr, error)) ([]ifaceData, error) {
	if err != nil {
		return nil, err
	}
	out := make([]ifaceData, 0, len(ifaces))
	for _, ifi := range ifaces {
		d := ifaceData{
			Name:     ifi.Name,
			MAC:      ifi.HardwareAddr.String(),
			MTU:      ifi.MTU,
			Up:       ifi.Flags&net.FlagUp != 0,
			Loopback: ifi.Flags&net.FlagLoopback != 0,
		}
		addrs, aerr := addrsOf(ifi)
		if aerr == nil {
			for _, a := range addrs {
				if ipn, ok := a.(*net.IPNet); ok {
					d.Addrs = append(d.Addrs, toIPAddr(ipn))
				}
			}
		}
		out = append(out, d)
	}
	return out, nil
}

// toIPAddr converts a net.IPNet to the neutral ipAddr shape.
func toIPAddr(ipn *net.IPNet) ipAddr {
	prefix, _ := ipn.Mask.Size()
	if v4 := ipn.IP.To4(); v4 != nil {
		return ipAddr{IP: v4.String(), Prefix: prefix, IsV4: true}
	}
	return ipAddr{IP: ipn.IP.String(), Prefix: prefix, IsV4: false}
}

// defaultCurUser resolves the current user, looking group names up via the OS.
func defaultCurUser() (userInfo, error) {
	u, err := user.Current()
	return buildUser(u, err, lookupGroupName)
}

// lookupGroupName resolves a gid to a group name via the OS user database.
func lookupGroupName(gid string) (string, error) {
	g, err := user.LookupGroupId(gid)
	if err != nil {
		return "", err
	}
	return g.Name, nil
}

// buildUser converts a standard-library *user.User (and any lookup error) to the
// neutral userInfo, resolving the primary group through groupName. Neutral so
// every branch is testable.
func buildUser(u *user.User, err error, groupName func(string) (string, error)) (userInfo, error) {
	if err != nil {
		return userInfo{}, err
	}
	info := userInfo{Username: u.Username, UID: u.Uid, GID: u.Gid}
	if name, gerr := groupName(u.Gid); gerr == nil {
		info.Group = name
	}
	return info, nil
}

// readText reads a file through the seam and returns it as a string, reporting
// presence rather than the error, which collectors treat as "source absent".
func (e *env) readText(path string) (string, bool) {
	b, err := e.readFile(path)
	if err != nil {
		return "", false
	}
	return string(b), true
}

// cmd runs a command through the seam and reports success as a bool.
func (e *env) cmd(name string, args ...string) (string, bool) {
	out, err := e.runCmd(name, args...)
	if err != nil {
		return "", false
	}
	return out, true
}

// stringify renders a resolved fact value the way a Ruby caller expects to read
// it: strings verbatim, integers without a decimal point, booleans as true/false.
func stringify(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return ""
	}
}

// firstField returns the first whitespace-separated field of s, or "".
func firstField(s string) string {
	fs := strings.Fields(s)
	if len(fs) == 0 {
		return ""
	}
	return fs[0]
}
