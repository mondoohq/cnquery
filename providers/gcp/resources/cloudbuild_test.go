// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTriggerGithubConfig(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result, err := buildTriggerGithubConfig(nil, "parent", nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestBuildTriggerPubsubConfig(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result, err := buildTriggerPubsubConfig(nil, "parent", "", nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestBuildTriggerWebhookConfig(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result, err := buildTriggerWebhookConfig(nil, "parent", nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestBuildTriggerRepoEventConfig(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result, err := buildTriggerRepoEventConfig(nil, "parent", nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestBuildWorkerPoolWorkerConfig(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result, err := buildWorkerPoolWorkerConfig(nil, "parent", nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("nil worker config within private pool returns nil", func(t *testing.T) {
		cfg := &cloudbuildpb.PrivatePoolV1Config{
			WorkerConfig: nil,
		}
		result, err := buildWorkerPoolWorkerConfig(nil, "parent", cfg)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestBuildWorkerPoolNetworkConfig(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result, err := buildWorkerPoolNetworkConfig(nil, "parent", nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("nil network config within private pool returns nil", func(t *testing.T) {
		cfg := &cloudbuildpb.PrivatePoolV1Config{
			NetworkConfig: nil,
		}
		result, err := buildWorkerPoolNetworkConfig(nil, "parent", cfg)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}
