package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/types"
)

func TestResource_Processes(t *testing.T) {
	t.Run("list processes", func(t *testing.T) {
		res := x.TestQuery(t, "processes.list")
		assert.NotEmpty(t, res)
	})

	t.Run("test a specific process entry", func(t *testing.T) {
		res := x.TestQuery(t, "processes.list[0].pid")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(1), res[0].Data.Value)
	})

	t.Run("test a specific process entry with filter v1", func(t *testing.T) {
		res := x.TestQuery(t, "processes{ pid command }.list[0]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)

		m, ok := res[0].Data.Value.(map[string]interface{})
		if !ok {
			t.Error("failed to retrieve correct type of result")
			t.FailNow()
		}

		assert.Equal(t, types.Block, res[0].Data.Type)
		assert.Equal(t, llx.StringData("/sbin/init"), m["inW9aIPV3zVln3ROYYeru57EdXnE2cK452ZDPxvPs9HFaftOPsef3usY0JSS/J+EWStj+thfd7AH5XdflLF81Q=="])
		assert.Equal(t, llx.IntData(1), m["vGNOj/UnoXRncBiEGYvtT8Xml8xKuzl85lo7SkIdwF7X3tQLa/Tnv0M0UEA8pZdsQmfGkhHh3FFH3PiDFBEMwA=="])
	})
}
