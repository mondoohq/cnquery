// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"

	"golang.org/x/crypto/ssh"
)

// parseSSHPublicKey extracts the key algorithm and size in bits from an
// OpenSSH-format public key. It returns ("", 0) when the key cannot be parsed
// so weak-key audits can distinguish "unparseable" from a known algorithm.
func parseSSHPublicKey(publicKey string) (algorithm string, bits int64) {
	pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil || pub == nil {
		return "", 0
	}
	algorithm = pub.Type()
	ck, ok := pub.(ssh.CryptoPublicKey)
	if !ok {
		return algorithm, 0
	}
	switch k := ck.CryptoPublicKey().(type) {
	case *rsa.PublicKey:
		bits = int64(k.N.BitLen())
	case *ecdsa.PublicKey:
		bits = int64(k.Curve.Params().BitSize)
	case ed25519.PublicKey:
		bits = 256
	}
	return algorithm, bits
}
