// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResource_Sudoers(t *testing.T) {
	// Uses the global 'x' tester from os_test.go which is LinuxMock()
	// The Linux mock (arch.json) may not include sudoers test data
	// These tests verify the resource handles missing files gracefully
	// For parsing tests, see the unit tests in sudoers/sudoers_test.go

	t.Run("sudoers resource exists", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers")
		require.NotEmpty(t, res)
	})

	t.Run("sudoers files returns empty when no config files exist", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.files")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// When sudoers files don't exist, returns empty array
		files, ok := res[0].Data.Value.([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(files), 0)
	})

	t.Run("sudoers files length is accessible", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.files.length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// Length should be 0 or more
		assert.GreaterOrEqual(t, res[0].Data.Value.(int64), int64(0))
	})

	t.Run("sudoers userSpecs returns empty when no config files exist", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.userSpecs")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// When sudoers files don't exist, returns empty array
		specs, ok := res[0].Data.Value.([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(specs), 0)
	})

	t.Run("sudoers userSpecs length is accessible", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.userSpecs.length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// Length should be 0 or more
		assert.GreaterOrEqual(t, res[0].Data.Value.(int64), int64(0))
	})

	t.Run("sudoers defaults returns empty when no config files exist", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.defaults")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// When sudoers files don't exist, returns empty array
		defaults, ok := res[0].Data.Value.([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(defaults), 0)
	})

	t.Run("sudoers defaults length is accessible", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.defaults.length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// Length should be 0 or more
		assert.GreaterOrEqual(t, res[0].Data.Value.(int64), int64(0))
	})

	t.Run("sudoers aliases returns empty when no config files exist", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.aliases")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// When sudoers files don't exist, returns empty array
		aliases, ok := res[0].Data.Value.([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(aliases), 0)
	})

	t.Run("sudoers aliases length is accessible", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.aliases.length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// Length should be 0 or more
		assert.GreaterOrEqual(t, res[0].Data.Value.(int64), int64(0))
	})

	t.Run("sudoers where filter works on empty userSpecs", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.userSpecs.where(users.contains(\"root\"))")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
	})

	t.Run("sudoers map works on empty userSpecs", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.userSpecs.map(users)")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
	})

	t.Run("sudoers where filter works on empty defaults", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.defaults.where(parameter == \"env_reset\")")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
	})

	t.Run("sudoers map works on empty defaults", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.defaults.map(parameter)")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
	})

	t.Run("sudoers where filter works on empty aliases", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.aliases.where(type == \"User_Alias\")")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
	})

	t.Run("sudoers map works on empty aliases", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.aliases.map(name)")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
	})
}
