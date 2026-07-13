// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"strings"

	"golang.org/x/crypto/ssh"
)

// parseAuthorizedKey extracts the algorithm, size in bits, and trailing
// comment from a single OpenSSH-format public key line. ok is false when the
// line cannot be parsed so weak-key audits can distinguish "unparseable" from
// a known algorithm.
func parseAuthorizedKey(line string) (algorithm string, bits int64, comment string, ok bool) {
	pub, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(line))
	if err != nil || pub == nil {
		return "", 0, "", false
	}
	algorithm = pub.Type()
	if ck, isCrypto := pub.(ssh.CryptoPublicKey); isCrypto {
		switch k := ck.CryptoPublicKey().(type) {
		case *rsa.PublicKey:
			bits = int64(k.N.BitLen())
		case *ecdsa.PublicKey:
			bits = int64(k.Curve.Params().BitSize)
		case ed25519.PublicKey:
			bits = 256
		}
	}
	return algorithm, bits, comment, true
}

// parseInstanceSSHKeys turns the GCE instance metadata `ssh-keys` value into
// one dict per configured key. Each metadata line has the form
// `<username>:<openssh-public-key>`; unparseable lines are skipped.
func parseInstanceSSHKeys(raw string) []any {
	out := []any{}
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// A metadata line is `<username>:<key>`. The key itself never
		// contains a colon before its first space, so split on the first
		// colon to peel off the username, then parse the remainder as an
		// authorized key.
		username, keyPart, found := strings.Cut(line, ":")
		if !found {
			username, keyPart = "", line
		}
		algorithm, bits, comment, ok := parseAuthorizedKey(keyPart)
		if !ok {
			continue
		}
		out = append(out, map[string]any{
			"username":  username,
			"algorithm": algorithm,
			"bits":      bits,
			"publicKey": strings.TrimSpace(keyPart),
			"comment":   comment,
		})
	}
	return out
}
