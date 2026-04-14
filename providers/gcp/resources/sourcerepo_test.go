// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sourcerepo "google.golang.org/api/sourcerepo/v1"
)

func TestBuildRepoMirrorConfig(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result, err := buildRepoMirrorConfig(nil, "parent", nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("non-nil config with url", func(t *testing.T) {
		mc := &sourcerepo.MirrorConfig{
			Url:         "https://github.com/example/repo.git",
			DeployKeyId: "key-123",
			WebhookId:   "webhook-456",
		}
		assert.Equal(t, "https://github.com/example/repo.git", mc.Url)
		assert.Equal(t, "key-123", mc.DeployKeyId)
		assert.Equal(t, "webhook-456", mc.WebhookId)
	})
}
