// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTpm(t *testing.T) {
	t.Run("a present TPM 2.0", func(t *testing.T) {
		input := `{
  "TpmPresent": true,
  "TpmReady": true,
  "TpmEnabled": true,
  "TpmActivated": true,
  "ManufacturerVersion": "7.2.3.1",
  "SpecVersion": "2.0, 0, 1.59"
}`
		info, err := ParseTpm(strings.NewReader(input))
		require.NoError(t, err)
		assert.True(t, info.TpmPresent)
		assert.True(t, info.TpmReady)
		assert.True(t, info.TpmEnabled)
		assert.True(t, info.TpmActivated)
		assert.Equal(t, "7.2.3.1", info.ManufacturerVersion)
		assert.Equal(t, "2.0", info.MajorSpecVersion())
	})

	t.Run("no TPM present", func(t *testing.T) {
		input := `{
  "TpmPresent": false,
  "TpmReady": false,
  "TpmEnabled": false,
  "TpmActivated": false,
  "ManufacturerVersion": "",
  "SpecVersion": ""
}`
		info, err := ParseTpm(strings.NewReader(input))
		require.NoError(t, err)
		assert.False(t, info.TpmPresent)
		assert.Equal(t, "", info.MajorSpecVersion())
	})

	t.Run("empty output is treated as an absent TPM", func(t *testing.T) {
		info, err := ParseTpm(strings.NewReader("   \n"))
		require.NoError(t, err)
		assert.False(t, info.TpmPresent)
		assert.Equal(t, "", info.MajorSpecVersion())
	})
}

func TestMajorSpecVersion(t *testing.T) {
	assert.Equal(t, "2.0", (&TpmInfo{SpecVersion: "2.0, 0, 1.59"}).MajorSpecVersion())
	assert.Equal(t, "1.2", (&TpmInfo{SpecVersion: "1.2"}).MajorSpecVersion())
	assert.Equal(t, "", (&TpmInfo{SpecVersion: ""}).MajorSpecVersion())
}
