// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import "strings"

// EC2/Azure link-local metadata endpoints. Probes against these run through the
// bounded, non-blocking httpGet seam, so a host that is not on the provider does
// not stall: an unreachable endpoint simply means "not that provider".
const (
	ec2MetadataBase = "http://169.254.169.254/latest/meta-data/"
	azureMetadata   = "http://169.254.169.254/metadata/instance?api-version=2021-02-01"
	azureAssetTag   = "7783-7084-3265-9085-8269-3286-77"
)

// collectCloud builds the cloud fact — {provider: aws|gce|azure} — identifying the
// hosting provider primarily from DMI (deterministic, no network) and, only when
// DMI is inconclusive, from a bounded metadata probe. It is absent on a host that
// is not recognisably on a supported cloud, so it never guesses.
func (c *Collection) collectCloud() (any, bool) {
	provider := c.cloudProvider()
	if provider == "" {
		return nil, false
	}
	return map[string]any{"provider": provider}, true
}

// cloudProvider returns the detected provider name, or "" when none is identified.
func (c *Collection) cloudProvider() string {
	// DMI fingerprints are deterministic and need no network, so they resolve
	// the provider on every real cloud instance without a probe.
	if p := c.dmiProvider(); p != "" {
		return p
	}
	// DMI is inconclusive. Only fall back to a bounded metadata probe when the
	// host is plausibly a cloud instance: a bare-metal host must never stall
	// waiting on a link-local endpoint that will never answer.
	if !c.plausiblyCloud() {
		return ""
	}
	if _, ok := c.env.httpGet(ec2MetadataBase, nil); ok {
		return "aws"
	}
	if _, ok := c.env.httpGet(azureMetadata, map[string]string{"Metadata": "true"}); ok {
		return "azure"
	}
	return ""
}

// dmiProvider identifies the hosting provider from DMI fingerprints alone —
// deterministic and network-free — returning "" when DMI carries no cloud hint.
func (c *Collection) dmiProvider() string {
	hay, assetTag := c.dmiCloudHints()
	switch {
	case strings.Contains(hay, "amazon"), strings.Contains(hay, "ec2"):
		return "aws"
	case strings.Contains(hay, "google"):
		return "gce"
	case assetTag == azureAssetTag:
		return "azure"
	default:
		return ""
	}
}

// plausiblyCloud reports whether the host could be a cloud instance, gating the
// metadata network probes behind a no-network signal: the is_virtual fact (which
// itself keys off container markers, DMI hypervisor strings and the CPU
// hypervisor flag). A bare-metal, non-virtualised host is never plausibly cloud,
// so its cloud and ec2_metadata facts resolve to absent with zero network wait —
// exactly what Facter does by gating cloud facts behind virtual detection.
func (c *Collection) plausiblyCloud() bool {
	// is_virtual is a built-in fact and always resolves to a bool, so a missing
	// or non-bool value degrades safely to "not cloud".
	v, _ := c.Value("is_virtual")
	b, _ := v.(bool)
	return b
}

// dmiCloudHints returns a lower-cased haystack of the DMI vendor/product strings
// and the chassis asset tag, the fingerprints cloud providers leave in DMI.
func (c *Collection) dmiCloudHints() (haystack, assetTag string) {
	v, ok := c.Value("dmi")
	if !ok {
		return "", ""
	}
	m := v.(map[string]any)
	var b strings.Builder
	b.WriteString(mapString(m, "manufacturer"))
	b.WriteByte(' ')
	b.WriteString(nestedString(m, "product", "name"))
	b.WriteByte(' ')
	b.WriteString(nestedString(m, "bios", "vendor"))
	b.WriteByte(' ')
	b.WriteString(nestedString(m, "board", "manufacturer"))
	return strings.ToLower(b.String()), nestedString(m, "chassis", "asset_tag")
}

// collectEC2Metadata crawls the EC2 instance-metadata tree (one level of the
// link-local IMDS index) into a flat map. It is best-effort and environment
// dependent: absent unless the IMDS endpoint answers, so it never blocks.
func (c *Collection) collectEC2Metadata() (any, bool) {
	// The IMDS endpoint only exists on cloud instances, so gate the crawl behind
	// the same no-network plausibility check: a bare-metal host makes zero HTTP
	// calls and the fact is simply absent.
	if !c.plausiblyCloud() {
		return nil, false
	}
	index, ok := c.env.httpGet(ec2MetadataBase, nil)
	if !ok {
		return nil, false
	}
	out := map[string]any{}
	for _, name := range strings.Fields(index) {
		if strings.HasSuffix(name, "/") {
			continue // a sub-tree; a shallow crawl records only leaf keys
		}
		if val, ok := c.env.httpGet(ec2MetadataBase+name, nil); ok {
			out[name] = strings.TrimSpace(val)
		}
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

// mapString reads a string member of m, returning "" when absent or not a string.
func mapString(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}
