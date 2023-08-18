// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/providers/terraform/connection"
	"go.mondoo.com/cnquery/providers/terraform/resources"
)

// FIXME: needs recordings???
// This needs terraform_test.go fixed with recordings

// func TestResource_Tfplan(t *testing.T) {
// 	t.Run("tf plan changes", func(t *testing.T) {
// 		res := testTerraformPlanQuery(t, "terraform.plan.resourceChanges[0].providerName")
// 		require.NotEmpty(t, res)
// 		assert.Empty(t, res[0].Result().Error)
// 		assert.Equal(t, "registry.terraform.io/hashicorp/google", res[0].Data.Value)
// 	})

// 	t.Run("tf plan configuration", func(t *testing.T) {
// 		res := testTerraformPlanQuery(t, "terraform.plan.configuration.resources[0]['name']")
// 		require.NotEmpty(t, res)
// 		assert.Empty(t, res[0].Result().Error)
// 		assert.Equal(t, "default", res[0].Data.Value)

// 		res = testTerraformPlanQuery(t, "terraform.plan.configuration.resources[0]['type']")
// 		require.NotEmpty(t, res)
// 		assert.Empty(t, res[0].Result().Error)
// 		assert.Equal(t, "google_compute_instance", res[0].Data.Value)
// 	})
// }

func TestTerraformPlanParsing(t *testing.T) {
	data, err := os.ReadFile("./testdata/tfplan-configuration/tfplan.json")
	require.NoError(t, err)

	var tfPlan connection.Plan
	err = json.Unmarshal(data, &tfPlan)

	pc := resources.PlanConfiguration{}

	err = json.Unmarshal(tfPlan.Configuration, &pc)
	require.NoError(t, err)

	assert.Equal(t, 1, len(pc.RootModule.Resources))
}
