package gcp

import (
	"context"
	"fmt"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/sqladmin/v1"
)

func (g *mqlGcpProjectSqlService) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/gcp.project.sqlService", projectId), nil
}

func (g *mqlGcpProject) GetSql() (interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.sqlService",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectSqlService) GetInstances() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, sqladmin.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	sqladminSvc, err := sqladmin.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	sqlinstances, err := sqladminSvc.Instances.List(projectId).Do()
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, 0, len(sqlinstances.Items))
	for i := range sqlinstances.Items {
		instance := sqlinstances.Items[i]
		instanceId := fmt.Sprintf("%s/%s", projectId, instance.Name)

		var mqlEncCfg resources.ResourceType
		if instance.DiskEncryptionConfiguration != nil {
			mqlEncCfg, err = g.MotorRuntime.CreateResource("gcp.project.sqlService.instance.diskEncryptionConfiguration",
				"id", fmt.Sprintf("%s/diskEncryptionConfiguration", instanceId),
				"kmsKeyName", instance.DiskEncryptionConfiguration.KmsKeyName,
			)
			if err != nil {
				return nil, err
			}
		}

		var mqlEncStatus resources.ResourceType
		if instance.DiskEncryptionStatus != nil {
			mqlEncStatus, err = g.MotorRuntime.CreateResource("gcp.project.sqlService.instance.diskEncryptionConfiguration",
				"id", fmt.Sprintf("%s/diskEncryptionStatus", instanceId),
				"kmsKeyVersionName", instance.DiskEncryptionStatus.KmsKeyVersionName,
			)
			if err != nil {
				return nil, err
			}
		}

		var mqlFailoverReplica resources.ResourceType
		if instance.FailoverReplica != nil {
			mqlEncStatus, err = g.MotorRuntime.CreateResource("gcp.project.sqlService.instance.failoverReplica",
				"id", fmt.Sprintf("%s/failoverReplica", instanceId),
				"available", instance.FailoverReplica.Available,
				"name", instance.FailoverReplica.Name,
			)
			if err != nil {
				return nil, err
			}
		}

		mqlIpAddresses := make([]interface{}, 0, len(instance.IpAddresses))
		for i, a := range instance.IpAddresses {
			mqlIpAddress, err := g.MotorRuntime.CreateResource("gcp.project.sqlService.instance.ipMapping",
				"id", fmt.Sprintf("%s/ipAddresses%d", instanceId, i),
				"ipAddress", a.IpAddress,
				"timeToRetire", parseTime(a.TimeToRetire),
				"type", a.Type,
			)
			if err != nil {
				return nil, err
			}
			mqlIpAddresses = append(mqlIpAddresses, mqlIpAddress)
		}

		s := instance.Settings
		dbFlags := make(map[string]string)
		for _, f := range s.DatabaseFlags {
			dbFlags[f.Name] = f.Value
		}

		var mqlADCfg resources.ResourceType
		if s.ActiveDirectoryConfig != nil {
			mqlADCfg, err = g.MotorRuntime.CreateResource("gcp.project.sqlService.instance.settings.activedirectoryconfig",
				"id", fmt.Sprintf("%s/settings/activeDirectoryConfig", instanceId),
				"domain", s.ActiveDirectoryConfig.Domain,
			)
			if err != nil {
				return nil, err
			}
		}

		var mqlBackupCfg resources.ResourceType
		if s.BackupConfiguration != nil {
			mqlRetentionSettings, err := g.MotorRuntime.CreateResource("gcp.project.sqlService.instance.settings.backupconfiguration.retentionsettings",
				"id", fmt.Sprintf("%s/settings/backupConfiguration/retentionSettings", instanceId),
				"retainedBackups", s.BackupConfiguration.BackupRetentionSettings.RetainedBackups,
				"retentionUnit", s.BackupConfiguration.BackupRetentionSettings.RetentionUnit,
			)
			if err != nil {
				return nil, err
			}

			mqlBackupCfg, err = g.MotorRuntime.CreateResource("gcp.project.sqlService.instance.settings.backupconfiguration",
				"id", fmt.Sprintf("%s/settings/backupConfiguration", instanceId),
				"backupRetentionSettings", mqlRetentionSettings,
				"binaryLogEnabled", s.BackupConfiguration.BinaryLogEnabled,
				"enabled", s.BackupConfiguration.Enabled,
				"location", s.BackupConfiguration.Location,
				"pointInTimeRecoveryEnabled", s.BackupConfiguration.PointInTimeRecoveryEnabled,
				"startTime", s.BackupConfiguration.StartTime,
				"transactionLogRetentionDays", s.BackupConfiguration.TransactionLogRetentionDays,
			)
			if err != nil {
				return nil, err
			}
		}

		mqlDenyMaintenancePeriods := make([]interface{}, 0, len(s.DenyMaintenancePeriods))
		for i, p := range s.DenyMaintenancePeriods {
			mqlPeriod, err := g.MotorRuntime.CreateResource("gcp.project.sqlService.instance.settings.denyMaintenancePeriod",
				"id", fmt.Sprintf("%s/settings/denyMaintenancePeriod%d", instanceId, i),
				"endDate", p.EndDate,
				"startDate", p.StartDate,
				"time", p.Time,
			)
			if err != nil {
				return nil, err
			}
			mqlDenyMaintenancePeriods = append(mqlDenyMaintenancePeriods, mqlPeriod)
		}

		var mqlInsightsConfig resources.ResourceType
		if s.InsightsConfig != nil {
			mqlInsightsConfig, err = g.MotorRuntime.CreateResource("gcp.project.sqlService.instance.settings.insightsConfig",
				"id", fmt.Sprintf("%s/settings/insightsConfig", instanceId),
				"queryInsightsEnabled", s.InsightsConfig.QueryInsightsEnabled,
				"queryPlansPerMinute", s.InsightsConfig.QueryPlansPerMinute,
				"queryStringLength", s.InsightsConfig.QueryStringLength,
				"recordApplicationTags", s.InsightsConfig.RecordApplicationTags,
				"recordClientAddress", s.InsightsConfig.RecordClientAddress,
			)
			if err != nil {
				return nil, err
			}
		}

		var mqlIpCfg resources.ResourceType
		if s.IpConfiguration != nil {
			mqlAclEntries := make([]interface{}, 0, len(s.IpConfiguration.AuthorizedNetworks))
			for i, e := range s.IpConfiguration.AuthorizedNetworks {
				mqlAclEntry, err := g.MotorRuntime.CreateResource("gcp.project.sqlService.instance.settings.ipConfiguration.authorizedNetworks",
					"id", fmt.Sprintf("%s/settings/ipConfiguration/authorizedNetworks%d", instanceId, i),
					"expirationTime", parseTime(e.ExpirationTime),
					"name", e.Name,
					"value", e.Value,
				)
				if err != nil {
					return nil, err
				}
				mqlAclEntries = append(mqlAclEntries, mqlAclEntry)
			}

			mqlIpCfg, err = g.MotorRuntime.CreateResource("gcp.project.sqlService.instance.settings.ipConfiguration",
				"id", fmt.Sprintf("%s/settings/ipConfiguration", instanceId),
				"allocatedIpRange", s.IpConfiguration.AllocatedIpRange,
				"authorizedNetworks", mqlAclEntries,
				"ipv4Enabled", s.IpConfiguration.Ipv4Enabled,
				"privateNetwork", s.IpConfiguration.PrivateNetwork,
				"requireSsl", s.IpConfiguration.RequireSsl,
			)
			if err != nil {
				return nil, err
			}
		}

		var mqlLocationPref resources.ResourceType
		if s.LocationPreference != nil {
			mqlLocationPref, err = g.MotorRuntime.CreateResource("gcp.project.sqlService.instance.settings.locationPreference",
				"id", fmt.Sprintf("%s/settings/locationPreference", instanceId),
				"followGaeApplication", s.LocationPreference.FollowGaeApplication,
				"secondaryZone", s.LocationPreference.SecondaryZone,
				"zone", s.LocationPreference.Zone,
			)
			if err != nil {
				return nil, err
			}
		}

		var mqlMaintenanceWindow resources.ResourceType
		if s.MaintenanceWindow != nil {
			mqlMaintenanceWindow, err = g.MotorRuntime.CreateResource("gcp.project.sqlService.instance.settings.maintenanceWindow",
				"id", fmt.Sprintf("%s/settings/maintenanceWindow", instanceId),
				"day", s.MaintenanceWindow.Day,
				"hour", s.MaintenanceWindow.Hour,
				"updateTrack", s.MaintenanceWindow.UpdateTrack,
			)
			if err != nil {
				return nil, err
			}
		}

		var mqlPwdValidationPolicy resources.ResourceType
		if s.PasswordValidationPolicy != nil {
			mqlPwdValidationPolicy, err = g.MotorRuntime.CreateResource("gcp.project.sqlService.instance.settings.passwordValidationPolicy",
				"id", fmt.Sprintf("%s/settings/passwordValidationPolicy", instanceId),
				"complexity", s.PasswordValidationPolicy.Complexity,
				"disallowUsernameSubstring", s.PasswordValidationPolicy.DisallowUsernameSubstring,
				"enabledPasswordPolicy", s.PasswordValidationPolicy.EnablePasswordPolicy,
				"minLength", s.PasswordValidationPolicy.MinLength,
				"passwordChangeInterval", s.PasswordValidationPolicy.PasswordChangeInterval,
				"reuseInterval", s.PasswordValidationPolicy.ReuseInterval,
			)
			if err != nil {
				return nil, err
			}
		}

		var mqlSqlServerAuditCfg resources.ResourceType
		if s.SqlServerAuditConfig != nil {
			mqlSqlServerAuditCfg, err = g.MotorRuntime.CreateResource("gcp.project.sqlService.instance.settings.sqlServerAuditConfig",
				"id", fmt.Sprintf("%s/settings/sqlSertverAuditConfig", instanceId),
				"bucket", s.SqlServerAuditConfig.Bucket,
				"retentionInterval", s.SqlServerAuditConfig.RetentionInterval,
				"uploadInterval", s.SqlServerAuditConfig.UploadInterval,
			)
			if err != nil {
				return nil, err
			}
		}

		mqlSettings, err := g.MotorRuntime.CreateResource("gcp.project.sqlService.instance.settings",
			"projectId", projectId,
			"instanceName", instance.Name,
			"activationPolicy", s.ActivationPolicy,
			"activeDirectoryConfig", mqlADCfg,
			"availabilityType", s.AvailabilityType,
			"backupConfiguration", mqlBackupCfg,
			"collation", s.Collation,
			"connectorEnforcement", s.ConnectorEnforcement,
			"crashSafeReplicationEnabled", s.CrashSafeReplicationEnabled,
			"dataDiskSizeGb", s.DataDiskSizeGb,
			"dataDiskType", s.DataDiskType,
			"databaseFlags", core.StrMapToInterface(dbFlags),
			"databaseReplicationEnabled", s.DatabaseReplicationEnabled,
			"deletionProtectionEnabled", s.DeletionProtectionEnabled,
			"denyMaintenancePeriods", mqlDenyMaintenancePeriods,
			"insightsConfig", mqlInsightsConfig,
			"ipConfiguration", mqlIpCfg,
			"locationPreference", mqlLocationPref,
			"maintenanceWindow", mqlMaintenanceWindow,
			"passwordValidationPolicy", mqlPwdValidationPolicy,
			"pricingPlan", s.PricingPlan,
			"replicationType", s.ReplicationType,
			"settingsVersion", s.SettingsVersion,
			"sqlServerAuditConfig", mqlSqlServerAuditCfg,
			"storageAutoResize", *s.StorageAutoResize,
			"storageAutoResizeLimit", s.StorageAutoResizeLimit,
			"tier", s.Tier,
			"timeZone", s.TimeZone,
			"userLabels", core.StrMapToInterface(s.UserLabels),
		)
		if err != nil {
			return nil, err
		}

		mqlInstance, err := g.MotorRuntime.CreateResource("gcp.project.sqlService.instance",
			"projectId", projectId,
			"availableMaintenanceVersions", core.StrSliceToInterface(instance.AvailableMaintenanceVersions),
			"backendType", instance.BackendType,
			"connectionName", instance.ConnectionName,
			"created", parseTime(instance.CreateTime),
			"currentDiskSize", instance.CurrentDiskSize,
			"databaseInstalledVersion", instance.DatabaseInstalledVersion,
			"databaseVersion", instance.DatabaseVersion,
			"diskEncryptionConfiguration", mqlEncCfg,
			"diskEncryptionStatus", mqlEncStatus,
			"failoverReplica", mqlFailoverReplica,
			"gceZone", instance.GceZone,
			"instanceType", instance.InstanceType,
			"ipAddresses", mqlIpAddresses,
			"maintenanceVersion", instance.MaintenanceVersion,
			"masterInstanceName", instance.MasterInstanceName,
			"maxDiskSize", instance.MaxDiskSize,
			"name", instance.Name,
			// ref project
			"project", instance.Project,
			"region", instance.Region,
			"replicaNames", core.StrSliceToInterface(instance.ReplicaNames),
			"settings", mqlSettings,
			"serviceAccountEmailAddress", instance.ServiceAccountEmailAddress,
			"state", instance.State,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (g *mqlGcpProjectSqlServiceInstance) GetDatabases() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	instanceName, err := g.Name()
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, sqladmin.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	sqladminSvc, err := sqladmin.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	dbs, err := sqladminSvc.Databases.List(projectId, instanceName).Do()
	if err != nil {
		return nil, err
	}

	mqlDbs := make([]interface{}, 0, len(dbs.Items))
	for _, db := range dbs.Items {
		dbId := fmt.Sprintf("%s/%s/%s", projectId, instanceName, db.Name)
		var sqlServerDbDetails resources.ResourceType
		if db.SqlserverDatabaseDetails != nil {
			sqlServerDbDetails, err = g.MotorRuntime.CreateResource("gcp.project.sqlService.instance.database.sqlserverDatabaseDetails",
				"id", fmt.Sprintf("%s/sqlserverDatabaseDetails", dbId),
				"compatibilityLevel", db.SqlserverDatabaseDetails.CompatibilityLevel,
				"recoveryModel", db.SqlserverDatabaseDetails.RecoveryModel,
			)
			if err != nil {
				return nil, err
			}
		}

		mqlDb, err := g.MotorRuntime.CreateResource("gcp.project.sqlService.instance.database",
			"projectId", projectId,
			"charset", db.Charset,
			"collation", db.Collation,
			"instance", instanceName,
			"name", db.Name,
			"project", projectId,
			"sqlserverDatabaseDetails", sqlServerDbDetails,
		)
		if err != nil {
			return nil, err
		}
		mqlDbs = append(mqlDbs, mqlDb)
	}
	return mqlDbs, nil
}

func (g *mqlGcpProjectSqlServiceInstance) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", projectId, name), nil
}

