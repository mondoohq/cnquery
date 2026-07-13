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

// parseAuthorizedKeys turns a newline-separated list of OpenSSH public keys
// (the format of the instance metadata `ssh_authorized_keys` value) into one
// dict per key. Blank and unparseable lines are skipped so weak-key audits
// only see keys whose algorithm and size are known.
func parseAuthorizedKeys(raw string) []any {
	out := []any{}
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pub, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(line))
		if err != nil || pub == nil {
			continue
		}
		var bits int64
		if ck, ok := pub.(ssh.CryptoPublicKey); ok {
			switch k := ck.CryptoPublicKey().(type) {
			case *rsa.PublicKey:
				bits = int64(k.N.BitLen())
			case *ecdsa.PublicKey:
				bits = int64(k.Curve.Params().BitSize)
			case ed25519.PublicKey:
				bits = 256
			}
		}
		out = append(out, map[string]any{
			"algorithm": pub.Type(),
			"bits":      bits,
			"publicKey": line,
			"comment":   comment,
		})
	}
	return out
}
