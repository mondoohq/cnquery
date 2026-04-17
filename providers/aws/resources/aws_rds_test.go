// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRdsKmsKey(t *testing.T) {
	t.Run("nil key ID sets null state", func(t *testing.T) {
		db := &mqlAwsRdsDbinstance{}
		result, err := db.kmsKey()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, db.KmsKey.IsNull())
		assert.True(t, db.KmsKey.IsSet())
	})

	t.Run("empty key ID sets null state", func(t *testing.T) {
		db := &mqlAwsRdsDbinstance{}
		empty := ""
		db.cacheKmsKeyId = &empty
		result, err := db.kmsKey()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, db.KmsKey.IsNull())
		assert.True(t, db.KmsKey.IsSet())
	})
}

func TestRdsPerformanceInsightsKmsKey(t *testing.T) {
	t.Run("nil key ID sets null state", func(t *testing.T) {
		db := &mqlAwsRdsDbinstance{}
		// cachePerformanceInsightsKmsKeyId is nil by default
		result, err := db.performanceInsightsKmsKey()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, db.PerformanceInsightsKmsKey.IsNull())
		assert.True(t, db.PerformanceInsightsKmsKey.IsSet())
	})

	t.Run("empty key ID sets null state", func(t *testing.T) {
		db := &mqlAwsRdsDbinstance{}
		empty := ""
		db.cachePerformanceInsightsKmsKeyId = &empty
		result, err := db.performanceInsightsKmsKey()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, db.PerformanceInsightsKmsKey.IsNull())
		assert.True(t, db.PerformanceInsightsKmsKey.IsSet())
	})
}

func TestRdsClusterMonitoringRole(t *testing.T) {
	t.Run("empty arn sets null state", func(t *testing.T) {
		c := &mqlAwsRdsDbcluster{}
		result, err := c.monitoringRole()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, c.MonitoringRole.IsNull())
		assert.True(t, c.MonitoringRole.IsSet())
	})
}

func TestRdsClusterDbClusterParameterGroup(t *testing.T) {
	t.Run("empty parameter group name sets null state", func(t *testing.T) {
		c := &mqlAwsRdsDbcluster{}
		result, err := c.dbClusterParameterGroup()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, c.DbClusterParameterGroup.IsNull())
		assert.True(t, c.DbClusterParameterGroup.IsSet())
	})
}
