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

func TestRdsDbInstanceActivityStreamKmsKey(t *testing.T) {
	t.Run("nil key ID sets null state", func(t *testing.T) {
		db := &mqlAwsRdsDbinstance{}
		result, err := db.activityStreamKmsKey()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, db.ActivityStreamKmsKey.IsNull())
		assert.True(t, db.ActivityStreamKmsKey.IsSet())
	})
}

func TestRdsClusterKmsKey(t *testing.T) {
	t.Run("nil key ID sets null state", func(t *testing.T) {
		c := &mqlAwsRdsDbcluster{}
		result, err := c.kmsKey()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, c.KmsKey.IsNull())
		assert.True(t, c.KmsKey.IsSet())
	})
}

func TestRdsClusterActivityStreamKmsKey(t *testing.T) {
	t.Run("nil key ID sets null state", func(t *testing.T) {
		c := &mqlAwsRdsDbcluster{}
		result, err := c.activityStreamKmsKey()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, c.ActivityStreamKmsKey.IsNull())
		assert.True(t, c.ActivityStreamKmsKey.IsSet())
	})
}

func TestRdsSnapshotKmsKey(t *testing.T) {
	t.Run("nil key ID sets null state", func(t *testing.T) {
		s := &mqlAwsRdsSnapshot{}
		result, err := s.kmsKey()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, s.KmsKey.IsNull())
		assert.True(t, s.KmsKey.IsSet())
	})
}

func TestRdsBackupSettingKmsKey(t *testing.T) {
	t.Run("nil key ID sets null state", func(t *testing.T) {
		b := &mqlAwsRdsBackupsetting{}
		result, err := b.kmsKey()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, b.KmsKey.IsNull())
		assert.True(t, b.KmsKey.IsSet())
	})
}

func TestRdsProxyVpc(t *testing.T) {
	t.Run("nil VPC ID sets null state", func(t *testing.T) {
		p := &mqlAwsRdsProxy{}
		result, err := p.vpc()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, p.Vpc.IsNull())
		assert.True(t, p.Vpc.IsSet())
	})
}

func TestRdsProxyIamRole(t *testing.T) {
	t.Run("nil role ARN sets null state", func(t *testing.T) {
		p := &mqlAwsRdsProxy{}
		result, err := p.iamRole()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, p.IamRole.IsNull())
		assert.True(t, p.IamRole.IsSet())
	})
}

func TestRdsEventSubscriptionSnsTopic(t *testing.T) {
	t.Run("empty SNS topic ARN sets null state", func(t *testing.T) {
		e := &mqlAwsRdsEventSubscription{}
		result, err := e.snsTopic()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, e.SnsTopic.IsNull())
		assert.True(t, e.SnsTopic.IsSet())
	})
}

func TestRdsDbInstanceAssociatedRoleIamRole(t *testing.T) {
	t.Run("empty role ARN sets null state", func(t *testing.T) {
		r := &mqlAwsRdsDbinstanceAssociatedRole{}
		result, err := r.iamRole()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, r.IamRole.IsNull())
		assert.True(t, r.IamRole.IsSet())
	})
}
