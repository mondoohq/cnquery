// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package windows

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWindowsOSBuild_Integration(t *testing.T) {
	conn := &mockLocalConnection{}
	ver, err := GetWindowsOSBuild(conn)
	require.NoError(t, err)
	require.NotNil(t, ver)

	assert.NotEmpty(t, ver.CurrentBuild, "CurrentBuild should not be empty")
	assert.NotEmpty(t, ver.ProductName, "ProductName should not be empty")
	assert.NotEmpty(t, ver.Architecture, "Architecture should not be empty")
}
