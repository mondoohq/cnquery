// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestElasticacheClusterKmsKey covers the early-return null-state paths of
// kmsKey(), which resolve before any connection or API call. A cluster without
// a replication group (the KMS key only ever comes from the replication group)
// must report a set-but-null field rather than returning a bare nil, which would
// panic the runtime. The populated path requires a live API and is covered by
// interactive testing.
func TestElasticacheClusterKmsKey(t *testing.T) {
	t.Run("nil replication group ID sets null state", func(t *testing.T) {
		c := &mqlAwsElasticacheCluster{}
		result, err := c.kmsKey()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, c.KmsKey.IsNull())
		assert.True(t, c.KmsKey.IsSet())
	})

	t.Run("empty replication group ID sets null state", func(t *testing.T) {
		c := &mqlAwsElasticacheCluster{}
		empty := ""
		c.cacheReplicationGroupId = &empty
		result, err := c.kmsKey()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, c.KmsKey.IsNull())
		assert.True(t, c.KmsKey.IsSet())
	})
}
