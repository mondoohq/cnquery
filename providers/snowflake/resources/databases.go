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

func (r *mqlSnowflakeAccount) databases() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	databases, err := client.Databases.Show(ctx, &sdk.ShowDatabasesOptions{})
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range databases {
		mqlResource, err := newMqlSnowflakeDatabase(r.MqlRuntime, databases[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlResource)
	}

	return list, nil
}

func newMqlSnowflakeDatabase(runtime *plugin.Runtime, database sdk.Database) (*mqlSnowflakeDatabase, error) {
	r, err := CreateResource(runtime, "snowflake.database", map[string]*llx.RawData{
		"__id":          llx.StringData(database.ID().FullyQualifiedName()),
		"name":          llx.StringData(database.Name),
		"isDefault":     llx.BoolData(database.IsDefault),
		"isCurrent":     llx.BoolData(database.IsCurrent),
		"origin":        llx.StringData(database.Origin),
		"owner":         llx.StringData(database.Owner),
		"comment":       llx.StringData(database.Comment),
		"options":       llx.StringData(database.Options),
		"retentionTime": llx.IntData(database.RetentionTime),
		"resourceGroup": llx.StringData(database.ResourceGroup),
		"transient":     llx.BoolData(database.Transient),
		"createdAt":     llx.TimeData(database.CreatedOn),
		"droppedAt":     llx.TimeData(database.DroppedOn),
	})
	if err != nil {
		return nil, err
	}
	mqlResource := r.(*mqlSnowflakeDatabase)
	return mqlResource, nil
}
