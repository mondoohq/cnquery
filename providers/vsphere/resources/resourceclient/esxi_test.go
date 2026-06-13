// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resourceclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyPersistenceUnsupportedRegex(t *testing.T) {
	// Errors from hosts that lack the keypersistence namespace (pre-7.0 Update 2
	// or no TPM) should be treated as "feature not enabled", not propagated.
	unsupported := []string{
		"Error: Not supported on this host",
		"Unknown command or namespace system security keypersistence get",
		"Invalid namespace: keypersistence",
		"This host does not have a TPM",
	}
	for _, msg := range unsupported {
		assert.True(t, keyPersistenceUnsupportedRegex.MatchString(msg), "expected match: %q", msg)
	}

	// Genuine failures (auth, connectivity) must surface as real errors.
	realErrors := []string{
		"connection refused",
		"401 Unauthorized",
		"context deadline exceeded",
	}
	for _, msg := range realErrors {
		assert.False(t, keyPersistenceUnsupportedRegex.MatchString(msg), "expected no match: %q", msg)
	}
}
