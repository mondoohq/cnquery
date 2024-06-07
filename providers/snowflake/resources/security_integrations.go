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

func (r *mqlSnowflakeAccount) securityIntegrations() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	integrations, err := client.SecurityIntegrations.Show(ctx, &sdk.ShowSecurityIntegrationRequest{})
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range integrations {
		mqlSecurityIntegration, err := newMqlSnowflakeSecurityIntegration(r.MqlRuntime, integrations[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlSecurityIntegration)
	}

	return list, nil
}

func newMqlSnowflakeSecurityIntegration(runtime *plugin.Runtime, integration sdk.SecurityIntegration) (*mqlSnowflakeSecurityIntegration, error) {
	r, err := CreateResource(runtime, "snowflake.securityIntegration", map[string]*llx.RawData{
		"__id":      llx.StringData(integration.Name), // TODO: update key
		"name":      llx.StringData(integration.Name),
		"type":      llx.StringData(integration.IntegrationType),
		"comment":   llx.StringData(integration.Comment),
		"enabled":   llx.BoolData(integration.Enabled),
		"createdAt": llx.TimeData(integration.CreatedOn),
		"category":  llx.StringData(integration.Category),
	})
	if err != nil {
		return nil, err
	}
	mqlResource := r.(*mqlSnowflakeSecurityIntegration)
	return mqlResource, nil
}
