package terraform_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	terraform_provider "go.mondoo.com/cnquery/motor/providers/terraform"
	"go.mondoo.com/cnquery/resources/packs/terraform"
)

func TestResource_Tfplan(t *testing.T) {
	t.Run("tf plan changes", func(t *testing.T) {
		res := testTerraformPlanQuery(t, "terraform.plan.resourceChanges[0].providerName")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "registry.terraform.io/hashicorp/google", res[0].Data.Value)
	})

	t.Run("tf plan configuration", func(t *testing.T) {
		res := testTerraformPlanQuery(t, "terraform.plan.configuration.resources[0]['name']")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "default", res[0].Data.Value)

		res = testTerraformPlanQuery(t, "terraform.plan.configuration.resources[0]['type']")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "google_compute_instance", res[0].Data.Value)
	})
}

func TestTerraformPlanParsing(t *testing.T) {
	data, err := os.ReadFile("./testdata/tfplan-configuration/tfplan.json")
	require.NoError(t, err)

	var plan terraform_provider.Plan
	err = json.Unmarshal(data, &plan)

	pc := terraform.PlanConfiguration{}

	err = json.Unmarshal(plan.Configuration, &pc)
	require.NoError(t, err)

	assert.Equal(t, 1, len(pc.RootModule.Resources))
}
