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

type mqlSnowflakeExternalTableInternal struct {
	cacheOwner string
	cacheStage string
}

func (r *mqlSnowflakeAccount) externalTables() ([]any, error) {
	return listSnowflakeExternalTables(r.MqlRuntime,
		sdk.NewShowExternalTableInRequest().WithAccount(true))
}

func (r *mqlSnowflakeDatabase) externalTables() ([]any, error) {
	return listSnowflakeExternalTables(r.MqlRuntime,
		sdk.NewShowExternalTableInRequest().WithDatabase(sdk.NewAccountObjectIdentifier(r.Name.Data)))
}

// listSnowflakeExternalTables fetches external tables within the given scope
// (account-wide or a single database) and maps them to resources.
func listSnowflakeExternalTables(runtime *plugin.Runtime, in *sdk.ShowExternalTableInRequest) ([]any, error) {
	conn := runtime.Connection.(*connection.SnowflakeConnection)
	tables, err := conn.Client().ExternalTables.Show(context.Background(),
		sdk.NewShowExternalTableRequest().WithIn(*in))
	if err != nil {
		return nil, err
	}

	list := make([]any, 0, len(tables))
	for i := range tables {
		mqlTable, err := newMqlSnowflakeExternalTable(runtime, tables[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlTable)
	}
	return list, nil
}

func newMqlSnowflakeExternalTable(runtime *plugin.Runtime, table sdk.ExternalTable) (*mqlSnowflakeExternalTable, error) {
	r, err := CreateResource(runtime, "snowflake.externalTable", map[string]*llx.RawData{
		"__id":                llx.StringData(snowflakeSchemaObjectID(table.DatabaseName, table.SchemaName, table.Name)),
		"name":                llx.StringData(table.Name),
		"databaseName":        llx.StringData(table.DatabaseName),
		"schemaName":          llx.StringData(table.SchemaName),
		"ownerRoleType":       llx.StringData(table.OwnerRoleType),
		"location":            llx.StringData(table.Location),
		"cloud":               llx.StringData(table.Cloud),
		"region":              llx.StringData(table.Region),
		"fileFormatType":      llx.StringData(table.FileFormatType),
		"tableFormat":         llx.StringData(table.TableFormat),
		"notificationChannel": llx.StringData(table.NotificationChannel),
		"invalid":             llx.BoolData(table.Invalid),
		"invalidReason":       llx.StringData(table.InvalidReason),
		"lastRefreshedOn":     llx.TimeData(table.LastRefreshedOn),
		"lastRefreshDetails":  llx.StringData(table.LastRefreshDetails),
		"comment":             llx.StringData(table.Comment),
		"createdAt":           llx.TimeData(table.CreatedOn),
	})
	if err != nil {
		return nil, err
	}
	mqlTable := r.(*mqlSnowflakeExternalTable)
	mqlTable.cacheOwner = table.Owner
	mqlTable.cacheStage = table.Stage
	return mqlTable, nil
}

func (r *mqlSnowflakeExternalTable) database() (*mqlSnowflakeDatabase, error) {
	return resolveDatabaseRef(r.MqlRuntime, r.DatabaseName.Data, &r.Database)
}

func (r *mqlSnowflakeExternalTable) schema() (*mqlSnowflakeSchema, error) {
	return resolveSchemaRef(r.MqlRuntime, r.DatabaseName.Data, r.SchemaName.Data, &r.Schema)
}

func (r *mqlSnowflakeExternalTable) ownerRole() (*mqlSnowflakeRole, error) {
	return resolveOwnerRole(r.MqlRuntime, r.cacheOwner, &r.OwnerRole)
}

func (r *mqlSnowflakeExternalTable) stage() (*mqlSnowflakeStage, error) {
	return resolveStageRef(r.MqlRuntime, r.cacheStage, &r.Stage)
}
