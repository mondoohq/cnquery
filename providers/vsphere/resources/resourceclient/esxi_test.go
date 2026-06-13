// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resourceclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEsxcliNamespaceUnavailableRegex(t *testing.T) {
	// Errors from hosts that lack a namespace because the feature is too new
	// (keypersistence pre-7.0u2, tls pre-8.0) or the hardware lacks a TPM should
	// be treated as "not configured", not propagated.
	unsupported := []string{
		"Error: Not supported on this host",
		"Unknown command or namespace system security keypersistence get",
		"Unknown command or namespace system tls server get",
		"Invalid namespace: keypersistence",
		"This host does not have a TPM",
	}
	for _, msg := range unsupported {
		assert.True(t, esxcliNamespaceUnavailableRegex.MatchString(msg), "expected match: %q", msg)
	}

	// Genuine failures (auth, connectivity) must surface as real errors.
	realErrors := []string{
		"connection refused",
		"401 Unauthorized",
		"context deadline exceeded",
	}
	for _, msg := range realErrors {
		assert.False(t, esxcliNamespaceUnavailableRegex.MatchString(msg), "expected no match: %q", msg)
	}
}
