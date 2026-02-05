// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResource_Sudoers(t *testing.T) {
	// Uses the global 'x' tester from os_test.go (LinuxMock with arch.json)
	// For parsing unit tests, see sudoers/sudoers_test.go

	t.Run("files are discovered", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.files.length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		assert.Equal(t, int64(1), res[0].Data.Value)
	})

	t.Run("userSpecs parsing", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.userSpecs.length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		assert.Equal(t, int64(6), res[0].Data.Value)
	})

	t.Run("userSpec fields", func(t *testing.T) {
		// Test root user spec has all expected fields populated
		res := x.TestQuery(t, "sudoers.userSpecs.where(users.contains(\"root\")).first")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)

		// Verify hosts field
		res = x.TestQuery(t, "sudoers.userSpecs.where(users.contains(\"root\")).first.hosts")
		require.NotEmpty(t, res)
		hosts, ok := res[0].Data.Value.([]any)
		require.True(t, ok)
		assert.Contains(t, hosts, "ALL")

		// Verify commands field
		res = x.TestQuery(t, "sudoers.userSpecs.where(users.contains(\"root\")).first.commands")
		require.NotEmpty(t, res)
		commands, ok := res[0].Data.Value.([]any)
		require.True(t, ok)
		assert.Contains(t, commands, "ALL")
	})

	t.Run("userSpec with NOPASSWD tag", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.userSpecs.where(tags.contains(\"NOPASSWD\")).length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		assert.Equal(t, int64(2), res[0].Data.Value)
	})

	t.Run("defaults parsing", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.defaults.length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		assert.Equal(t, int64(6), res[0].Data.Value)
	})

	t.Run("defaults fields", func(t *testing.T) {
		// Test secure_path has correct value
		res := x.TestQuery(t, "sudoers.defaults.where(parameter == \"secure_path\").first.value")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		assert.Equal(t, "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin", res[0].Data.Value)

		// Test negated flag (!lecture)
		res = x.TestQuery(t, "sudoers.defaults.where(parameter == \"lecture\").first.negated")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		assert.Equal(t, true, res[0].Data.Value)
	})

	t.Run("defaults scoped entries", func(t *testing.T) {
		// Test host-scoped default
		res := x.TestQuery(t, "sudoers.defaults.where(scope == \"host\").first.target")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		assert.Equal(t, "webservers", res[0].Data.Value)
	})

	t.Run("aliases parsing", func(t *testing.T) {
		res := x.TestQuery(t, "sudoers.aliases.length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		assert.Equal(t, int64(7), res[0].Data.Value)
	})

	t.Run("alias fields", func(t *testing.T) {
		// Note: alias type is stored as lowercase without "_Alias" suffix
		res := x.TestQuery(t, "sudoers.aliases.where(type == \"user\" && name == \"ADMINS\").first.members")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		members, ok := res[0].Data.Value.([]any)
		require.True(t, ok)
		assert.Equal(t, 3, len(members))
		assert.Contains(t, members, "alice")
	})

	t.Run("all alias types present", func(t *testing.T) {
		// Verify all 4 alias types are parsed
		res := x.TestQuery(t, "sudoers.aliases.where(type == \"host\").length")
		require.NotEmpty(t, res)
		assert.Equal(t, int64(2), res[0].Data.Value)

		res = x.TestQuery(t, "sudoers.aliases.where(type == \"cmnd\").length")
		require.NotEmpty(t, res)
		assert.Equal(t, int64(2), res[0].Data.Value)

		res = x.TestQuery(t, "sudoers.aliases.where(type == \"runas\").length")
		require.NotEmpty(t, res)
		assert.Equal(t, int64(1), res[0].Data.Value)
	})

	t.Run("metadata fields", func(t *testing.T) {
		// Test file and lineNumber are populated
		res := x.TestQuery(t, "sudoers.userSpecs.first.file")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		assert.Equal(t, "/etc/sudoers", res[0].Data.Value)

		res = x.TestQuery(t, "sudoers.userSpecs.first.lineNumber")
		require.NotEmpty(t, res)
		lineNum, ok := res[0].Data.Value.(int64)
		require.True(t, ok)
		assert.Greater(t, lineNum, int64(0))
	})
}
