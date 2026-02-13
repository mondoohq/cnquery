// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"cloud.google.com/go/redis/apiv1/redispb"
	"cloud.google.com/go/redis/cluster/apiv1/clusterpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/dayofweek"
	"google.golang.org/genproto/googleapis/type/timeofday"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRedisConvertSuspensionReasons(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := redisConvertSuspensionReasons(nil)
		assert.Empty(t, result)
	})

	t.Run("multiple reasons", func(t *testing.T) {
		reasons := []redispb.Instance_SuspensionReason{
			redispb.Instance_CUSTOMER_MANAGED_KEY_ISSUE,
			redispb.Instance_SUSPENSION_REASON_UNSPECIFIED,
		}
		result := redisConvertSuspensionReasons(reasons)
		require.Len(t, result, 2)
		assert.Equal(t, "CUSTOMER_MANAGED_KEY_ISSUE", result[0])
		assert.Equal(t, "SUSPENSION_REASON_UNSPECIFIED", result[1])
	})
}

func TestRedisConvertPersistenceConfig(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result, err := redisConvertPersistenceConfig(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("basic config", func(t *testing.T) {
		pc := &redispb.PersistenceConfig{
			PersistenceMode:   redispb.PersistenceConfig_RDB,
			RdbSnapshotPeriod: redispb.PersistenceConfig_SIX_HOURS,
		}
		result, err := redisConvertPersistenceConfig(pc)
		require.NoError(t, err)
		assert.Equal(t, "RDB", result["persistenceMode"])
		assert.Equal(t, "SIX_HOURS", result["rdbSnapshotPeriod"])
		assert.Nil(t, result["rdbNextSnapshotTime"])
		assert.Nil(t, result["rdbSnapshotStartTime"])
	})

	t.Run("with timestamps", func(t *testing.T) {
		ts := timestamppb.New(time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC))
		pc := &redispb.PersistenceConfig{
			PersistenceMode:      redispb.PersistenceConfig_RDB,
			RdbSnapshotPeriod:    redispb.PersistenceConfig_TWENTY_FOUR_HOURS,
			RdbNextSnapshotTime:  ts,
			RdbSnapshotStartTime: ts,
		}
		result, err := redisConvertPersistenceConfig(pc)
		require.NoError(t, err)
		assert.Equal(t, "2026-01-15T10:30:00Z", result["rdbNextSnapshotTime"])
		assert.Equal(t, "2026-01-15T10:30:00Z", result["rdbSnapshotStartTime"])
	})
}

func TestRedisConvertMaintenancePolicy(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result, err := redisConvertMaintenancePolicy(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("full policy", func(t *testing.T) {
		createTime := timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		updateTime := timestamppb.New(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))
		mp := &redispb.MaintenancePolicy{
			CreateTime:  createTime,
			UpdateTime:  updateTime,
			Description: "weekly maintenance",
			WeeklyMaintenanceWindow: []*redispb.WeeklyMaintenanceWindow{
				{
					Day:       dayofweek.DayOfWeek_MONDAY,
					StartTime: &timeofday.TimeOfDay{Hours: 2, Minutes: 30, Seconds: 0},
					Duration:  durationpb.New(3 * time.Hour),
				},
			},
		}
		result, err := redisConvertMaintenancePolicy(mp)
		require.NoError(t, err)
		assert.Equal(t, "2026-01-01T00:00:00Z", result["createTime"])
		assert.Equal(t, "2026-02-01T00:00:00Z", result["updateTime"])
		assert.Equal(t, "weekly maintenance", result["description"])

		windows, ok := result["weeklyMaintenanceWindow"].([]any)
		require.True(t, ok)
		require.Len(t, windows, 1)
		w := windows[0].(map[string]any)
		assert.Equal(t, "MONDAY", w["day"])
		assert.Equal(t, "02:30:00", w["startTime"])
		assert.Equal(t, "3h0m0s", w["duration"])
	})

	t.Run("skips nil windows", func(t *testing.T) {
		mp := &redispb.MaintenancePolicy{
			WeeklyMaintenanceWindow: []*redispb.WeeklyMaintenanceWindow{
				nil,
				{
					Day: dayofweek.DayOfWeek_FRIDAY,
				},
			},
		}
		result, err := redisConvertMaintenancePolicy(mp)
		require.NoError(t, err)
		windows := result["weeklyMaintenanceWindow"].([]any)
		require.Len(t, windows, 1)
		w := windows[0].(map[string]any)
		assert.Equal(t, "FRIDAY", w["day"])
	})
}

