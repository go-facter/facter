// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// sshKeyKind names one host-key algorithm: the fact key Facter groups it under,
// the public-key file basename, and the SSHFP algorithm number the fingerprint
// strings carry.
type sshKeyKind struct {
	fact     string // ssh sub-map key: rsa/dsa/ecdsa/ed25519
	file     string // /etc/ssh basename
	sshfpNum string // SSHFP algorithm number (1=RSA 2=DSA 3=ECDSA 4=Ed25519)
}

// sshKeyKinds enumerates the host-key algorithms in the deterministic order the
// legacy aliases and the structured map are built from.
var sshKeyKinds = []sshKeyKind{
	{"dsa", "ssh_host_dsa_key.pub", "2"},
	{"rsa", "ssh_host_rsa_key.pub", "1"},
	{"ecdsa", "ssh_host_ecdsa_key.pub", "3"},
	{"ed25519", "ssh_host_ed25519_key.pub", "4"},
}

// collectSSH builds the structured ssh fact: for each host key present under
// /etc/ssh, a sub-map with the key blob, its type string, and the SHA1/SHA256
// SSHFP fingerprints — computed in pure Go from the decoded key, so no ssh-keygen
// subprocess is needed. Absent when the host publishes no readable host keys.
func (c *Collection) collectSSH() (any, bool) {
	out := map[string]any{}
	for _, k := range sshKeyKinds {
		if entry, ok := c.sshKeyEntry(k); ok {
			out[k.fact] = entry
		}
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

// sshKeyEntry reads and renders one host key's Facter sub-map.
func (c *Collection) sshKeyEntry(k sshKeyKind) (map[string]any, bool) {
	text, ok := c.env.readText("/etc/ssh/" + k.file)
	if !ok {
		return nil, false
	}
	typ, blob, ok := parsePubKey(text)
	if !ok {
		return nil, false
	}
	raw, err := base64.StdEncoding.DecodeString(blob)
	if err != nil {
		return nil, false
	}
	sha1sum := sha1.Sum(raw)
	sha256sum := sha256.Sum256(raw)
	return map[string]any{
		"type": typ,
		"key":  blob,
		"fingerprints": map[string]any{
			"sha1":   "SSHFP " + k.sshfpNum + " 1 " + hex.EncodeToString(sha1sum[:]),
			"sha256": "SSHFP " + k.sshfpNum + " 2 " + hex.EncodeToString(sha256sum[:]),
		},
	}, true
}

// parsePubKey splits an OpenSSH public-key line into its type and base64 blob,
// tolerating leading options and a trailing comment.
func parsePubKey(text string) (typ, blob string, ok bool) {
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		return fields[0], fields[1], true
	}
	return "", "", false
}

// collectSSHLegacyKey resolves a legacy sshXXXkey fact to the bare key blob.
func (c *Collection) collectSSHLegacyKey(fact string) (any, bool) {
	return c.Value("ssh." + fact + ".key")
}

// collectSSHFP resolves a legacy sshfp_XXX fact to the newline-joined SHA1 and
// SHA256 SSHFP lines, matching Facter's flat fingerprint fact.
func (c *Collection) collectSSHFP(fact string) (any, bool) {
	v, ok := c.Value("ssh." + fact + ".fingerprints")
	if !ok {
		return nil, false
	}
	fps := v.(map[string]any)
	lines := []string{fps["sha1"].(string), fps["sha256"].(string)}
	return strings.Join(lines, "\n"), true
}
