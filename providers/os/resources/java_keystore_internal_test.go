// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func selfSignedDER(t *testing.T, serial *big.Int) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "Example Root CA", OrganizationalUnit: []string{"Certification Authority"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IsCA:         true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return der
}

// A trust store built up over years holds certificates that no longer parse.
// Handing the whole batch to the certificate parser fails on the first of them
// and the store then reports nothing at all — which is how one twenty-year-old
// CA blinds every check written against a 133-entry store. This is the guard
// against that regressing.
func TestReadableDERKeepsTheStoreUsable(t *testing.T) {
	good := selfSignedDER(t, big.NewInt(42))

	t.Run("all readable", func(t *testing.T) {
		out, unreadable := readableDER([][]byte{good, good})
		assert.Len(t, out, 2)
		assert.Equal(t, 0, unreadable)
	})

	t.Run("one bad certificate does not lose the others", func(t *testing.T) {
		garbage := []byte{0x30, 0x03, 0x02, 0x01, 0x00}
		out, unreadable := readableDER([][]byte{good, garbage, good})
		assert.Len(t, out, 2, "the readable certificates survive")
		assert.Equal(t, 1, unreadable, "and the loss is counted, not silent")
	})

	// The specific shape seen in the wild, and the reason this exists at all.
	// testdata holds the real certificate: the "ec-acc" CA out of RHEL 7's
	// trust store, whose serial number is negative — legal when it was issued
	// in 2003 and rejected by the parser now. It is one entry out of 133, and
	// before this it took the other 132 down with it. Using the real
	// certificate rather than a synthesised one matters because Go will not
	// even create such a certificate today, so a synthetic version could not
	// reproduce the case.
	t.Run("a negative serial number is skipped and counted", func(t *testing.T) {
		negative, err := os.ReadFile("testdata/java-negative-serial.der")
		require.NoError(t, err)

		_, err = x509.ParseCertificate(negative)
		require.Error(t, err, "the parser rejects this, which is what makes it worth handling")
		assert.Contains(t, err.Error(), "negative serial number")

		out, unreadable := readableDER([][]byte{good, negative})
		assert.Len(t, out, 1, "the good certificate survives")
		assert.Equal(t, 1, unreadable)
	})

	t.Run("empty input", func(t *testing.T) {
		out, unreadable := readableDER(nil)
		assert.Empty(t, out)
		assert.Equal(t, 0, unreadable)
	})
}

// The discovery lists are the thing most likely to be edited carelessly later,
// and a missing path means a JVM is simply not audited — a silence, not an
// error. These pin the layouts that were verified on real images.
func TestJavaTruststoreDiscoveryCoversKnownLayouts(t *testing.T) {
	assert.Contains(t, javaTruststoreDirs, "lib/security", "JDK 9 and later")
	assert.Contains(t, javaTruststoreDirs, "jre/lib/security", "JDK 8")

	assert.Contains(t, javaHomeRoots, "/usr/lib/jvm", "the Red Hat and Debian layout")
	assert.Contains(t, javaHomeRoots, "/Library/Java/JavaVirtualMachines", "macOS")

	assert.Contains(t, javaTruststoreFiles, "/etc/pki/java/cacerts", "the Red Hat distribution store")
	assert.Contains(t, javaTruststoreFiles, "/etc/ssl/certs/java/cacerts", "the Debian distribution store")
}
