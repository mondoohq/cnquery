// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGcpPlatformCatalog(t *testing.T) {
	for _, pi := range Platforms {
		assert.Equal(t, []string{"google"}, pi.Family, "family for %q", pi.Name)
		assert.Equal(t, []string{"gcp-object"}, pi.Kind, "kind for %q", pi.Name)
		assert.Equal(t, []string{"gcp"}, pi.Runtime, "runtime for %q", pi.Name)
		assert.NotEmpty(t, pi.Title, "title for %q", pi.Name)
		assert.NotEqual(t, "Google Cloud Platform", pi.Title, "generic fallback title leaked for %q", pi.Name)
	}

	// roots resolve via the catalog with the historical titles
	org := newGcpPlatform("gcp-org")
	assert.Equal(t, "gcp-org", org.Name)
	assert.Equal(t, "GCP Organization", org.Title)
	assert.Equal(t, "gcp-object", org.Kind)
	assert.Equal(t, "gcp", org.Runtime)
	assert.Equal(t, []string{"google"}, org.Family)

	// a discovery override resolves to its catalog entry
	inst := newGcpPlatform("gcp-compute-instance")
	assert.Equal(t, "GCP Compute Instance", inst.Title)
	assert.Equal(t, "gcp-object", inst.Kind)

	// an unknown name falls back to a generic GCP object
	u := newGcpPlatform("gcp-brand-new")
	assert.Equal(t, "gcp-brand-new", u.Name)
	assert.Equal(t, "Google Cloud Platform", u.Title)
	assert.Equal(t, "gcp-object", u.Kind)
	assert.Equal(t, "gcp", u.Runtime)
}
