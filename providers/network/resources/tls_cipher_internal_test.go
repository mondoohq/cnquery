// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyCipher(t *testing.T) {
	cases := []struct {
		name string
		want cipherInfo
	}{
		{
			name: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			want: cipherInfo{
				keyExchange: "ECDHE", authentication: "RSA",
				encryption: "AES_128_GCM", mac: "SHA256",
				forwardSecrecy: true, aead: true,
			},
		},
		{
			name: "TLS_DHE_RSA_WITH_3DES_EDE_CBC_SHA",
			want: cipherInfo{
				keyExchange: "DHE", authentication: "RSA",
				encryption: "3DES_EDE_CBC", mac: "SHA",
				forwardSecrecy: true, cbc: true,
			},
		},
		{
			name: "TLS_RSA_WITH_AES_128_CBC_SHA",
			want: cipherInfo{
				keyExchange: "RSA", authentication: "RSA",
				encryption: "AES_128_CBC", mac: "SHA",
				cbc: true,
			},
		},
		{
			name: "TLS_RSA_WITH_RC4_128_SHA",
			want: cipherInfo{
				keyExchange: "RSA", authentication: "RSA",
				encryption: "RC4_128", mac: "SHA",
			},
		},
		{
			name: "TLS_DH_anon_WITH_AES_128_CBC_SHA",
			want: cipherInfo{
				keyExchange: "DH", authentication: "anon",
				encryption: "AES_128_CBC", mac: "SHA",
				anonymous: true, cbc: true,
			},
		},
		{
			name: "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256",
			want: cipherInfo{
				keyExchange: "ECDHE", authentication: "ECDSA",
				encryption: "CHACHA20_POLY1305", mac: "SHA256",
				forwardSecrecy: true, aead: true,
			},
		},
		{
			// TLS 1.3: key exchange / auth negotiated separately
			name: "TLS_AES_256_GCM_SHA384",
			want: cipherInfo{
				encryption: "AES_256_GCM", mac: "SHA384",
				forwardSecrecy: true, aead: true,
			},
		},
		{
			name: "TLS_NULL_WITH_NULL_NULL",
			want: cipherInfo{
				keyExchange: "NULL", authentication: "NULL",
				encryption: "NULL", mac: "NULL",
				null: true,
			},
		},
		{
			name: "SSL_RSA_WITH_RC4_128_MD5",
			want: cipherInfo{
				keyExchange: "RSA", authentication: "RSA",
				encryption: "RC4_128", mac: "MD5",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, classifyCipher(tc.name))
		})
	}
}

func TestClassifyCipher_ExportIsFlagged(t *testing.T) {
	// Export-grade suites are reliably flagged even though their component
	// layout is unusual (the EXPORT token sits on the key-exchange side).
	info := classifyCipher("TLS_RSA_EXPORT_WITH_RC4_40_MD5")
	assert.True(t, info.export)
	assert.Equal(t, "RSA", info.keyExchange)
	assert.False(t, info.forwardSecrecy)
}