func (g *mqlGcpProjectSqlServiceInstanceDatabase) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	instance, err := g.Instance()
	if err != nil {
		return "", err
	}
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/%s", projectId, instance, name), nil
}

func (g *mqlGcpProjectSqlServiceInstanceDatabaseSqlserverDatabaseDetails) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlServiceInstanceDiskEncryptionConfiguration) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlServiceInstanceDiskEncryptionStatus) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlServiceInstanceFailoverReplica) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlServiceInstanceIpMapping) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlServiceInstanceSettings) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	name, err := g.InstanceName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/settings", projectId, name), nil
}

func (g *mqlGcpProjectSqlServiceInstanceSettingsActivedirectoryconfig) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlServiceInstanceSettingsBackupconfiguration) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlServiceInstanceSettingsBackupconfigurationRetentionsettings) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlServiceInstanceSettingsDenyMaintenancePeriod) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlServiceInstanceSettingsInsightsConfig) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlServiceInstanceSettingsIpConfiguration) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlServiceInstanceSettingsIpConfigurationAclEntry) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlServiceInstanceSettingsLocationPreference) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlServiceInstanceSettingsMaintenanceWindow) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlServiceInstanceSettingsPasswordValidationPolicy) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlServiceInstanceSettingsSqlServerAuditConfig) id() (string, error) {
	return g.Id()
}
