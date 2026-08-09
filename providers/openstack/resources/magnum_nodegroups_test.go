// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCertificatePEM builds a self-signed certificate and returns it PEM
// encoded, so the parser is exercised against a real certificate rather than a
// checked-in fixture that would eventually expire.
func testCertificatePEM(t *testing.T, subject string, notAfter time.Time) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: subject},
		NotBefore:    notAfter.Add(-24 * time.Hour),
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func TestParseCertificatePEM(t *testing.T) {
	expiry := time.Date(2030, 6, 1, 12, 0, 0, 0, time.UTC)

	t.Run("single certificate", func(t *testing.T) {
		cert, err := parseCertificatePEM(testCertificatePEM(t, "kubernetes-ca", expiry))
		require.NoError(t, err)
		require.NotNil(t, cert)
		assert.Equal(t, "kubernetes-ca", cert.Subject.CommonName)
		assert.Equal(t, expiry.Unix(), cert.NotAfter.Unix())
	})

	// Magnum can hand back a bundle; the authority is the first certificate in
	// it, and a leading key block must not stop the search.
	t.Run("first certificate of a bundle", func(t *testing.T) {
		bundle := "-----BEGIN EC PRIVATE KEY-----\nZm9v\n-----END EC PRIVATE KEY-----\n" +
			testCertificatePEM(t, "kubernetes-ca", expiry) +
			testCertificatePEM(t, "intermediate", expiry.Add(24*time.Hour))
		cert, err := parseCertificatePEM(bundle)
		require.NoError(t, err)
		require.NotNil(t, cert)
		assert.Equal(t, "kubernetes-ca", cert.Subject.CommonName)
	})

	// A cluster whose authority is withheld reports nothing rather than failing
	// the whole query.
	t.Run("no certificate block", func(t *testing.T) {
		for _, raw := range []string{"", "not pem at all", "-----BEGIN EC PRIVATE KEY-----\nZm9v\n-----END EC PRIVATE KEY-----\n"} {
			cert, err := parseCertificatePEM(raw)
			require.NoError(t, err)
			assert.Nil(t, cert)
		}
	})

	// A block that claims to be a certificate but is not must be reported, not
	// silently treated as absent.
	t.Run("malformed certificate block", func(t *testing.T) {
		cert, err := parseCertificatePEM("-----BEGIN CERTIFICATE-----\nZm9v\n-----END CERTIFICATE-----\n")
		require.Error(t, err)
		assert.Nil(t, cert)
	})
}
