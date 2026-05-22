// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"cloud.google.com/go/functions/apiv2/functionspb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFnV2BuildConfig(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result, err := fnV2BuildConfig(nil, "parent", nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestFnV2ServiceConfig(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result, err := fnV2ServiceConfig(nil, "parent", nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestFnV2EventTrigger(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result, err := fnV2EventTrigger(nil, "parent", "", nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestFnV2EventTriggerRetryPolicy(t *testing.T) {
	t.Run("retry policy string values", func(t *testing.T) {
		assert.Equal(t, "RETRY_POLICY_DO_NOT_RETRY", functionspb.EventTrigger_RETRY_POLICY_DO_NOT_RETRY.String())
		assert.Equal(t, "RETRY_POLICY_RETRY", functionspb.EventTrigger_RETRY_POLICY_RETRY.String())
	})
}

func TestFnV2FunctionState(t *testing.T) {
	t.Run("state string values", func(t *testing.T) {
		assert.Equal(t, "ACTIVE", functionspb.Function_ACTIVE.String())
		assert.Equal(t, "FAILED", functionspb.Function_FAILED.String())
		assert.Equal(t, "DEPLOYING", functionspb.Function_DEPLOYING.String())
		assert.Equal(t, "DELETING", functionspb.Function_DELETING.String())
	})
}

func TestFnV2Environment(t *testing.T) {
	t.Run("environment string values", func(t *testing.T) {
		assert.Equal(t, "GEN_1", functionspb.Environment_GEN_1.String())
		assert.Equal(t, "GEN_2", functionspb.Environment_GEN_2.String())
	})
}
