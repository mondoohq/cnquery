// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
)

func TestCustomerEncryptionKeyToDict(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		assert.Nil(t, customerEncryptionKeyToDict(nil))
	})

	t.Run("populated key maps every field", func(t *testing.T) {
		got := customerEncryptionKeyToDict(&compute.CustomerEncryptionKey{
			KmsKeyName:           "projects/p/locations/us/keyRings/r/cryptoKeys/k",
			KmsKeyServiceAccount: "sa@p.iam.gserviceaccount.com",
			RawKey:               "raw",
			RsaEncryptedKey:      "rsa",
			Sha256:               "deadbeef",
		})
		assert.Equal(t, map[string]any{
			"kmsKeyName":           "projects/p/locations/us/keyRings/r/cryptoKeys/k",
			"kmsKeyServiceAccount": "sa@p.iam.gserviceaccount.com",
			"rawKey":               "raw",
			"rsaEncryptedKey":      "rsa",
			"sha256":               "deadbeef",
		}, got)
	})

	t.Run("empty key still returns a map", func(t *testing.T) {
		// A non-nil key with all-empty fields should still produce a dict
		// (with empty-string values), so callers can distinguish "no key
		// configured" (nil) from "key configured but raw value not returned"
		// (populated dict with empty strings — common for KMS responses).
		got := customerEncryptionKeyToDict(&compute.CustomerEncryptionKey{})
		assert.NotNil(t, got)
		assert.Equal(t, "", got["kmsKeyName"])
		assert.Equal(t, "", got["sha256"])
	})
}
