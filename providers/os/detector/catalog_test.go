// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogPlatforms(t *testing.T) {
	cat := CatalogPlatforms()
	require.NotEmpty(t, cat)

	// one entry per detector-tree leaf, no more, no less
	assert.Len(t, cat, len(osTree))

	byName := map[string]bool{}
	for _, pi := range cat {
		byName[pi.Name] = true
		// runtime is unconstrained for OS; kinds are the closed set
		assert.Empty(t, pi.Runtime, "OS runtime must stay unconstrained for %q", pi.Name)
		assert.Equal(t, osPlatformKinds, pi.Kind, "kinds for %q", pi.Name)
	}
	for name := range osTree {
		assert.True(t, byName[name], "tree leaf %q missing from catalog", name)
	}

	// a representative platform is present with the leaf-first family chain
	// matching what the runtime emits (see resolvePlatform)
	var ubuntu *struct {
		fam []string
	}
	for _, pi := range cat {
		if pi.Name == "ubuntu" {
			ubuntu = &struct{ fam []string }{pi.Family}
		}
	}
	require.NotNil(t, ubuntu, "ubuntu should be a catalogued platform")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, ubuntu.fam)
}
