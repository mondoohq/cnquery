// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/terraform/resources"
)

const (
	terraformHclPath       = "./testdata/terraform"
	terraformHclModulePath = "./testdata/terraform-module"
)

func TestResource_Terraform(t *testing.T) {
	t.Run("terraform providers", func(t *testing.T) {
		srv, connRes := newTestService(HclConnectionType, terraformHclPath)
		require.NotEmpty(t, srv)
		// simulate "terraform.providers[0].type"

		// create terraform resource
		dataResp, err := srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform",
		})
		require.NoError(t, err)
		resourceId := string(dataResp.Data.Value)

		// fetch providers
		dataResp, err = srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform",
			ResourceId: resourceId,
			Field:      "providers",
		})
		require.NoError(t, err)
		assert.Equal(t, 3, len(dataResp.Data.Array))

		// get provider details
		providerResourceID := string(dataResp.Data.Array[0].Value)
		dataResp, err = srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform.block",
			ResourceId: providerResourceID,
			Field:      "type",
		})
		require.NoError(t, err)
		assert.Equal(t, "provider", string(dataResp.Data.Value))
	})

	t.Run("terraform ignore commented out resources", func(t *testing.T) {
		srv, connRes := newTestService(HclConnectionType, terraformHclPath)
		require.NotEmpty(t, srv)
		// simulate "terraform.providers.length"

		// create terraform resource
		dataResp, err := srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform",
		})
		require.NoError(t, err)
		resourceId := string(dataResp.Data.Value)

		// fetch providers
		dataResp, err = srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform",
			ResourceId: resourceId,
			Field:      "providers",
		})
		require.NoError(t, err)
		assert.Equal(t, 3, len(dataResp.Data.Array))

		// get provider details
		providerResourceID := string(dataResp.Data.Array[0].Value)
		dataResp, err = srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform.block",
			ResourceId: providerResourceID,
			Field:      "type",
		})
		require.NoError(t, err)
		assert.Equal(t, "provider", string(dataResp.Data.Value))
	})

	// FIXME: reimplement, when we can use MQL directly
	// t.Run("terraform nested blocks", func(t *testing.T) {
	// 	res := testTerraformHclQuery(t, terraformHclPath, "terraform.blocks.where( type == \"resource\" && labels.contains(\"aws_instance\"))[0].type")
	// 	require.NotEmpty(t, res)
	// 	assert.Empty(t, res[0].Result().Error)
	// 	assert.Equal(t, string("resource"), res[0].Data.Value)
	// })

	// t.Run("terraform jsonencode blocks", func(t *testing.T) {
	// 	res := testTerraformHclQuery(t, terraformHclPath, "terraform.resources.where( nameLabel == 'aws_iam_policy' && labels[1] == 'policy' )[0].arguments['policy'][0]['Version']")
	// 	require.NotEmpty(t, res)
	// 	assert.Empty(t, res[0].Result().Error)
	// 	assert.Equal(t, string("2012-10-17"), res[0].Data.Value)
	// })

	// t.Run("terraform providers", func(t *testing.T) {
	// 	res := testTerraformHclQuery(t, terraformHclPath, "terraform.resources.where( nameLabel  == 'google_compute_instance')[0].arguments['metadata']")
	// 	require.NotEmpty(t, res)
	// 	assert.Empty(t, res[0].Result().Error)
	// 	assert.Equal(t, map[string]interface{}{"enable-oslogin": true}, res[0].Data.Value)
	// })

	// t.Run("terraform settings", func(t *testing.T) {
	// 	res := testTerraformHclQuery(t, terraformHclPath, "terraform.settings.requiredProviders['aws']['version']")
	// 	require.NotEmpty(t, res)
	// 	assert.Empty(t, res[0].Result().Error)
	// 	assert.Equal(t, "~> 3.74", res[0].Data.Value)
	// })
}

func TestModuleWithoutResources_Terraform(t *testing.T) {
	t.Run("terraform settings", func(t *testing.T) {
		srv, connRes := newTestService(HclConnectionType, terraformHclModulePath)
		require.NotEmpty(t, srv)
		// simulate "terraform.settings"

		// fetch settings
		dataResp, err := srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform.settings",
		})
		require.NoError(t, err)
		assert.Empty(t, dataResp.Error)
	})

	t.Run("terraform settings", func(t *testing.T) {
		srv, connRes := newTestService(HclConnectionType, terraformHclModulePath)
		require.NotEmpty(t, srv)
		// simulate "terraform.settings.block"

		// create terraform resource
		dataResp, err := srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform.settings",
		})
		require.NoError(t, err)
		assert.Empty(t, dataResp.Error)

		resourceId := string(dataResp.Data.Value)

		// fetch providers
		dataResp, err = srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform.settings",
			ResourceId: resourceId,
			Field:      "block",
		})
		require.NoError(t, err)
		assert.Empty(t, dataResp.Error)
		assert.Nil(t, dataResp.Data.Value)
		assert.Empty(t, dataResp.Data.Array)
		assert.Empty(t, dataResp.Data.Map)
	})
}

func TestKeyString(t *testing.T) {
	require.Equal(t, "keytest", resources.GetKeyString("keytest"))
	require.Equal(t, "key,thing", resources.GetKeyString([]string{"key", "thing"}))
	require.Equal(t, "keything", resources.GetKeyString([]interface{}{"key", "thing"}))
}
