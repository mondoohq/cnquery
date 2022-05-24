package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	t.Run("existent registry key", func(t *testing.T) {
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
}
