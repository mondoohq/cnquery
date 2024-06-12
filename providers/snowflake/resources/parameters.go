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

func (r *mqlSnowflakeAccount) parameters() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	parameters, err := client.Parameters.ShowParameters(ctx, &sdk.ShowParametersOptions{
		In: &sdk.ParametersIn{
			Account: sdk.Bool(true),
		},
	})
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range parameters {
		mqlResource, err := newMqlSnowflakeParameter(r.MqlRuntime, parameters[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlResource)
	}

	return list, nil
}

func newMqlSnowflakeParameter(runtime *plugin.Runtime, parameter *sdk.Parameter) (*mqlSnowflakeParameter, error) {
	r, err := CreateResource(runtime, "snowflake.parameter", map[string]*llx.RawData{
		"__id":         llx.StringData(parameter.Key), // TODO: update key
		"key":          llx.StringData(parameter.Key),
		"value":        llx.StringData(parameter.Value),
		"description":  llx.StringData(parameter.Description),
		"defaultValue": llx.StringData(parameter.Default),
		"level":        llx.StringData(string(parameter.Level)),
	})
	if err != nil {
		return nil, err
	}
	mqlResource := r.(*mqlSnowflakeParameter)
	return mqlResource, nil
}