func TestRedisConvertMaintenanceSchedule(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result, err := redisConvertMaintenanceSchedule(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("full schedule", func(t *testing.T) {
		start := timestamppb.New(time.Date(2026, 3, 1, 2, 0, 0, 0, time.UTC))
		end := timestamppb.New(time.Date(2026, 3, 1, 5, 0, 0, 0, time.UTC))
		deadline := timestamppb.New(time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC))
		ms := &redispb.MaintenanceSchedule{
			StartTime:            start,
			EndTime:              end,
			ScheduleDeadlineTime: deadline,
		}
		result, err := redisConvertMaintenanceSchedule(ms)
		require.NoError(t, err)
		assert.Equal(t, "2026-03-01T02:00:00Z", result["startTime"])
		assert.Equal(t, "2026-03-01T05:00:00Z", result["endTime"])
		assert.Equal(t, "2026-03-08T00:00:00Z", result["scheduleDeadlineTime"])
	})

	t.Run("partial schedule", func(t *testing.T) {
		start := timestamppb.New(time.Date(2026, 3, 1, 2, 0, 0, 0, time.UTC))
		ms := &redispb.MaintenanceSchedule{
			StartTime: start,
		}
		result, err := redisConvertMaintenanceSchedule(ms)
		require.NoError(t, err)
		assert.Equal(t, "2026-03-01T02:00:00Z", result["startTime"])
		assert.Nil(t, result["endTime"])
		assert.Nil(t, result["scheduleDeadlineTime"])
	})
}

