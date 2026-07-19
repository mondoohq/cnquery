// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/snowflake/connection"
)

// initSnowflakeSchema resolves a schema by its database and name when the
// resource is requested through a typed reference (e.g.
// snowflake.rowAccessPolicy.schema). Listing via the account goes through
// CreateResource, which skips init, so this only runs for on-demand lookups.
func initSnowflakeSchema(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// Keyed by databaseName + name; more than those two keys means the caller
	// already supplied full data, so skip the lookup.
	if len(args) > 2 {
		return args, nil, nil
	}
	dbRaw, ok1 := args["databaseName"]
	nameRaw, ok2 := args["name"]
	if !ok1 || !ok2 {
		return args, nil, nil
	}
	databaseName, _ := dbRaw.Value.(string)
	name, _ := nameRaw.Value.(string)
	if databaseName == "" || name == "" {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	schemas, err := client.Schemas.Show(ctx, &sdk.ShowSchemaOptions{
		Like: &sdk.Like{Pattern: sdk.String(name)},
		In:   &sdk.SchemaIn{Database: sdk.Bool(true), Name: sdk.NewAccountObjectIdentifier(databaseName)},
	})
	if err != nil {
		return nil, nil, err
	}
	for i := range schemas {
		if schemas[i].Name == name && schemas[i].DatabaseName == databaseName {
			res, err := newMqlSnowflakeSchema(runtime, schemas[i])
			if err != nil {
				return nil, nil, err
			}
			return nil, res, nil
		}
	}
	return nil, nil, fmt.Errorf("snowflake.schema %q not found in database %q", name, databaseName)
}

// resolveSchemaRef returns the typed schema a database/name pair refers to, or a
// null resource when either coordinate is empty. Shared by the row-access-policy
// and network-rule resources.
func resolveSchemaRef(runtime *plugin.Runtime, databaseName, schemaName string, field *plugin.TValue[*mqlSnowflakeSchema]) (*mqlSnowflakeSchema, error) {
	if databaseName == "" || schemaName == "" {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(runtime, "snowflake.schema", map[string]*llx.RawData{
		"databaseName": llx.StringData(databaseName),
		"name":         llx.StringData(schemaName),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlSnowflakeSchema), nil
}

func (r *mqlSnowflakeSchema) database() (*mqlSnowflakeDatabase, error) {
	return resolveDatabaseRef(r.MqlRuntime, r.DatabaseName.Data, &r.Database)
}

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
