// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestBoolValueToPtr(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result := boolValueToPtr(nil)
		assert.Nil(t, result)
	})

	t.Run("true value", func(t *testing.T) {
		result := boolValueToPtr(wrapperspb.Bool(true))
		require.NotNil(t, result)
		assert.True(t, *result)
	})

	t.Run("false value", func(t *testing.T) {
		result := boolValueToPtr(wrapperspb.Bool(false))
		require.NotNil(t, result)
		assert.False(t, *result)
	})
}

func TestRegionNameFromRegionUrl(t *testing.T) {
	assert.Equal(t, "us-central1", RegionNameFromRegionUrl("https://www.googleapis.com/compute/v1/projects/my-project/regions/us-central1"))
	assert.Equal(t, "europe-west1", RegionNameFromRegionUrl("europe-west1"))
	assert.Equal(t, "", RegionNameFromRegionUrl(""))
}
