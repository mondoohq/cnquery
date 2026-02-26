// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractCaPoolName(t *testing.T) {
	t.Run("full CA path", func(t *testing.T) {
		result := extractCaPoolName("projects/my-project/locations/us-central1/caPools/my-pool/certificateAuthorities/my-ca")
		assert.Equal(t, "my-pool", result)
	})

	t.Run("CA pool path without children", func(t *testing.T) {
		result := extractCaPoolName("projects/my-project/locations/us-central1/caPools/my-pool")
		assert.Equal(t, "my-pool", result)
	})

	t.Run("certificate path", func(t *testing.T) {
		result := extractCaPoolName("projects/my-project/locations/us-east1/caPools/prod-pool/certificates/cert-123")
		assert.Equal(t, "prod-pool", result)
	})

	t.Run("no caPools segment returns empty", func(t *testing.T) {
		result := extractCaPoolName("projects/my-project/locations/us-central1")
		assert.Equal(t, "", result)
	})

	t.Run("empty string returns empty", func(t *testing.T) {
		result := extractCaPoolName("")
		assert.Equal(t, "", result)
	})

	t.Run("caPools at end of path with no value returns empty", func(t *testing.T) {
		result := extractCaPoolName("projects/my-project/locations/us-central1/caPools")
		assert.Equal(t, "", result)
	})

	t.Run("pool name with hyphens and numbers", func(t *testing.T) {
		result := extractCaPoolName("projects/project-123/locations/europe-west1/caPools/my-pool-v2/certificateAuthorities/ca-1")
		assert.Equal(t, "my-pool-v2", result)
	})
}
