// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vmwaretypes "github.com/vmware/govmomi/vim25/types"
)

// TestDecodeMoRef covers the inverse of ManagedObjectReference.Encode, which
// FromString is not (it splits on ":"). This decode underpins both the typed
// tagRefs accessors and the cluster.vsan lookup.
func TestDecodeMoRef(t *testing.T) {
	t.Run("plain refs with hyphenated values", func(t *testing.T) {
		ref, ok := decodeMoRef("VirtualMachine-vm-62")
		require.True(t, ok)
		assert.Equal(t, "VirtualMachine", ref.Type)
		assert.Equal(t, "vm-62", ref.Value)

		ref, ok = decodeMoRef("ClusterComputeResource-domain-c28")
		require.True(t, ok)
		assert.Equal(t, "ClusterComputeResource", ref.Type)
		assert.Equal(t, "domain-c28", ref.Value)
	})

	t.Run("round-trips Encode for representative refs", func(t *testing.T) {
		for _, in := range []vmwaretypes.ManagedObjectReference{
			{Type: "VirtualMachine", Value: "vm-62"},
			{Type: "Datastore", Value: "datastore-15"},
			{Type: "Folder", Value: "group-d1"},
			// A value that Encode URL-escapes, to prove the unescape path.
			{Type: "HostSystem", Value: "host one+two"},
		} {
			out, ok := decodeMoRef(in.Encode())
			require.True(t, ok, "decode %q", in.Encode())
			assert.Equal(t, in, out)
		}
	})

	t.Run("rejects input without a separator", func(t *testing.T) {
		_, ok := decodeMoRef("noseparator")
		assert.False(t, ok)
		_, ok = decodeMoRef("")
		assert.False(t, ok)
	})
}

// TestDecodeChainID covers both serializations the trusted-root-chains list
// endpoint uses across vCenter versions: plain strings and {"chain":"..."}.
func TestDecodeChainID(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain string", `"abc-123"`, "abc-123"},
		{"summary object", `{"chain":"xyz-9"}`, "xyz-9"},
		{"object without chain", `{"other":"x"}`, ""},
		{"unexpected number", `42`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, decodeChainID(json.RawMessage(tc.in)))
		})
	}
}

// TestParsePEMCerts verifies the certificate resolver's PEM parsing: it decodes
// every CERTIFICATE block and skips non-certificate blocks and malformed bytes.
func TestParsePEMCerts(t *testing.T) {
	certPEM, _ := selfSignedCertPEM(t, "vcenter.example.com")
	other, _ := selfSignedCertPEM(t, "root.example.com")

	t.Run("parses a single certificate", func(t *testing.T) {
		certs := parsePEMCerts(certPEM)
		require.Len(t, certs, 1)
		assert.Equal(t, "CN=vcenter.example.com", certs[0].Subject.String())
	})

	t.Run("parses a multi-cert bundle", func(t *testing.T) {
		certs := parsePEMCerts(certPEM + other)
		require.Len(t, certs, 2)
	})

	t.Run("skips non-certificate and malformed blocks", func(t *testing.T) {
		junk := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("not a key")}))
		bad := "-----BEGIN CERTIFICATE-----\nbm90IGEgY2VydA==\n-----END CERTIFICATE-----\n"
		certs := parsePEMCerts(junk + certPEM + bad)
		require.Len(t, certs, 1, "only the valid certificate should parse")
		assert.Equal(t, "CN=vcenter.example.com", certs[0].Subject.String())
	})

	t.Run("returns nil for empty input", func(t *testing.T) {
		assert.Nil(t, parsePEMCerts(""))
	})
}

// selfSignedCertPEM creates a throwaway self-signed certificate and returns it
// PEM-encoded, for exercising the PEM parsing path.
func selfSignedCertPEM(t *testing.T, cn string) (string, *x509.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Unix(0, 0),
		NotAfter:     time.Unix(1<<31, 0),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), cert
}
