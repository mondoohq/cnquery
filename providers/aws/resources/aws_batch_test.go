// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	batch_types "github.com/aws/aws-sdk-go-v2/service/batch/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchContainerNullState(t *testing.T) {
	t.Run("nil containerProperties sets null state", func(t *testing.T) {
		jd := &mqlAwsBatchJobDefinition{}
		// cacheContainerProperties is nil by default
		result, err := jd.container()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, jd.Container.IsNull())
		assert.True(t, jd.Container.IsSet())
	})
}

func TestBatchRetryNullState(t *testing.T) {
	t.Run("nil retryStrategy sets null state", func(t *testing.T) {
		jd := &mqlAwsBatchJobDefinition{}
		result, err := jd.retry()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, jd.Retry.IsNull())
		assert.True(t, jd.Retry.IsSet())
	})
}

func TestBatchJobTimeoutNullState(t *testing.T) {
	t.Run("nil timeout sets null state", func(t *testing.T) {
		jd := &mqlAwsBatchJobDefinition{}
		result, err := jd.jobTimeout()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, jd.JobTimeout.IsNull())
		assert.True(t, jd.JobTimeout.IsSet())
	})
}

func TestBatchContainerPropertiesDict(t *testing.T) {
	t.Run("nil cache returns nil dict", func(t *testing.T) {
		jd := &mqlAwsBatchJobDefinition{}
		result, err := jd.containerProperties()
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("non-nil cache returns dict", func(t *testing.T) {
		jd := &mqlAwsBatchJobDefinition{}
		jd.cacheContainerProperties = &batch_types.ContainerProperties{
			Image:   aws.String("alpine:latest"),
			Command: []string{"echo", "hello"},
		}
		result, err := jd.containerProperties()
		require.NoError(t, err)
		require.NotNil(t, result)
		dict, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "alpine:latest", dict["Image"])
	})
}

func TestBatchRetryStrategyDict(t *testing.T) {
	t.Run("nil cache returns nil dict", func(t *testing.T) {
		jd := &mqlAwsBatchJobDefinition{}
		result, err := jd.retryStrategy()
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("non-nil cache returns dict", func(t *testing.T) {
		jd := &mqlAwsBatchJobDefinition{}
		attempts := int32(3)
		jd.cacheRetryStrategy = &batch_types.RetryStrategy{
			Attempts: &attempts,
		}
		result, err := jd.retryStrategy()
		require.NoError(t, err)
		require.NotNil(t, result)
		dict, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(3), dict["Attempts"])
	})
}

func TestBatchTimeoutDict(t *testing.T) {
	t.Run("nil cache returns nil dict", func(t *testing.T) {
		jd := &mqlAwsBatchJobDefinition{}
		result, err := jd.timeout()
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("non-nil cache returns dict", func(t *testing.T) {
		jd := &mqlAwsBatchJobDefinition{}
		dur := int32(600)
		jd.cacheTimeout = &batch_types.JobTimeout{
			AttemptDurationSeconds: &dur,
		}
		result, err := jd.timeout()
		require.NoError(t, err)
		require.NotNil(t, result)
		dict, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(600), dict["AttemptDurationSeconds"])
	})
}
