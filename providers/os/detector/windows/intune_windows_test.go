// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package windows

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetIntuneDeviceIDFromRegistry(t *testing.T) {
	// This test runs on Windows and reads the real registry.
	// On machines without Intune enrollment, it should return an empty string without error.
	// On Intune-enrolled machines, it should return the device ID.
	id, err := getIntuneDeviceIDFromRegistry()
	require.NoError(t, err)
	assert.IsType(t, "", id)
}
