// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/clearsign"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// signProvenance builds a Helm-style .prov: a PGP clear-signed message whose
// body is the chart metadata followed by a "files:" block. It returns the
// armored bytes and the signing key's ID so tests can assert against both.
func signProvenance(t *testing.T, body string) ([]byte, uint64) {
	t.Helper()
	entity, err := openpgp.NewEntity("Test User", "helm provenance test", "test@example.com", nil)
	require.NoError(t, err)

	var buf bytes.Buffer
	w, err := clearsign.Encode(&buf, entity.PrivateKey, nil)
	require.NoError(t, err)
	_, err = w.Write([]byte(body))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	return buf.Bytes(), entity.PrimaryKey.KeyId
}

func provBody(archiveSHA256 string) string {
	return "apiVersion: v2\nname: mychart\nversion: 1.2.3\n...\n" +
		"files:\n  mychart-1.2.3.tgz: sha256:" + archiveSHA256 + "\n"
}

func TestParseProvenance(t *testing.T) {
	archive := []byte("fake helm chart archive bytes")
	sum := sha256.Sum256(archive)
	archiveHex := hex.EncodeToString(sum[:])

	provData, keyID := signProvenance(t, provBody(archiveHex))

	t.Run("matching archive", func(t *testing.T) {
		info := parseProvenance(provData, archiveHex)
		assert.True(t, info.signed)
		assert.Equal(t, "sha256:"+archiveHex, info.digest)
		assert.True(t, info.digestMatches)
		assert.Equal(t, fmt.Sprintf("%016X", keyID), info.keyId)
	})

	t.Run("mismatched archive flags tampering", func(t *testing.T) {
		info := parseProvenance(provData, "0123456789abcdef")
		assert.True(t, info.signed)
		assert.False(t, info.digestMatches)
		assert.Equal(t, "sha256:"+archiveHex, info.digest)
	})

	t.Run("unknown archive hash skips comparison", func(t *testing.T) {
		info := parseProvenance(provData, "")
		assert.True(t, info.signed)
		assert.False(t, info.digestMatches)
		assert.Equal(t, "sha256:"+archiveHex, info.digest)
	})

	t.Run("not a provenance file", func(t *testing.T) {
		info := parseProvenance([]byte("this is not a pgp message"), archiveHex)
		assert.False(t, info.signed)
		assert.Empty(t, info.digest)
		assert.Empty(t, info.keyId)
		assert.False(t, info.digestMatches)
	})
}

func TestProvenanceDigest(t *testing.T) {
	t.Run("prefers the .tgz entry", func(t *testing.T) {
		plaintext := "name: c\n...\nfiles:\n  c-1.0.0.tgz: sha256:aaa\n  values.schema.json: sha256:bbb\n"
		assert.Equal(t, "sha256:aaa", provenanceDigest([]byte(plaintext)))
	})

	t.Run("falls back to first file when no tgz", func(t *testing.T) {
		plaintext := "name: c\n...\nfiles:\n  only.json: sha256:ccc\n"
		assert.Equal(t, "sha256:ccc", provenanceDigest([]byte(plaintext)))
	})

	t.Run("no files block", func(t *testing.T) {
		assert.Empty(t, provenanceDigest([]byte("name: c\nversion: 1.0.0\n")))
	})
}
