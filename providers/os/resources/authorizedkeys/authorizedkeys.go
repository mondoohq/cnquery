// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package authorizedkeys

import (
	"bufio"
	"crypto/dsa" //nolint:staticcheck // DSA is deprecated, but we still detect legacy DSA keys for crypto-posture auditing
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/base64"
	"io"
	"strings"

	"golang.org/x/crypto/ssh"
)

// most ssh keys include base64 padding, so lets use it too (not default in Go)
var RawStdEncoding = base64.StdEncoding.WithPadding(base64.StdPadding)

type Entry struct {
	Line    int64
	Key     ssh.PublicKey
	Label   string
	Options []string
}

func (e Entry) Base64Key() string {
	return RawStdEncoding.EncodeToString(e.Key.Marshal())
}

// Bits returns the key size in bits of the entry's public key. It unwraps SSH
// certificates to inspect the underlying key and returns 0 when the key type is
// unknown or does not expose an underlying crypto public key (e.g., security-key
// backed keys that do not implement ssh.CryptoPublicKey).
func (e Entry) Bits() int64 {
	key := e.Key
	if cert, ok := key.(*ssh.Certificate); ok {
		key = cert.Key
	}

	cryptoKey, ok := key.(ssh.CryptoPublicKey)
	if !ok {
		return 0
	}

	switch k := cryptoKey.CryptoPublicKey().(type) {
	case *rsa.PublicKey:
		return int64(k.N.BitLen())
	case *ecdsa.PublicKey:
		return int64(k.Curve.Params().BitSize)
	case ed25519.PublicKey:
		return 256
	case *dsa.PublicKey:
		return int64(k.P.BitLen())
	default:
		return 0
	}
}

func Parse(r io.Reader) ([]Entry, error) {
	res := []Entry{}
	scanner := bufio.NewScanner(r)

	// lineNo tracks the physical 1-based line in the file, so it must advance
	// for skipped blank/comment lines too — Entry.Line is meant to locate the
	// key in the file, not count key entries.
	lineNo := int64(0)
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()

		in := strings.TrimSpace(line)
		if len(in) == 0 || in[0] == '#' {
			continue
		}

		key, comment, options, _, err := ssh.ParseAuthorizedKey([]byte(line))
		if err != nil {
			return nil, err
		}

		res = append(res, Entry{
			Line:    lineNo,
			Key:     key,
			Label:   comment,
			Options: options,
		})
	}
	return res, nil
}
