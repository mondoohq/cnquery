// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"cloud.google.com/go/container/apiv1/containerpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildGKENotificationConfig(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result, err := buildGKENotificationConfig(nil, "parent", nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("nil pubsub returns nil", func(t *testing.T) {
		nc := &containerpb.NotificationConfig{Pubsub: nil}
		result, err := buildGKENotificationConfig(nil, "parent", nc)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}
