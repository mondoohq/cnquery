// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseWinRegistryHotpatch(t *testing.T) {

	t.Run("parse hptpatching settings correctly", func(t *testing.T) {
		data := `{
			"Name":  "Hotpatch Enrollment Package",
			"HotPatchTableSize": "4096",
			"EnableVirtualizationBasedSecurity": "1"
		}`

		m, err := ParseWinRegistryHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.True(t, m)
	})

	t.Run("parse missing table size", func(t *testing.T) {
		data := `{
			"Name":  "Hotpatch Enrollment Package",
			"HotPatchTableSize": "0",
			"EnableVirtualizationBasedSecurity": "1"
		}`

		m, err := ParseWinRegistryHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.False(t, m)
	})

	t.Run("parse missing name", func(t *testing.T) {
		data := `{
			"Name":  "",
			"HotPatchTableSize": "4096",
			"EnableVirtualizationBasedSecurity": "1"
		}`

		m, err := ParseWinRegistryHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.False(t, m)
	})

	t.Run("parse missing VBS", func(t *testing.T) {
		data := `{
			"Name":  "Hotpatch Enrollment Package",
			"HotPatchTableSize": "1",
			"EnableVirtualizationBasedSecurity": "0"
		}`

		m, err := ParseWinRegistryHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.False(t, m)
	})
}
