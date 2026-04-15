// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	spannerdatabase "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	spannerinstance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProject) spanner() (*mqlGcpProjectSpannerService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	res, err := CreateResource(g.MqlRuntime, "gcp.project.spannerService", map[string]*llx.RawData{
		"projectId": llx.StringData(g.Id.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectSpannerService), nil
}

func initGcpProjectSpannerService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}
	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	args["projectId"] = llx.StringData(conn.ResourceID())
	return args, nil, nil
}

func (g *mqlGcpProjectSpannerService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf("gcp.project/%s/spannerService", g.ProjectId.Data), nil
}

func (g *mqlGcpProjectSpannerService) instances() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(spannerinstance.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := spannerinstance.NewInstanceAdminClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListInstances(ctx, &instancepb.ListInstancesRequest{
		Parent: fmt.Sprintf("projects/%s", projectId),
	})

	var res []any
	for {
		inst, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		autoscalingConfig, err := protoToDict(inst.AutoscalingConfig)
		if err != nil {
			return nil, err
		}

		var createdAt *llx.RawData
		if inst.CreateTime != nil {
			createdAt = llx.TimeData(inst.CreateTime.AsTime())
		} else {
			createdAt = llx.NilData
		}

		var updatedAt *llx.RawData
		if inst.UpdateTime != nil {
			updatedAt = llx.TimeData(inst.UpdateTime.AsTime())
		} else {
			updatedAt = llx.NilData
		}

		mqlInst, err := CreateResource(g.MqlRuntime, "gcp.project.spannerService.instance", map[string]*llx.RawData{
			"projectId":                 llx.StringData(projectId),
			"name":                      llx.StringData(inst.Name),
			"displayName":               llx.StringData(inst.DisplayName),
			"config":                    llx.StringData(inst.Config),
			"nodeCount":                 llx.IntData(int64(inst.NodeCount)),
			"processingUnits":           llx.IntData(int64(inst.ProcessingUnits)),
			"state":                     llx.StringData(inst.State.String()),
			"instanceType":              llx.StringData(inst.InstanceType.String()),
			"labels":                    llx.MapData(convert.MapToInterfaceMap(inst.Labels), types.String),
			"edition":                   llx.StringData(inst.Edition.String()),
			"autoscalingConfig":         llx.DictData(autoscalingConfig),
			"defaultBackupScheduleType": llx.StringData(inst.DefaultBackupScheduleType.String()),
			"createdAt":                 createdAt,
			"updatedAt":                 updatedAt,
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInst)
	}

	return res, nil
}

func (g *mqlGcpProjectSpannerServiceInstance) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	return fmt.Sprintf("gcp.project/%s/spannerService/%s", g.ProjectId.Data, g.Name.Data), nil
}

func (g *mqlGcpProjectSpannerServiceInstance) databases() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	projectId := g.ProjectId.Data
	instanceName := g.Name.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(spannerdatabase.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := spannerdatabase.NewDatabaseAdminClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListDatabases(ctx, &databasepb.ListDatabasesRequest{
		Parent: instanceName,
	})

	var res []any
	for {
		db, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		encryptionConfig, err := protoToDict(db.EncryptionConfig)
		if err != nil {
			return nil, err
		}

		encryptionInfo := make([]any, 0, len(db.EncryptionInfo))
		for _, ei := range db.EncryptionInfo {
			eiDict, err := protoToDict(ei)
			if err != nil {
				return nil, err
			}
			encryptionInfo = append(encryptionInfo, eiDict)
		}

		var earliestVersionTime *llx.RawData
		if db.EarliestVersionTime != nil {
			earliestVersionTime = llx.TimeData(db.EarliestVersionTime.AsTime())
		} else {
			earliestVersionTime = llx.NilData
		}

		var createdAt *llx.RawData
		if db.CreateTime != nil {
			createdAt = llx.TimeData(db.CreateTime.AsTime())
		} else {
			createdAt = llx.NilData
		}

		mqlDb, err := CreateResource(g.MqlRuntime, "gcp.project.spannerService.instance.database", map[string]*llx.RawData{
			"projectId":              llx.StringData(projectId),
			"instanceName":           llx.StringData(instanceName),
			"name":                   llx.StringData(db.Name),
			"state":                  llx.StringData(db.State.String()),
			"databaseDialect":        llx.StringData(db.DatabaseDialect.String()),
			"versionRetentionPeriod": llx.StringData(db.VersionRetentionPeriod),
			"earliestVersionTime":    earliestVersionTime,
			"encryptionConfig":       llx.DictData(encryptionConfig),
			"encryptionInfo":         llx.ArrayData(encryptionInfo, types.Dict),
			"defaultLeader":          llx.StringData(db.DefaultLeader),
			"enableDropProtection":   llx.BoolData(db.EnableDropProtection),
			"reconciling":            llx.BoolData(db.Reconciling),
			"createdAt":              createdAt,
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlDb)
	}

	return res, nil
}

func (g *mqlGcpProjectSpannerServiceInstanceDatabase) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	if g.InstanceName.Error != nil {
		return "", g.InstanceName.Error
	}
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	return fmt.Sprintf("gcp.project/%s/spannerService/%s/database/%s", g.ProjectId.Data, g.InstanceName.Data, g.Name.Data), nil
}

