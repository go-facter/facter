// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"net"
	"strconv"
	"strings"
)

// collectNetworking builds the structured networking fact: hostname / fqdn /
// domain, a map of interfaces (each with ip/ip6/mac/mtu/netmask/network and the
// bindings arrays Facter exposes), and the primary interface's summary fields.
func (c *Collection) collectNetworking() (any, bool) {
	short, domain, fqdn := c.hostIdentity()

	out := map[string]any{
		"hostname": short,
		"fqdn":     fqdn,
	}
	if domain != "" {
		out["domain"] = domain
	}

	ifaces, err := c.env.interfaces()
	if err != nil {
		// Hostname facts are still useful without interface enumeration.
		return out, true
	}

	ifaceMap := map[string]any{}
	var primaryName string
	var primary map[string]any
	for _, ifi := range ifaces {
		entry := interfaceFact(ifi)
		ifaceMap[ifi.Name] = entry
		if primary == nil && !ifi.Loopback && ifi.Up {
			if _, ok := entry["ip"]; ok {
				primaryName = ifi.Name
				primary = entry
			}
		}
	}
	if len(ifaceMap) > 0 {
		out["interfaces"] = ifaceMap
	}
	if primary != nil {
		out["primary"] = primaryName
		for _, k := range []string{"ip", "ip6", "mac", "mtu", "netmask", "network"} {
			if v, ok := primary[k]; ok {
				out[k] = v
			}
		}
	}
	return out, true
}

// interfaceFact renders one interface as its Facter map, splitting IPv4 and IPv6
// addresses into scalar summaries and the bindings / bindings6 arrays.
func interfaceFact(ifi ifaceData) map[string]any {
	entry := map[string]any{}
	if ifi.MAC != "" {
		entry["mac"] = ifi.MAC
	}
	if ifi.MTU != 0 {
		entry["mtu"] = ifi.MTU
	}
	var bindings, bindings6 []any
	for _, a := range ifi.Addrs {
		if a.IsV4 {
			mask := netmaskV4(a.Prefix)
			netw := networkAddr(a.IP, a.Prefix)
			if _, set := entry["ip"]; !set {
				entry["ip"] = a.IP
				entry["netmask"] = mask
				entry["network"] = netw
			}
			bindings = append(bindings, map[string]any{
				"address": a.IP, "netmask": mask, "network": netw,
			})
		} else {
			netw := networkAddr(a.IP, a.Prefix)
			if _, set := entry["ip6"]; !set {
				entry["ip6"] = a.IP
				entry["netmask6"] = prefix6(a.Prefix)
				entry["network6"] = netw
			}
			bindings6 = append(bindings6, map[string]any{
				"address": a.IP, "netmask": prefix6(a.Prefix), "network": netw,
			})
		}
	}
	if bindings != nil {
		entry["bindings"] = bindings
	}
	if bindings6 != nil {
		entry["bindings6"] = bindings6
	}
	return entry
}

// hostIdentity resolves the short hostname, DNS domain and fully-qualified name.
func (c *Collection) hostIdentity() (short, domain, fqdn string) {
	raw, err := c.env.hostname()
	if err != nil {
		raw = ""
	}
	if h, d, ok := strings.Cut(raw, "."); ok {
		short, domain = h, d
	} else {
		short = raw
		domain = c.dnsDomain()
	}
	if domain != "" {
		fqdn = short + "." + domain
	} else {
		fqdn = short
	}
	return short, domain, fqdn
}

// dnsDomain finds the host's DNS domain without a lookup: the resolv.conf
// domain/search directive on Unix, or the USERDNSDOMAIN environment on Windows.
func (c *Collection) dnsDomain() string {
	if c.env.goos == "windows" {
		if v, ok := c.env.lookupEnv("USERDNSDOMAIN"); ok {
			return strings.ToLower(v)
		}
		return ""
	}
	text, ok := c.env.readText("/etc/resolv.conf")
	if !ok {
		return ""
	}
	return parseResolvDomain(text)
}

// parseResolvDomain extracts the domain from a resolv.conf, preferring an
// explicit domain directive and falling back to the first search entry.
func parseResolvDomain(text string) string {
	search := ""
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "domain":
			return fields[1]
		case "search":
			if search == "" {
				search = fields[1]
			}
		}
	}
	return search
}

// netmaskV4 renders an IPv4 prefix length as a dotted-decimal mask.
func netmaskV4(prefix int) string {
	mask := net.CIDRMask(prefix, 32)
	if len(mask) != 4 {
		return ""
	}
	ip := net.IP(mask)
	return ip.String()
}

// prefix6 renders an IPv6 prefix length back as a CIDR suffix string.
func prefix6(prefix int) string {
	return "/" + strconv.Itoa(prefix)
}

// networkAddr computes the network address of ip under the given prefix length.
func networkAddr(ip string, prefix int) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}
	bits := 32
	if parsed.To4() == nil {
		bits = 128
	}
	mask := net.CIDRMask(prefix, bits)
	return parsed.Mask(mask).String()
}