func TestClusterConvertPersistenceConfig(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result, err := clusterConvertPersistenceConfig(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("RDB mode with config", func(t *testing.T) {
		ts := timestamppb.New(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
		pc := &clusterpb.ClusterPersistenceConfig{
			Mode: clusterpb.ClusterPersistenceConfig_RDB,
			RdbConfig: &clusterpb.ClusterPersistenceConfig_RDBConfig{
				RdbSnapshotPeriod:    clusterpb.ClusterPersistenceConfig_RDBConfig_SIX_HOURS,
				RdbSnapshotStartTime: ts,
			},
		}
		result, err := clusterConvertPersistenceConfig(pc)
		require.NoError(t, err)
		assert.Equal(t, "RDB", result["mode"])
		rdb, ok := result["rdbConfig"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "SIX_HOURS", rdb["rdbSnapshotPeriod"])
		assert.Equal(t, "2026-06-01T00:00:00Z", rdb["rdbSnapshotStartTime"])
	})

	t.Run("AOF mode with config", func(t *testing.T) {
		pc := &clusterpb.ClusterPersistenceConfig{
			Mode: clusterpb.ClusterPersistenceConfig_AOF,
			AofConfig: &clusterpb.ClusterPersistenceConfig_AOFConfig{
				AppendFsync: clusterpb.ClusterPersistenceConfig_AOFConfig_EVERYSEC,
			},
		}
		result, err := clusterConvertPersistenceConfig(pc)
		require.NoError(t, err)
		assert.Equal(t, "AOF", result["mode"])
		aof, ok := result["aofConfig"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "EVERYSEC", aof["appendFsync"])
	})
}

func TestClusterConvertZoneDistributionConfig(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result, err := clusterConvertZoneDistributionConfig(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("multi zone", func(t *testing.T) {
		zdc := &clusterpb.ZoneDistributionConfig{
			Mode: clusterpb.ZoneDistributionConfig_MULTI_ZONE,
		}
		result, err := clusterConvertZoneDistributionConfig(zdc)
		require.NoError(t, err)
		assert.Equal(t, "MULTI_ZONE", result["mode"])
		assert.Equal(t, "", result["zone"])
	})

	t.Run("single zone", func(t *testing.T) {
		zdc := &clusterpb.ZoneDistributionConfig{
			Mode: clusterpb.ZoneDistributionConfig_SINGLE_ZONE,
			Zone: "us-central1-a",
		}
		result, err := clusterConvertZoneDistributionConfig(zdc)
		require.NoError(t, err)
		assert.Equal(t, "SINGLE_ZONE", result["mode"])
		assert.Equal(t, "us-central1-a", result["zone"])
	})
}

func TestClusterConvertMaintenancePolicy(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result, err := clusterConvertMaintenancePolicy(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("full policy", func(t *testing.T) {
		createTime := timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		mp := &clusterpb.ClusterMaintenancePolicy{
			CreateTime: createTime,
			WeeklyMaintenanceWindow: []*clusterpb.ClusterWeeklyMaintenanceWindow{
				{
					Day:       dayofweek.DayOfWeek_TUESDAY,
					StartTime: &timeofday.TimeOfDay{Hours: 4, Minutes: 0, Seconds: 0},
				},
			},
		}
		result, err := clusterConvertMaintenancePolicy(mp)
		require.NoError(t, err)
		assert.Equal(t, "2026-01-01T00:00:00Z", result["createTime"])

		windows := result["weeklyMaintenanceWindow"].([]any)
		require.Len(t, windows, 1)
		w := windows[0].(map[string]any)
		assert.Equal(t, "TUESDAY", w["day"])
		assert.Equal(t, "04:00:00", w["startTime"])
	})
}

func TestClusterConvertMaintenanceSchedule(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result, err := clusterConvertMaintenanceSchedule(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("with times", func(t *testing.T) {
		start := timestamppb.New(time.Date(2026, 4, 1, 3, 0, 0, 0, time.UTC))
		end := timestamppb.New(time.Date(2026, 4, 1, 6, 0, 0, 0, time.UTC))
		ms := &clusterpb.ClusterMaintenanceSchedule{
			StartTime: start,
			EndTime:   end,
		}
		result, err := clusterConvertMaintenanceSchedule(ms)
		require.NoError(t, err)
		assert.Equal(t, "2026-04-01T03:00:00Z", result["startTime"])
		assert.Equal(t, "2026-04-01T06:00:00Z", result["endTime"])
	})
}

func TestClusterConvertEncryptionInfo(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result, err := clusterConvertEncryptionInfo(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("CMEK encryption", func(t *testing.T) {
		ts := timestamppb.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
		ei := &clusterpb.EncryptionInfo{
			EncryptionType:     clusterpb.EncryptionInfo_CUSTOMER_MANAGED_ENCRYPTION,
			KmsKeyVersions:     []string{"projects/p/locations/l/keyRings/kr/cryptoKeys/k/cryptoKeyVersions/1"},
			KmsKeyPrimaryState: clusterpb.EncryptionInfo_ENABLED,
			LastUpdateTime:     ts,
		}
		result, err := clusterConvertEncryptionInfo(ei)
		require.NoError(t, err)
		assert.Equal(t, "CUSTOMER_MANAGED_ENCRYPTION", result["encryptionType"])
		assert.Equal(t, "ENABLED", result["kmsKeyPrimaryState"])
		assert.Equal(t, "2026-05-01T12:00:00Z", result["lastUpdateTime"])

		versions, ok := result["kmsKeyVersions"].([]any)
		require.True(t, ok)
		require.Len(t, versions, 1)
	})

	t.Run("Google default encryption", func(t *testing.T) {
		ei := &clusterpb.EncryptionInfo{
			EncryptionType:     clusterpb.EncryptionInfo_GOOGLE_DEFAULT_ENCRYPTION,
			KmsKeyPrimaryState: clusterpb.EncryptionInfo_KMS_KEY_STATE_UNSPECIFIED,
		}
		result, err := clusterConvertEncryptionInfo(ei)
		require.NoError(t, err)
		assert.Equal(t, "GOOGLE_DEFAULT_ENCRYPTION", result["encryptionType"])
		assert.Nil(t, result["lastUpdateTime"])
	})
}

func TestClusterConvertAutomatedBackupConfig(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result, err := clusterConvertAutomatedBackupConfig(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("enabled with schedule", func(t *testing.T) {
		abc := &clusterpb.AutomatedBackupConfig{
			AutomatedBackupMode: clusterpb.AutomatedBackupConfig_ENABLED,
			Retention:           durationpb.New(7 * 24 * time.Hour),
			Schedule: &clusterpb.AutomatedBackupConfig_FixedFrequencySchedule_{
				FixedFrequencySchedule: &clusterpb.AutomatedBackupConfig_FixedFrequencySchedule{
					StartTime: &timeofday.TimeOfDay{Hours: 3, Minutes: 0, Seconds: 0},
				},
			},
		}
		result, err := clusterConvertAutomatedBackupConfig(abc)
		require.NoError(t, err)
		assert.Equal(t, "ENABLED", result["automatedBackupMode"])
		assert.Equal(t, "168h0m0s", result["retention"])

		sched, ok := result["fixedFrequencySchedule"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "03:00:00", sched["startTime"])
	})

	t.Run("disabled", func(t *testing.T) {
		abc := &clusterpb.AutomatedBackupConfig{
			AutomatedBackupMode: clusterpb.AutomatedBackupConfig_DISABLED,
		}
		result, err := clusterConvertAutomatedBackupConfig(abc)
		require.NoError(t, err)
		assert.Equal(t, "DISABLED", result["automatedBackupMode"])
		assert.Nil(t, result["retention"])
		assert.Nil(t, result["fixedFrequencySchedule"])
	})
}

func TestClusterConvertCrossClusterReplicationConfig(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result, err := clusterConvertCrossClusterReplicationConfig(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("primary cluster", func(t *testing.T) {
		ts := timestamppb.New(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
		ccrc := &clusterpb.CrossClusterReplicationConfig{
			ClusterRole: clusterpb.CrossClusterReplicationConfig_PRIMARY,
			SecondaryClusters: []*clusterpb.CrossClusterReplicationConfig_RemoteCluster{
				{Cluster: "projects/p/locations/l/clusters/secondary-1", Uid: "uid-1"},
				{Cluster: "projects/p/locations/l/clusters/secondary-2", Uid: "uid-2"},
			},
			UpdateTime: ts,
		}
		result, err := clusterConvertCrossClusterReplicationConfig(ccrc)
		require.NoError(t, err)
		assert.Equal(t, "PRIMARY", result["clusterRole"])
		assert.Nil(t, result["primaryCluster"])
		assert.Equal(t, "2026-06-01T00:00:00Z", result["updateTime"])

		secondaries, ok := result["secondaryClusters"].([]any)
		require.True(t, ok)
		require.Len(t, secondaries, 2)
		sc := secondaries[0].(map[string]any)
		assert.Equal(t, "projects/p/locations/l/clusters/secondary-1", sc["cluster"])
		assert.Equal(t, "uid-1", sc["uid"])
	})

	t.Run("secondary cluster", func(t *testing.T) {
		ccrc := &clusterpb.CrossClusterReplicationConfig{
			ClusterRole: clusterpb.CrossClusterReplicationConfig_SECONDARY,
			PrimaryCluster: &clusterpb.CrossClusterReplicationConfig_RemoteCluster{
				Cluster: "projects/p/locations/l/clusters/primary",
				Uid:     "primary-uid",
			},
		}
		result, err := clusterConvertCrossClusterReplicationConfig(ccrc)
		require.NoError(t, err)
		assert.Equal(t, "SECONDARY", result["clusterRole"])

		primary, ok := result["primaryCluster"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "projects/p/locations/l/clusters/primary", primary["cluster"])
		assert.Equal(t, "primary-uid", primary["uid"])
	})

	t.Run("skips nil secondary clusters", func(t *testing.T) {
		ccrc := &clusterpb.CrossClusterReplicationConfig{
			ClusterRole: clusterpb.CrossClusterReplicationConfig_PRIMARY,
			SecondaryClusters: []*clusterpb.CrossClusterReplicationConfig_RemoteCluster{
				nil,
				{Cluster: "projects/p/locations/l/clusters/s1", Uid: "uid-1"},
			},
		}
		result, err := clusterConvertCrossClusterReplicationConfig(ccrc)
		require.NoError(t, err)
		secondaries := result["secondaryClusters"].([]any)
		require.Len(t, secondaries, 1)
	})
}
