package terraform_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResource_Tfstate(t *testing.T) {
	t.Run("tf state outputs", func(t *testing.T) {
		res := testTerraformStateQuery(t, "terraform.state.outputs.length")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(0), res[0].Data.Value)
	})

	t.Run("tf state recursive modules", func(t *testing.T) {
		res := testTerraformStateQuery(t, "terraform.state.modules.length")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(1), res[0].Data.Value)
	})

	t.Run("tf state direct init", func(t *testing.T) {
		// NOTE tfstate root modules have no name
		res := testTerraformStateQuery(t, `terraform.state.module("").resources[0].address`)
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "aws_instance.app_server", res[0].Data.Value)
	})
}
