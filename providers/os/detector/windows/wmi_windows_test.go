// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package windows

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToString(t *testing.T) {
	t.Run("nil returns empty string", func(t *testing.T) {
		assert.Equal(t, "", toString(nil))
	})

	t.Run("non-nil returns value", func(t *testing.T) {
		s := "hello"
		assert.Equal(t, "hello", toString(&s))
	})

	t.Run("empty string returns empty string", func(t *testing.T) {
		s := ""
		assert.Equal(t, "", toString(&s))
	})
}

func TestIntToString(t *testing.T) {
	t.Run("nil returns empty string", func(t *testing.T) {
		assert.Equal(t, "", intToString(nil))
	})

	t.Run("non-nil returns string representation", func(t *testing.T) {
		i := 42
		assert.Equal(t, "42", intToString(&i))
	})

	t.Run("zero returns 0", func(t *testing.T) {
		i := 0
		assert.Equal(t, "0", intToString(&i))
	})
}

func TestGetWmiInformation_Integration(t *testing.T) {
	// Reuse the same mock connection type from build_version_windows_test.go
	conn := &mockLocalConnection{}
	info, err := GetWmiInformation(conn)
	require.NoError(t, err)
	require.NotNil(t, info)

	assert.NotEmpty(t, info.Version, "Version should not be empty")
	assert.NotEmpty(t, info.BuildNumber, "BuildNumber should not be empty")
	assert.NotEmpty(t, info.Caption, "Caption should not be empty")
}
