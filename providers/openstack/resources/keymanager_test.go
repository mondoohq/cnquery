// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/keymanager/v1/containers"
	"github.com/stretchr/testify/assert"
)

func TestBarbicanRefID(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want string
	}{
		{"empty", "", ""},
		{"full url", "https://barbican.example/v1/secrets/abc-123", "abc-123"},
		{"trailing id only", "abc-123", "abc-123"},
		{"trailing slash returns empty", "https://example/v1/secrets/", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, barbicanRefID(tt.ref))
		})
	}
}

func TestNamedSecretRef(t *testing.T) {
	refs := []containers.SecretRef{
		{Name: "certificate", SecretRef: "https://example/secrets/cert-uuid"},
		{Name: "private_key", SecretRef: "https://example/secrets/key-uuid"},
		{Name: "intermediates", SecretRef: "https://example/secrets/inter-uuid"},
	}
	t.Run("found by name", func(t *testing.T) {
		assert.Equal(t, "https://example/secrets/key-uuid", namedSecretRef(refs, "private_key"))
	})
	t.Run("missing returns empty", func(t *testing.T) {
		assert.Equal(t, "", namedSecretRef(refs, "passphrase"))
	})
	t.Run("empty refs returns empty", func(t *testing.T) {
		assert.Equal(t, "", namedSecretRef(nil, "any"))
	})
}
