// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

func vaultLocationTValue(location string) *plugin.TValue[string] {
	return &plugin.TValue[string]{Data: location, State: plugin.StateIsSet}
}

func TestVaultChildLocationAndTags(t *testing.T) {
	ptr := func(s string) *string { return &s }

	t.Run("falls back to the vault region when ARM omits location", func(t *testing.T) {
		location, _ := vaultChildLocationAndTags(vaultLocationTValue("westus2"), nil, nil)
		require.Equal(t, types.String, location.Type)
		assert.Equal(t, "westus2", location.Value)
	})

	t.Run("prefers a location the API did return", func(t *testing.T) {
		location, _ := vaultChildLocationAndTags(vaultLocationTValue("westus2"), ptr("eastus"), nil)
		assert.Equal(t, "eastus", location.Value)
	})

	t.Run("an empty API location is not a location", func(t *testing.T) {
		location, _ := vaultChildLocationAndTags(vaultLocationTValue("westus2"), ptr(""), nil)
		assert.Equal(t, "westus2", location.Value)
	})

	t.Run("null location when neither the API nor the vault has one", func(t *testing.T) {
		location, _ := vaultChildLocationAndTags(vaultLocationTValue(""), nil, nil)
		assert.Same(t, llx.NilData, location)
	})

	t.Run("a vault whose location errored does not become a location", func(t *testing.T) {
		vault := &plugin.TValue[string]{Data: "westus2", State: plugin.StateIsSet, Error: errors.New("boom")}
		location, _ := vaultChildLocationAndTags(vault, nil, nil)
		assert.Same(t, llx.NilData, location)
	})

	t.Run("a null vault location does not become a location", func(t *testing.T) {
		vault := &plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
		location, _ := vaultChildLocationAndTags(vault, nil, nil)
		assert.Same(t, llx.NilData, location)
	})

	t.Run("a nil vault value is tolerated", func(t *testing.T) {
		location, _ := vaultChildLocationAndTags(nil, nil, nil)
		assert.Same(t, llx.NilData, location)
	})
}

// The regression this guards: an empty map asserts "this resource carries no
// tags". ARM never returns tags for these proxy resources, so that assertion
// would make a tag-required policy fail on every policy in the tenant and a
// tag-exempting policy exempt none of them. Null says "no tag data", which is
// what is actually true.
func TestVaultChildTagsAreNullNotEmpty(t *testing.T) {
	ptr := func(s string) *string { return &s }

	t.Run("nil tag map is null, not an empty map", func(t *testing.T) {
		_, tags := vaultChildLocationAndTags(vaultLocationTValue("westus2"), nil, nil)
		assert.Same(t, llx.NilData, tags)
	})

	t.Run("empty tag map is null, not an empty map", func(t *testing.T) {
		_, tags := vaultChildLocationAndTags(vaultLocationTValue("westus2"), nil, map[string]*string{})
		assert.Same(t, llx.NilData, tags)
	})

	t.Run("tags are reported when the API does return them", func(t *testing.T) {
		_, tags := vaultChildLocationAndTags(vaultLocationTValue("westus2"), nil, map[string]*string{
			"owner": ptr("platform"),
		})
		require.NotSame(t, llx.NilData, tags)
		assert.Equal(t, map[string]any{"owner": "platform"}, tags.Value)
	})
}
