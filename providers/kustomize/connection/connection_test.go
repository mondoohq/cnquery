// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A directory with a kustomization.yaml that fails YAML parsing used
// to be silently swallowed: loadKustomizations fell through to a
// subdir scan, returned [], and the connection saw "no kustomization
// found." Now the parse error propagates so the operator sees what's
// actually wrong.
func TestLoadKustomizations_MalformedRootPropagates(t *testing.T) {
	_, err := loadKustomizations("../testdata/malformed")
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrNoKustomization),
		"a parse error must not look like ErrNoKustomization (which only means \"no file here\")")
}

// Sanity check the sentinel: a directory with no kustomization
// filename at any of the recognized names still produces ErrNoKustomization.
func TestLoadSingleKustomization_NoFileReturnsSentinel(t *testing.T) {
	_, err := loadSingleKustomization("../testdata/empty-dir")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNoKustomization))
}

// The basic fixture still loads (regression for the sentinel refactor).
func TestLoadKustomizations_BasicFixtureStillWorks(t *testing.T) {
	entries, err := loadKustomizations("../testdata/basic")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "../testdata/basic", entries[0].Path)
}
