package terraform_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResource_Tfplan(t *testing.T) {
	t.Run("tf plan changes", func(t *testing.T) {
		res := testTerraformPlanQuery(t, "terraform.plan.resourceChanges[0].providerName")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "registry.terraform.io/hashicorp/google", res[0].Data.Value)
	})
}
