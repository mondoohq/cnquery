// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/snowflake/connection"
)

func (r *mqlSnowflakeAccount) stages() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	stages, err := client.Stages.Show(ctx, &sdk.ShowStageRequest{})
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range stages {
		mqlResource, err := newMqlSnowflakeStage(r.MqlRuntime, stages[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlResource)
	}

	return list, nil
}

func newMqlSnowflakeStage(runtime *plugin.Runtime, user sdk.Stage) (*mqlSnowflakeStage, error) {
	r, err := CreateResource(runtime, "snowflake.stage", map[string]*llx.RawData{
		"__id":             llx.StringData(user.ID().FullyQualifiedName()),
		"name":             llx.StringData(user.Name),
		"databaseName":     llx.StringData(user.DatabaseName),
		"schemaName":       llx.StringData(user.SchemaName),
		"owner":            llx.StringData(user.Owner),
		"comment":          llx.StringData(user.Comment),
		"createdAt":        llx.TimeData(user.CreatedOn),
		"hasCredentials":   llx.BoolData(user.HasCredentials),
		"hasEncryptionKey": llx.BoolData(user.HasEncryptionKey),
		"url":              llx.StringData(user.Url),
		"type":             llx.StringData(user.Type),
		"cloud":            llx.StringDataPtr(user.Cloud),
		"storeIntegration": llx.StringDataPtr(user.StorageIntegration),
		"endpoint":         llx.StringDataPtr(user.Endpoint),
		"ownerRoleType":    llx.StringDataPtr(user.OwnerRoleType),
		"directoryEnabled": llx.BoolData(user.DirectoryEnabled),
	})
	if err != nil {
		return nil, err
	}
	mqlResource := r.(*mqlSnowflakeStage)
	return mqlResource, nil
}
