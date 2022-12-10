package gcp

import (
	"context"

	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/sqladmin/v1"
)

func (g *mqlGcpSql) id() (string, error) {
	return "gcp.sql", nil
}

func (g *mqlGcpSql) GetInstances() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
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

	projectName := provider.ResourceID()
	sqlinstances, err := sqladminSvc.Instances.List(projectName).Do()
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

		mqlInstance, err := g.MotorRuntime.CreateResource("gcp.sql.instance",
			"name", instance.Name,
			"backendType", instance.BackendType,
			"connectionName", instance.ConnectionName,
			"databaseVersion", instance.DatabaseVersion,
			"gceZone", instance.GceZone,
			"instanceType", instance.InstanceType,
			"kind", instance.Kind,
			"currentDiskSize", instance.CurrentDiskSize,
			"maxDiskSize", instance.MaxDiskSize,
			"state", instance.State,
			// ref project
			"project", instance.Project,
			"region", instance.Region,
			"serviceAccountEmailAddress", instance.ServiceAccountEmailAddress,
			"settings", settingsDict,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (g *mqlGcpSqlInstance) id() (string, error) {
	// TODO: instances are scoped in project
	return g.Name()
}