func (g *mqlGcpProjectSpannerServiceInstanceDatabase) ddl() ([]any, error) {
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	dbName := g.Name.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(spannerdatabase.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := spannerdatabase.NewDatabaseAdminClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	resp, err := client.GetDatabaseDdl(ctx, &databasepb.GetDatabaseDdlRequest{
		Database: dbName,
	})
	if err != nil {
		return nil, err
	}

	return convert.SliceAnyToInterface(resp.Statements), nil
}

func (g *mqlGcpProjectSpannerServiceInstance) backups() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	projectId := g.ProjectId.Data
	instanceName := g.Name.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(spannerdatabase.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := spannerdatabase.NewDatabaseAdminClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListBackups(ctx, &databasepb.ListBackupsRequest{
		Parent: instanceName,
	})

	var res []any
	for {
		backup, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		encryptionInfo, err := protoToDict(backup.EncryptionInfo)
		if err != nil {
			return nil, err
		}

		var expireTime *llx.RawData
		if backup.ExpireTime != nil {
			expireTime = llx.TimeData(backup.ExpireTime.AsTime())
		} else {
			expireTime = llx.NilData
		}

		var versionTime *llx.RawData
		if backup.VersionTime != nil {
			versionTime = llx.TimeData(backup.VersionTime.AsTime())
		} else {
			versionTime = llx.NilData
		}

		var createdAt *llx.RawData
		if backup.CreateTime != nil {
			createdAt = llx.TimeData(backup.CreateTime.AsTime())
		} else {
			createdAt = llx.NilData
		}

		var maxExpireTime *llx.RawData
		if backup.MaxExpireTime != nil {
			maxExpireTime = llx.TimeData(backup.MaxExpireTime.AsTime())
		} else {
			maxExpireTime = llx.NilData
		}

		mqlBackup, err := CreateResource(g.MqlRuntime, "gcp.project.spannerService.instance.backup", map[string]*llx.RawData{
			"projectId":       llx.StringData(projectId),
			"instanceName":    llx.StringData(instanceName),
			"name":            llx.StringData(backup.Name),
			"database":        llx.StringData(backup.Database),
			"state":           llx.StringData(backup.State.String()),
			"expireTime":      expireTime,
			"versionTime":     versionTime,
			"createdAt":       createdAt,
			"sizeBytes":       llx.IntData(backup.SizeBytes),
			"encryptionInfo":  llx.DictData(encryptionInfo),
			"databaseDialect": llx.StringData(backup.DatabaseDialect.String()),
			"maxExpireTime":   maxExpireTime,
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBackup)
	}

	return res, nil
}

func (g *mqlGcpProjectSpannerServiceInstanceBackup) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	if g.InstanceName.Error != nil {
		return "", g.InstanceName.Error
	}
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	return fmt.Sprintf("gcp.project/%s/spannerService/%s/backup/%s", g.ProjectId.Data, g.InstanceName.Data, g.Name.Data), nil
}

func (g *mqlGcpProjectSpannerService) instanceConfigs() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(spannerinstance.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := spannerinstance.NewInstanceAdminClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListInstanceConfigs(ctx, &instancepb.ListInstanceConfigsRequest{
		Parent: fmt.Sprintf("projects/%s", projectId),
	})

	var res []any
	for {
		cfg, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		replicas := make([]any, 0, len(cfg.Replicas))
		for _, r := range cfg.Replicas {
			replicaDict, err := protoToDict(r)
			if err != nil {
				return nil, err
			}
			replicas = append(replicas, replicaDict)
		}

		mqlCfg, err := CreateResource(g.MqlRuntime, "gcp.project.spannerService.instanceConfig", map[string]*llx.RawData{
			"projectId":                llx.StringData(projectId),
			"name":                     llx.StringData(cfg.Name),
			"displayName":              llx.StringData(cfg.DisplayName),
			"replicas":                 llx.ArrayData(replicas, types.Dict),
			"leaderOptions":            llx.ArrayData(convert.SliceAnyToInterface(cfg.LeaderOptions), types.String),
			"baseConfig":               llx.StringData(cfg.BaseConfig),
			"configType":               llx.StringData(cfg.ConfigType.String()),
			"freeInstanceAvailability": llx.StringData(cfg.FreeInstanceAvailability.String()),
			"labels":                   llx.MapData(convert.MapToInterfaceMap(cfg.Labels), types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCfg)
	}

	return res, nil
}

func (g *mqlGcpProjectSpannerServiceInstanceConfig) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	return fmt.Sprintf("gcp.project/%s/spannerService/instanceConfig/%s", g.ProjectId.Data, g.Name.Data), nil
}

func initGcpProjectSpannerServiceInstanceConfig(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	nameRaw, ok := args["name"]
	if !ok {
		return args, nil, nil
	}
	name := nameRaw.Value.(string)

	projectIdRaw, ok := args["projectId"]
	if !ok {
		return args, nil, nil
	}
	projectId := projectIdRaw.Value.(string)

	obj, err := CreateResource(runtime, "gcp.project.spannerService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, nil, err
	}
	svc := obj.(*mqlGcpProjectSpannerService)
	configs := svc.GetInstanceConfigs()
	if configs.Error != nil {
		return nil, nil, configs.Error
	}

	for _, c := range configs.Data {
		cfg := c.(*mqlGcpProjectSpannerServiceInstanceConfig)
		if cfg.Name.Data == name {
			return args, cfg, nil
		}
	}

	return nil, nil, fmt.Errorf("spanner instance config %q not found in project %q", name, projectId)
}

func (g *mqlGcpProjectSpannerServiceInstance) instanceConfig() (*mqlGcpProjectSpannerServiceInstanceConfig, error) {
	if g.Config.Error != nil {
		return nil, g.Config.Error
	}
	configName := g.Config.Data
	if configName == "" {
		g.InstanceConfig.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	res, err := NewResource(g.MqlRuntime, "gcp.project.spannerService.instanceConfig", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
		"name":      llx.StringData(configName),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectSpannerServiceInstanceConfig), nil
}
