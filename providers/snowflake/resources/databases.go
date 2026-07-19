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

// initSnowflakeDatabase resolves a database by name when the resource is
// requested through a typed reference (e.g. snowflake.schema.database). Listing
// via the account goes through CreateResource, which skips init, so this only
// runs for on-demand lookups.
func initSnowflakeDatabase(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	nameRaw, ok := args["name"]
	if !ok {
		return args, nil, nil
	}
	name, ok := nameRaw.Value.(string)
	if !ok || name == "" {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	databases, err := client.Databases.Show(ctx, &sdk.ShowDatabasesOptions{
		Like: &sdk.Like{Pattern: sdk.String(name)},
	})
	if err != nil {
		return nil, nil, err
	}
	for i := range databases {
		if databases[i].Name == name {
			res, err := newMqlSnowflakeDatabase(runtime, databases[i])
			if err != nil {
				return nil, nil, err
			}
			return nil, res, nil
		}
	}
	return nil, nil, fmt.Errorf("snowflake.database %q not found", name)
}

// resolveDatabaseRef returns the typed database a name refers to, or a null
// resource when the name is empty. Shared by the schema, row-access-policy, and
// network-rule resources.
func resolveDatabaseRef(runtime *plugin.Runtime, name string, field *plugin.TValue[*mqlSnowflakeDatabase]) (*mqlSnowflakeDatabase, error) {
	if name == "" {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(runtime, "snowflake.database", map[string]*llx.RawData{
		"name": llx.StringData(name),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlSnowflakeDatabase), nil
}

func (r *mqlSnowflakeAccount) databases() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	databases, err := client.Databases.Show(ctx, &sdk.ShowDatabasesOptions{})
	if err != nil {
		return nil, err
	}

	list := []any{}
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
	var origin string
	if database.Origin != nil {
		origin = database.Origin.FullyQualifiedName()
	}
	r, err := CreateResource(runtime, "snowflake.database", map[string]*llx.RawData{
		"__id":          llx.StringData(database.ID().FullyQualifiedName()),
		"name":          llx.StringData(database.Name),
		"isDefault":     llx.BoolData(database.IsDefault),
		"isCurrent":     llx.BoolData(database.IsCurrent),
		"origin":        llx.StringData(origin),
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
