// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/types"
)

func TestResource_Registrykey(t *testing.T) {
	t.Run("non existent registry key", func(t *testing.T) {
		res := testWindowsQuery(t, "registrykey('HKEY_LOCAL_MACHINE\\Software\\Policies\\Microsoft\\Windows\\Personalization').exists")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, false, res[0].Data.Value)
	})

	t.Run("registry key path", func(t *testing.T) {
		res := testWindowsQuery(t, "registrykey('HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\System').path")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\System", res[0].Data.Value)
	})

	t.Run("existing registry key", func(t *testing.T) {
		res := testWindowsQuery(t, "registrykey('HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\System').exists")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[0].Data.Value)
	})

	t.Run("registry key properties", func(t *testing.T) {
		res := testWindowsQuery(t, "registrykey('HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\System').properties")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, 24, len(res[0].Data.Value.(map[string]interface{})))
	})

	t.Run("registry key children", func(t *testing.T) {
		res := testWindowsQuery(t, "registrykey('HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\System').children")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\System\\Audit", res[0].Data.Value.([]interface{})[0])
	})

	t.Run("non-existent registry key - props", func(t *testing.T) {
		res := testWindowsQuery(t, "registrykey('nope').properties")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, &llx.RawData{Type: types.Map(types.String, types.String)}, res[0].Data)
	})

	t.Run("non-existent registry key - items", func(t *testing.T) {
		res := testWindowsQuery(t, "registrykey('nope').items")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Nil(t, res[0].Data.Value)
	})
}
