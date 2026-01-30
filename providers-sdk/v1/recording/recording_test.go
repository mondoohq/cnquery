// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package recording

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
)

func TestLoadRecording(t *testing.T) {
	record, err := LoadRecordingFile("testdata/recording.json")
	require.NoError(t, err)
	assert.NotNil(t, record)
}

func TestAddAndGetData(t *testing.T) {
	t.Run("adds data for existing asset", func(t *testing.T) {
		r := &recording{
			Assets: []*Asset{},
		}
		r.refreshCache()

		// Create an asset and add it to the recording
		asset := &Asset{
			Asset: &inventory.Asset{
				Id:   "test-asset-1",
				Name: "test-asset",
			},
			connections: map[string]*connection{},
			resources:   map[string]*Resource{},
		}
		r.Assets = append(r.Assets, asset)
		r.assets.Set("1", asset)

		// Add data for a new resource
		req := llx.AddDataReq{
			ConnectionID:      1,
			Resource:          "aws.ec2.instance",
			ResourceID:        "i-12345",
			RequestResourceId: "i-12345",
			Field:             "name",
			Data:              llx.StringData("test-instance"),
		}
		r.AddData(req)

		// Verify the resource was created
		resourceKey := "aws.ec2.instance\x00i-12345"
		res, exists := asset.resources[resourceKey]
		require.True(t, exists)
		assert.Equal(t, "aws.ec2.instance", res.Resource)
		assert.Equal(t, "i-12345", res.ID)
		assert.NotNil(t, res.Fields["name"])
		assert.Equal(t, "test-instance", res.Fields["name"].Value)

		// Verify GetData retrieves the field correctly
		data, ok := r.GetData(1, "aws.ec2.instance", "i-12345", "name")
		require.True(t, ok)
		assert.Equal(t, "test-instance", data.Value)

		// Verify GetData retrieves the resource id when the field is empty
		data, ok = r.GetData(1, "aws.ec2.instance", "i-12345", "")
		require.True(t, ok)
		assert.Equal(t, "i-12345", data.Value)
	})

	t.Run("adds data to existing resource", func(t *testing.T) {
		r := &recording{
			Assets: []*Asset{},
		}
		r.refreshCache()

		asset := &Asset{
			Asset: &inventory.Asset{
				Id:   "test-asset-2",
				Name: "test-asset",
			},
			connections: map[string]*connection{},
			resources:   map[string]*Resource{},
			IdsLookup:   map[string]string{},
		}
		r.Assets = append(r.Assets, asset)
		r.assets.Set("2", asset)

		// Add initial field
		req1 := llx.AddDataReq{
			ConnectionID:      2,
			Resource:          "aws.ec2.instance",
			ResourceID:        "i-67890",
			RequestResourceId: "i-67890",
			Field:             "name",
			Data:              llx.StringData("instance-1"),
		}
		r.AddData(req1)

		// Add another field to the same resource
		req2 := llx.AddDataReq{
			ConnectionID:      2,
			Resource:          "aws.ec2.instance",
			ResourceID:        "i-67890",
			RequestResourceId: "i-67890",
			Field:             "state",
			Data:              llx.StringData("running"),
		}
		r.AddData(req2)

		// Verify both fields exist
		resourceKey := "aws.ec2.instance\x00i-67890"
		res, exists := asset.resources[resourceKey]
		require.True(t, exists)
		assert.Equal(t, 2, len(res.Fields))
		assert.Equal(t, "instance-1", res.Fields["name"].Value)
		assert.Equal(t, "running", res.Fields["state"].Value)

		// Verify GetData retrieves both fields correctly
		nameData, ok := r.GetData(2, "aws.ec2.instance", "i-67890", "name")
		require.True(t, ok)
		assert.Equal(t, "instance-1", nameData.Value)

		stateData, ok := r.GetData(2, "aws.ec2.instance", "i-67890", "state")
		require.True(t, ok)
		assert.Equal(t, "running", stateData.Value)
	})

	t.Run("adds request resource id to the lookup map when the request and response resource ids differ", func(t *testing.T) {
		r := &recording{
			Assets: []*Asset{},
		}
		r.refreshCache()

		asset := &Asset{
			Asset: &inventory.Asset{
				Id:   "test-asset-3",
				Name: "test-asset",
			},
			connections: map[string]*connection{},
			resources:   map[string]*Resource{},
			IdsLookup:   map[string]string{},
		}
		r.Assets = append(r.Assets, asset)
		r.assets.Set("3", asset)

		// Add data where the request ID differs from the actual ID
		req := llx.AddDataReq{
			ConnectionID:      3,
			Resource:          "aws.ec2.instance",
			ResourceID:        "arn:aws:ec2:us-east-1:123456789012:instance/i-abc123",
			RequestResourceId: "",
			Field:             "name",
			Data:              llx.StringData("test-instance"),
		}
		r.AddData(req)

		lookupKey := "aws.ec2.instance\x00"
		actualID, exists := asset.IdsLookup[lookupKey]
		require.True(t, exists)
		assert.Equal(t, "arn:aws:ec2:us-east-1:123456789012:instance/i-abc123", actualID)

		data, ok := r.GetData(3, "aws.ec2.instance", "", "name")
		require.True(t, ok)
		assert.Equal(t, "test-instance", data.Value)
	})

	t.Run("does not add the resource id to the lookup map when both ids are equal", func(t *testing.T) {
		r := &recording{
			Assets: []*Asset{},
		}
		r.refreshCache()

		asset := &Asset{
			Asset: &inventory.Asset{
				Id:   "test-asset-4",
				Name: "test-asset",
			},
			connections: map[string]*connection{},
			resources:   map[string]*Resource{},
			IdsLookup:   map[string]string{},
		}
		r.Assets = append(r.Assets, asset)
		r.assets.Set("4", asset)

		req := llx.AddDataReq{
			ConnectionID:      4,
			Resource:          "aws.ec2.instance",
			ResourceID:        "i-xyz789",
			RequestResourceId: "i-xyz789",
			Field:             "name",
			Data:              llx.StringData("same-id-instance"),
		}
		r.AddData(req)

		// Verify IdsLookup was not updated
		_, exists := asset.IdsLookup["i-xyz789"]
		assert.False(t, exists)
		assert.Equal(t, 0, len(asset.IdsLookup))

		// Verify GetData retrieves the data using the resource ID
		data, ok := r.GetData(4, "aws.ec2.instance", "i-xyz789", "name")
		require.True(t, ok)
		assert.Equal(t, "same-id-instance", data.Value)
	})

	t.Run("ignores data when connection id not found", func(t *testing.T) {
		r := &recording{
			Assets: []*Asset{},
		}
		r.refreshCache()

		asset := &Asset{
			Asset: &inventory.Asset{
				Id:   "test-asset-5",
				Name: "test-asset",
			},
			connections: map[string]*connection{},
			resources:   map[string]*Resource{},
			IdsLookup:   map[string]string{},
		}
		r.Assets = append(r.Assets, asset)
		r.assets.Set("5", asset)

		// Add data for a non-existent connection
		req := llx.AddDataReq{
			ConnectionID:      999,
			Resource:          "aws.ec2.instance",
			ResourceID:        "i-should-not-exist",
			RequestResourceId: "i-should-not-exist",
			Field:             "name",
			Data:              llx.StringData("ghost-instance"),
		}
		r.AddData(req)

		// Verify no resource was created
		assert.Equal(t, 0, len(asset.resources))

		// Verify GetData returns false for non-existent connection
		data, ok := r.GetData(999, "aws.ec2.instance", "i-should-not-exist", "name")
		assert.False(t, ok)
		assert.Nil(t, data)
	})

	t.Run("adds data without a field", func(t *testing.T) {
		r := &recording{
			Assets: []*Asset{},
		}
		r.refreshCache()

		asset := &Asset{
			Asset: &inventory.Asset{
				Id:   "test-asset-6",
				Name: "test-asset",
			},
			connections: map[string]*connection{},
			resources:   map[string]*Resource{},
			IdsLookup:   map[string]string{},
		}
		r.Assets = append(r.Assets, asset)
		r.assets.Set("6", asset)

		req := llx.AddDataReq{
			ConnectionID:      6,
			Resource:          "aws.ec2.instance",
			ResourceID:        "i-field-test",
			RequestResourceId: "i-field-test",
			Field:             "",
			Data:              nil,
		}
		r.AddData(req)

		// Verify resource exists but has no fields
		resourceKey := "aws.ec2.instance\x00i-field-test"
		res, exists := asset.resources[resourceKey]
		require.True(t, exists)
		assert.Equal(t, "aws.ec2.instance", res.Resource)
		assert.Equal(t, "i-field-test", res.ID)
		assert.Equal(t, 0, len(res.Fields))

		// Verify GetData with empty field returns the resource id
		data, ok := r.GetData(6, "aws.ec2.instance", "i-field-test", "")
		require.True(t, ok)
		assert.Equal(t, "i-field-test", data.Value)

		// Verify GetData for "id" field returns the resource id
		data, ok = r.GetData(6, "aws.ec2.instance", "i-field-test", "id")
		require.True(t, ok)
		assert.Equal(t, "i-field-test", data.Value)
	})

	t.Run("overwrites field data when added multiple times", func(t *testing.T) {
		r := &recording{
			Assets: []*Asset{},
		}
		r.refreshCache()

		asset := &Asset{
			Asset: &inventory.Asset{
				Id:   "test-asset-7",
				Name: "test-asset",
			},
			connections: map[string]*connection{},
			resources:   map[string]*Resource{},
			IdsLookup:   map[string]string{},
		}
		r.Assets = append(r.Assets, asset)
		r.assets.Set("7", asset)

		// Add initial field value
		req1 := llx.AddDataReq{
			ConnectionID:      7,
			Resource:          "aws.ec2.instance",
			ResourceID:        "i-overwrite",
			RequestResourceId: "i-overwrite",
			Field:             "state",
			Data:              llx.StringData("pending"),
		}
		r.AddData(req1)

		// Verify initial value via GetData
		data, ok := r.GetData(7, "aws.ec2.instance", "i-overwrite", "state")
		require.True(t, ok)
		assert.Equal(t, "pending", data.Value)

		// Overwrite the same field
		req2 := llx.AddDataReq{
			ConnectionID:      7,
			Resource:          "aws.ec2.instance",
			ResourceID:        "i-overwrite",
			RequestResourceId: "i-overwrite",
			Field:             "state",
			Data:              llx.StringData("running"),
		}
		r.AddData(req2)

		// Verify the field was overwritten
		resourceKey := "aws.ec2.instance\x00i-overwrite"
		res, exists := asset.resources[resourceKey]
		require.True(t, exists)
		assert.Equal(t, 1, len(res.Fields))
		assert.Equal(t, "running", res.Fields["state"].Value)

		// Verify GetData retrieves the updated value
		data, ok = r.GetData(7, "aws.ec2.instance", "i-overwrite", "state")
		require.True(t, ok)
		assert.Equal(t, "running", data.Value)
	})

	t.Run("add multiple resources for the same asset", func(t *testing.T) {
		r := &recording{
			Assets: []*Asset{},
		}
		r.refreshCache()

		asset := &Asset{
			Asset: &inventory.Asset{
				Id:   "test-asset-8",
				Name: "test-asset",
			},
			connections: map[string]*connection{},
			resources:   map[string]*Resource{},
			IdsLookup:   map[string]string{},
		}
		r.Assets = append(r.Assets, asset)
		r.assets.Set("8", asset)

		// Add multiple resources
		req1 := llx.AddDataReq{
			ConnectionID:      8,
			Resource:          "aws.ec2.instance",
			ResourceID:        "i-multi-1",
			RequestResourceId: "i-multi-1",
			Field:             "name",
			Data:              llx.StringData("instance-1"),
		}
		r.AddData(req1)

		req2 := llx.AddDataReq{
			ConnectionID:      8,
			Resource:          "aws.ec2.instance",
			ResourceID:        "i-multi-2",
			RequestResourceId: "i-multi-2",
			Field:             "name",
			Data:              llx.StringData("instance-2"),
		}
		r.AddData(req2)

		req3 := llx.AddDataReq{
			ConnectionID:      8,
			Resource:          "aws.s3.bucket",
			ResourceID:        "bucket-1",
			RequestResourceId: "bucket-1",
			Field:             "name",
			Data:              llx.StringData("my-bucket"),
		}
		r.AddData(req3)

		// Verify all resources exist
		assert.Equal(t, 3, len(asset.resources))

		res1, exists := asset.resources["aws.ec2.instance\x00i-multi-1"]
		require.True(t, exists)
		assert.Equal(t, "instance-1", res1.Fields["name"].Value)

		res2, exists := asset.resources["aws.ec2.instance\x00i-multi-2"]
		require.True(t, exists)
		assert.Equal(t, "instance-2", res2.Fields["name"].Value)

		res3, exists := asset.resources["aws.s3.bucket\x00bucket-1"]
		require.True(t, exists)
		assert.Equal(t, "my-bucket", res3.Fields["name"].Value)

		// Verify GetData retrieves all resources correctly
		data1, ok := r.GetData(8, "aws.ec2.instance", "i-multi-1", "name")
		require.True(t, ok)
		assert.Equal(t, "instance-1", data1.Value)

		data2, ok := r.GetData(8, "aws.ec2.instance", "i-multi-2", "name")
		require.True(t, ok)
		assert.Equal(t, "instance-2", data2.Value)

		data3, ok := r.GetData(8, "aws.s3.bucket", "bucket-1", "name")
		require.True(t, ok)
		assert.Equal(t, "my-bucket", data3.Value)
	})
}
