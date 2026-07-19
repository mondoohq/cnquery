// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/snowflake/connection"
)

func (r *mqlSnowflakeAccount) schemas() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	schemas, err := client.Schemas.Show(ctx, &sdk.ShowSchemaOptions{
		In: &sdk.SchemaIn{Account: sdk.Bool(true)},
	})
	if err != nil {
		return nil, err
	}

	list := make([]any, 0, len(schemas))
	for i := range schemas {
		mqlSchema, err := newMqlSnowflakeSchema(r.MqlRuntime, schemas[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlSchema)
	}

	return list, nil
}

func newMqlSnowflakeSchema(runtime *plugin.Runtime, schema sdk.Schema) (*mqlSnowflakeSchema, error) {
	r, err := CreateResource(runtime, "snowflake.schema", map[string]*llx.RawData{
		"__id":          llx.StringData(schema.ID().FullyQualifiedName()),
		"name":          llx.StringData(schema.Name),
		"databaseName":  llx.StringData(schema.DatabaseName),
		"isDefault":     llx.BoolData(schema.IsDefault),
		"isCurrent":     llx.BoolData(schema.IsCurrent),
		"owner":         llx.StringData(schema.Owner),
		"ownerRoleType": llx.StringData(schema.OwnerRoleType),
		"comment":       llx.StringData(schema.Comment),
		"options":       llx.StringDataPtr(schema.Options),
		"retentionTime": llx.StringData(schema.RetentionTime),
		"createdAt":     llx.TimeData(schema.CreatedOn),
		"droppedAt":     llx.TimeData(schema.DroppedOn),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlSnowflakeSchema), nil
}
