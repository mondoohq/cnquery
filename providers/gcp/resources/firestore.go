// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	firestoreadmin "cloud.google.com/go/firestore/apiv1/admin"
	"cloud.google.com/go/firestore/apiv1/admin/adminpb"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"google.golang.org/api/option"
)

func (g *mqlGcpProject) firestore() (*mqlGcpProjectFirestoreService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	res, err := CreateResource(g.MqlRuntime, "gcp.project.firestoreService", map[string]*llx.RawData{
		"projectId": llx.StringData(g.Id.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectFirestoreService), nil
}

func initGcpProjectFirestoreService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (g *mqlGcpProjectFirestoreService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf("gcp.project/%s/firestoreService", g.ProjectId.Data), nil
}

func (g *mqlGcpProjectFirestoreService) databases() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(firestoreadmin.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := firestoreadmin.NewFirestoreAdminClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	resp, err := client.ListDatabases(ctx, &adminpb.ListDatabasesRequest{
		Parent: fmt.Sprintf("projects/%s", projectId),
	})
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(resp.Databases))
	for _, db := range resp.Databases {
		cmekConfig, err := protoToDict(db.CmekConfig)
		if err != nil {
			return nil, err
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

		var updatedAt *llx.RawData
		if db.UpdateTime != nil {
			updatedAt = llx.TimeData(db.UpdateTime.AsTime())
		} else {
			updatedAt = llx.NilData
		}

		var versionRetentionPeriod string
		if db.VersionRetentionPeriod != nil {
			versionRetentionPeriod = db.VersionRetentionPeriod.String()
		}

		mqlDb, err := CreateResource(g.MqlRuntime, "gcp.project.firestoreService.database", map[string]*llx.RawData{
			"projectId":                     llx.StringData(projectId),
			"name":                          llx.StringData(db.Name),
			"uid":                           llx.StringData(db.Uid),
			"locationId":                    llx.StringData(db.LocationId),
			"type":                          llx.StringData(db.Type.String()),
			"concurrencyMode":               llx.StringData(db.ConcurrencyMode.String()),
			"appEngineIntegrationMode":      llx.StringData(db.AppEngineIntegrationMode.String()),
			"pointInTimeRecoveryEnablement": llx.StringData(db.PointInTimeRecoveryEnablement.String()),
			"deleteProtectionState":         llx.StringData(db.DeleteProtectionState.String()),
			"cmekConfig":                    llx.DictData(cmekConfig),
			"versionRetentionPeriod":        llx.StringData(versionRetentionPeriod),
			"earliestVersionTime":           earliestVersionTime,
			"etag":                          llx.StringData(db.Etag),
			"createdAt":                     createdAt,
			"updatedAt":                     updatedAt,
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlDb)
	}

	return res, nil
}

func (g *mqlGcpProjectFirestoreServiceDatabase) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	return fmt.Sprintf("gcp.project/%s/firestoreService/%s", g.ProjectId.Data, g.Name.Data), nil
}
