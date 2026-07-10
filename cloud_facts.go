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
	hay, assetTag := c.dmiCloudHints()
	switch {
	case strings.Contains(hay, "amazon"), strings.Contains(hay, "ec2"):
		return "aws"
	case strings.Contains(hay, "google"):
		return "gce"
	case assetTag == azureAssetTag:
		return "azure"
	}
	// DMI inconclusive: fall back to a bounded metadata probe.
	if _, ok := c.env.httpGet(ec2MetadataBase, nil); ok {
		return "aws"
	}
	if _, ok := c.env.httpGet(azureMetadata, map[string]string{"Metadata": "true"}); ok {
		return "azure"
	}
	return ""
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
