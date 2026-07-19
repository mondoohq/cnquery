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

func (r *mqlSnowflakeAccount) functions() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	functions, err := client.Functions.Show(ctx, sdk.NewShowFunctionRequest())
	if err != nil {
		return nil, err
	}

	list := []any{}
	for i := range functions {
		mqlFunction, err := newMqlSnowflakeFunction(r.MqlRuntime, functions[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlFunction)
	}
	return list, nil
}

func newMqlSnowflakeFunction(runtime *plugin.Runtime, function sdk.Function) (*mqlSnowflakeFunction, error) {
	externalAccessIntegrations := ""
	if function.ExternalAccessIntegrations != nil {
		externalAccessIntegrations = *function.ExternalAccessIntegrations
	}
	secrets := ""
	if function.Secrets != nil {
		secrets = *function.Secrets
	}

	r, err := CreateResource(runtime, "snowflake.function", map[string]*llx.RawData{
		"__id":                       llx.StringData(function.ID().FullyQualifiedName()),
		"name":                       llx.StringData(function.Name),
		"databaseName":               llx.StringData(function.CatalogName),
		"schemaName":                 llx.StringData(function.SchemaName),
		"language":                   llx.StringData(function.Language),
		"isSecure":                   llx.BoolData(function.IsSecure),
		"isExternalFunction":         llx.BoolData(function.IsExternalFunction),
		"isBuiltin":                  llx.BoolData(function.IsBuiltin),
		"isAggregate":                llx.BoolData(function.IsAggregate),
		"isTableFunction":            llx.BoolData(function.IsTableFunction),
		"isMemoizable":               llx.BoolData(function.IsMemoizable),
		"isDataMetric":               llx.BoolData(function.IsDataMetric),
		"arguments":                  llx.StringData(function.ArgumentsRaw),
		"description":                llx.StringData(function.Description),
		"externalAccessIntegrations": llx.StringData(externalAccessIntegrations),
		"secrets":                    llx.StringData(secrets),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlSnowflakeFunction), nil
}
