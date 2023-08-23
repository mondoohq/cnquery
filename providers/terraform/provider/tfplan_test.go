// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/terraform/connection"
	"go.mondoo.com/cnquery/providers/terraform/resources"
)

func TestResource_Tfplan(t *testing.T) {
	t.Run("tf plan changes", func(t *testing.T) {
		srv, connRes := newTestService("plan", "")
		require.NotEmpty(t, srv)
		// simulate "terraform.plan.resourceChanges[0].providerName"

		// create terraform resource
		dataResp, err := srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform.plan",
		})
		require.NoError(t, err)
		resourceId := string(dataResp.Data.Value)

		// fetch providers
		dataResp, err = srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform.plan",
			ResourceId: resourceId,
			Field:      "resourceChanges",
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(dataResp.Data.Array))

		// get provider details
		providerResourceID := string(dataResp.Data.Array[0].Value)
		dataResp, err = srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform.plan.resourceChange",
			ResourceId: providerResourceID,
			Field:      "providerName",
		})
		require.NoError(t, err)
		assert.Equal(t, "registry.terraform.io/hashicorp/google", string(dataResp.Data.Value))
	})

	t.Run("tf plan configuration", func(t *testing.T) {
		srv, connRes := newTestService("plan", "")
		require.NotEmpty(t, srv)
		// simulate "terraform.plan.configuration.resources[0]['name'] | ['type']"

		// create terraform resource
		dataResp, err := srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform.plan.configuration",
		})
		require.NoError(t, err)
		resourceId := string(dataResp.Data.Value)

		// fetch providers
		dataResp, err = srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform.plan.configuration",
			ResourceId: resourceId,
			Field:      "resources",
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(dataResp.Data.Array))

		resZero := dataResp.Data.Array[0]
		assert.NotEmpty(t, resZero)

		assert.Contains(t, string(resZero.Value), "default")
		assert.Contains(t, string(resZero.Value), "google_compute_instance")
	})
}

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
