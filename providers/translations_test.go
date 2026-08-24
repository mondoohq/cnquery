// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A provider we could not read a catalog from is reported once, naming what was
// affected. One message per field would bury the point in a scan that touches
// thousands of them.
func TestUnavailableProvidersWarning(t *testing.T) {
	assert.Empty(t, UnavailableProviders(nil).Warning())
	assert.Empty(t, UnavailableProviders{}.Warning())

	warning := UnavailableProviders{"aws", "os"}.Warning()
	assert.Contains(t, warning, "aws, os")
	assert.Contains(t, warning, "provider binary not available")
	// It has to say the content still works, or it reads as a failure.
	assert.Contains(t, warning, "content will still run")
}

// The source caches its misses: a provider that cannot be reached is asked once,
// not once per field. Without this a large policy would try to start a missing
// provider thousands of times.
func TestTranslationSourceCachesMisses(t *testing.T) {
	src := NewTranslationSource(nil)
	assert.Nil(t, src.TranslationsFor("os"))
	assert.Nil(t, src.TranslationsFor("os"))
	assert.Nil(t, src.TranslationsFor(""))
}
