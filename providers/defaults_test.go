// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDefaultProvidersIncludesAllLocalProviders guards against the recurring
// mistake of adding a provider to ./providers without regenerating
// defaults.go (see PR #8187, which had to retroactively register bicep, helm,
// and kustomize). Every provider directory in this repository must have a
// matching entry in DefaultProviders. Extra entries (external providers that
// don't live in this repo) are fine — we only require the local ones to be
// present.
//
// Fix a failure by running: make providers/defaults
func TestDefaultProvidersIncludesAllLocalProviders(t *testing.T) {
	// The test runs with the package directory as its working directory, so
	// ./ is the providers/ directory that holds every provider subdirectory.
	entries, err := os.ReadDir(".")
	assert.NoError(t, err)

	foundProvider := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()

		// A provider directory is identified by its config/config.go. This
		// skips non-provider directories (e.g. test fixtures) without needing
		// a hardcoded allowlist.
		if _, err := os.Stat(filepath.Join(name, "config", "config.go")); err != nil {
			continue
		}
		foundProvider = true

		assert.Containsf(t, DefaultProviders, name,
			"provider %q is missing from DefaultProviders; run `make providers/defaults` to regenerate providers/defaults.go", name)
	}

	// Sanity check: make sure the directory scan actually discovered
	// providers, so a future refactor that breaks the scan doesn't silently
	// turn this into a no-op test.
	assert.True(t, foundProvider, "no provider directories were discovered; the directory scan is likely broken")
}
