// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"golang.org/x/crypto/ssh"
)

// pkcs8PEM marshals a private key to an unencrypted PKCS#8 PEM block.
func pkcs8PEM(t *testing.T, key any) string {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

// opensshPEM marshals a private key to an OpenSSH-format PEM block. This is the
// default output of `ssh-keygen`, and unlike PKCS#8 it makes ParseRawPrivateKey
// return a *pointer* (e.g. *ed25519.PrivateKey), which the introspection
// switches must handle alongside the PKCS#8 value type.
func opensshPEM(t *testing.T, key any) string {
	t.Helper()
	block, err := ssh.MarshalPrivateKey(key, "")
	require.NoError(t, err)
	return string(pem.EncodeToMemory(block))
}

// newPrivatekey builds a privatekey resource whose static pem field is set,
// so the computed publicKeyAlgorithm/publicKeyBits helpers can be exercised
// without a live connection.
func newPrivatekey(pemStr string) *mqlPrivatekey {
	return &mqlPrivatekey{
		Pem: plugin.TValue[string]{Data: pemStr, State: plugin.StateIsSet},
	}
}

func TestPrivatekeyPublicKeyIntrospection(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	_, edKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	tests := []struct {
		name          string
		pem           string
		wantAlgorithm string
		wantBits      int64
	}{
		{
			name:          "RSA-2048",
			pem:           pkcs8PEM(t, rsaKey),
			wantAlgorithm: "RSA",
			wantBits:      2048,
		},
		{
			name:          "ECDSA P-256",
			pem:           pkcs8PEM(t, ecKey),
			wantAlgorithm: "ECDSA",
			wantBits:      256,
		},
		{
			name:          "Ed25519",
			pem:           pkcs8PEM(t, edKey),
			wantAlgorithm: "Ed25519",
			wantBits:      256,
		},
		{
			// OpenSSH format yields a *ed25519.PrivateKey (pointer), unlike the
			// PKCS#8 case above which yields the value type. Both must resolve.
			name:          "Ed25519 (OpenSSH)",
			pem:           opensshPEM(t, edKey),
			wantAlgorithm: "Ed25519",
			wantBits:      256,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pk := newPrivatekey(tc.pem)

			algo, err := pk.publicKeyAlgorithm()
			require.NoError(t, err)
			require.Equal(t, tc.wantAlgorithm, algo)

			bits, err := pk.publicKeyBits()
			require.NoError(t, err)
			require.Equal(t, tc.wantBits, bits)
		})
	}
}

func TestPrivatekeyEncryptedKeyGraceful(t *testing.T) {
	// An encrypted (passphrase-protected) key cannot be introspected without
	// the passphrase and must yield empty/zero values without erroring.
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	require.NoError(t, err)
	//nolint:staticcheck // x509.EncryptPEMBlock is deprecated but still the
	// simplest way to produce an encrypted PEM fixture for this test.
	encBlock, err := x509.EncryptPEMBlock(rand.Reader, "RSA PRIVATE KEY", der, []byte("secret"), x509.PEMCipherAES256)
	require.NoError(t, err)

	pk := newPrivatekey(string(pem.EncodeToMemory(encBlock)))

	algo, err := pk.publicKeyAlgorithm()
	require.NoError(t, err)
	require.Equal(t, "", algo)

	bits, err := pk.publicKeyBits()
	require.NoError(t, err)
	require.Equal(t, int64(0), bits)
}

func TestPrivatekeyGarbagePEM(t *testing.T) {
	pk := newPrivatekey("not a valid pem")

	_, err := pk.publicKeyAlgorithm()
	require.Error(t, err)

	_, err = pk.publicKeyBits()
	require.Error(t, err)
}
