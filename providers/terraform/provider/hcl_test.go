// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/terraform/resources"
)

const (
	terraformHclPath       = "./testdata/terraform"
	terraformHclModulePath = "./testdata/terraform-module"
	terraformHclEmptyPath  = "./testdata/terraform-empty"
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

	t.Run("terraform datasources are not empty", func(t *testing.T) {
		srv, connRes := newTestService(HclConnectionType, terraformHclPath)
		require.NotEmpty(t, srv)

		// create terraform resource
		dataResp, err := srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform",
		})
		require.NoError(t, err)
		resourceId := string(dataResp.Data.Value)

		// fetch datasources — previously broken: appended to Providers.Data instead of Datasources.Data
		dataResp, err = srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform",
			ResourceId: resourceId,
			Field:      "datasources",
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(dataResp.Data.Array))
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
	// 	assert.Equal(t, map[string]any{"enable-oslogin": true}, res[0].Data.Value)
	// })

	t.Run("terraform settings required providers", func(t *testing.T) {
		srv, connRes := newTestService(HclConnectionType, terraformHclPath)
		require.NotEmpty(t, srv)

		// create terraform.settings resource
		dataResp, err := srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform.settings",
		})
		require.NoError(t, err)
		resourceId := string(dataResp.Data.Value)

		// fetch requiredProviders
		dataResp, err = srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform.settings",
			ResourceId: resourceId,
			Field:      "requiredProviders",
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(dataResp.Data.Array))

		// get provider details
		providerResourceID := string(dataResp.Data.Array[0].Value)
		nameResp, err := srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform.settings.requiredProvider",
			ResourceId: providerResourceID,
			Field:      "name",
		})
		require.NoError(t, err)
		assert.Equal(t, "aws", string(nameResp.Data.Value))

		sourceResp, err := srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform.settings.requiredProvider",
			ResourceId: providerResourceID,
			Field:      "source",
		})
		require.NoError(t, err)
		assert.Equal(t, "hashicorp/aws", string(sourceResp.Data.Value))

		versionResp, err := srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform.settings.requiredProvider",
			ResourceId: providerResourceID,
			Field:      "version",
		})
		require.NoError(t, err)
		assert.Equal(t, "~> 3.74", string(versionResp.Data.Value))
	})
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

// TestEmptyBlocks_NoStackOverflow verifies that terraform files with no HCL
// blocks (e.g., only comments) don't cause infinite recursion in refreshCache.
//
// Root cause: blocks() uses `var mqlHclBlocks []any` which stays nil when no
// blocks are found. refreshCache(nil) interprets nil as "not yet fetched" and
// calls GetBlocks() which calls blocks() again — infinite recursion.
func TestEmptyBlocks_NoStackOverflow(t *testing.T) {
	t.Run("terraform blocks on empty files", func(t *testing.T) {
		srv, connRes := newTestService(HclConnectionType, terraformHclEmptyPath)
		require.NotEmpty(t, srv)

		// create terraform resource
		dataResp, err := srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform",
		})
		require.NoError(t, err)
		resourceId := string(dataResp.Data.Value)

		// fetch blocks - this will stack overflow without the fix
		dataResp, err = srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform",
			ResourceId: resourceId,
			Field:      "blocks",
		})
		require.NoError(t, err)
		assert.Empty(t, dataResp.Data.Array)
	})

	t.Run("terraform providers on empty files", func(t *testing.T) {
		srv, connRes := newTestService(HclConnectionType, terraformHclEmptyPath)
		require.NotEmpty(t, srv)

		// create terraform resource
		dataResp, err := srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform",
		})
		require.NoError(t, err)
		resourceId := string(dataResp.Data.Value)

		// fetch providers via refreshCache(nil) path
		dataResp, err = srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "terraform",
			ResourceId: resourceId,
			Field:      "providers",
		})
		require.NoError(t, err)
		assert.Empty(t, dataResp.Data.Array)
	})
}

func TestKeyString(t *testing.T) {
	require.Equal(t, "keytest", resources.GetKeyString("keytest"))
	require.Equal(t, "key,thing", resources.GetKeyString([]string{"key", "thing"}))
	require.Equal(t, "keything", resources.GetKeyString([]any{"key", "thing"}))
}
