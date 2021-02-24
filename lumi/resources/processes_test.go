package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/lumi"
)

func TestResource_Processes(t *testing.T) {
	t.Run("list processes", func(t *testing.T) {
		res := testQuery(t, "processes.list")
		assert.NotEmpty(t, res)
	})

	t.Run("test a specific process entry", func(t *testing.T) {
		res := testQuery(t, "processes.list[0].pid")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(1), res[0].Data.Value)
	})

	t.Run("test a specific process entry with filter", func(t *testing.T) {
		res := testQuery(t, "processes{ pid command }.list[0]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, map[string]interface{}{
			"inW9aIPV3zVln3ROYYeru57EdXnE2cK452ZDPxvPs9HFaftOPsef3usY0JSS/J+EWStj+thfd7AH5XdflLF81Q==": llx.StringData("/sbin/init"),
			"vGNOj/UnoXRncBiEGYvtT8Xml8xKuzl85lo7SkIdwF7X3tQLa/Tnv0M0UEA8pZdsQmfGkhHh3FFH3PiDFBEMwA==": llx.IntData(1),
			"_": lumi.ResourceID{Id: "1", Name: "process"},
		}, res[0].Data.Value)
	})
}
