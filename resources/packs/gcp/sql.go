package gcp

import (
	"context"
	"fmt"

	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/sqladmin/v1"
)

func (g *mqlGcpProjectSql) id() (string, error) {
	return "gcp.project.sql", nil
}

func (g *mqlGcpProject) GetSql() (interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.sql",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectSql) GetInstances() ([]interface{}, error) {
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

	res := []interface{}{}

	for i := range sqlinstances.Items {
		instance := sqlinstances.Items[i]

		settingsDict := map[string]interface{}{}
		if instance.Settings != nil {
			settings := instance.Settings
			if settings.DatabaseFlags != nil {
				dbFlags := map[string]interface{}{}
				for di := range settings.DatabaseFlags {
					flag := settings.DatabaseFlags[di]
					dbFlags[flag.Name] = flag.Value
				}
				settingsDict["databaseFlags"] = dbFlags
			}

			if settings.IpConfiguration != nil {
				ipConfig := map[string]interface{}{}

				ipConfig["ipv4Enabled"] = settings.IpConfiguration.Ipv4Enabled
				ipConfig["requireSsl"] = settings.IpConfiguration.RequireSsl
				ipConfig["privateNetwork"] = settings.IpConfiguration.PrivateNetwork

				authorizedNetworks := []interface{}{}
				for ani := range settings.IpConfiguration.AuthorizedNetworks {
					aclEntry := settings.IpConfiguration.AuthorizedNetworks[ani]

					authorizedNetworks = append(authorizedNetworks, map[string]interface{}{
						"name":           aclEntry.Name,
						"value":          aclEntry.Value,
						"kind":           aclEntry.Kind,
						"expirationTime": aclEntry.ExpirationTime,
					})
				}
				ipConfig["authorizedNetworks"] = authorizedNetworks

				settingsDict["ipConfiguration"] = ipConfig
			}

			// TODO: handle all other database settings
		}

		mqlInstance, err := g.MotorRuntime.CreateResource("gcp.project.sql.instance",
			"projectId", projectId,
			"availableMaintenanceVersions", core.StrSliceToInterface(instance.AvailableMaintenanceVersions),
			"backendType", instance.BackendType,
			"connectionName", instance.ConnectionName,
			"created", parseTime(instance.CreateTime),
			"currentDiskSize", instance.CurrentDiskSize,
			"databaseInstalledVersion", instance.DatabaseInstalledVersion,
			"databaseVersion", instance.DatabaseVersion,
			"diskEncryptionConfiguration", nil, // TODO
			"diskEncryptionStatus", nil, // TODO
			"failoverReplica", nil, // TODO
			"gceZone", instance.GceZone,
			"instanceType", instance.InstanceType,
			"ipAddresses", nil, // TODO
			"maintenanceVersion", instance.MaintenanceVersion,
			"masterInstanceName", instance.MasterInstanceName,
			"maxDiskSize", instance.MaxDiskSize,
			"name", instance.Name,
			// ref project
			"project", instance.Project,
			"region", instance.Region,
			"replicaNames", core.StrSliceToInterface(instance.ReplicaNames),
			"settings", nil, // TODO
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

func (g *mqlGcpProjectSqlInstance) id() (string, error) {
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

func (g *mqlGcpProjectSqlInstanceDiskEncryptionConfiguration) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlInstanceDiskEncryptionStatus) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlInstanceFailoverReplica) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlInstanceIpMapping) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlInstanceSettings) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlInstanceSettingsActivedirectoryconfig) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlInstanceSettingsBackupconfiguration) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlInstanceSettingsBackupconfigurationRetentionsettings) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlInstanceSettingsDenyMaintenancePeriod) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlInstanceSettingsInsightsConfig) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlInstanceSettingsIpConfiguration) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlInstanceSettingsIpConfigurationAclEntry) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlInstanceSettingsLocationPreference) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlInstanceSettingsMaintenanceWindow) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlInstanceSettingsPasswordValidationPolicy) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectSqlInstanceSettingsSqlServerAuditConfig) id() (string, error) {
	return g.Id()
}
