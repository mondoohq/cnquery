// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/snowflake/connection"
)

type mqlSnowflakeNetworkRuleInternal struct {
	valuesLock    sync.Mutex
	valuesLoaded  bool
	valuesLoadErr error
	values        []any
}

func (r *mqlSnowflakeAccount) networkRules() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	rules, err := client.NetworkRules.Show(ctx, sdk.NewShowNetworkRuleRequest())
	if err != nil {
		return nil, err
	}

	list := make([]any, 0, len(rules))
	for i := range rules {
		mqlRule, err := newMqlSnowflakeNetworkRule(r.MqlRuntime, rules[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlRule)
	}

	return list, nil
}

func newMqlSnowflakeNetworkRule(runtime *plugin.Runtime, rule sdk.NetworkRule) (*mqlSnowflakeNetworkRule, error) {
	r, err := CreateResource(runtime, "snowflake.networkRule", map[string]*llx.RawData{
		"__id":               llx.StringData(rule.ID().FullyQualifiedName()),
		"name":               llx.StringData(rule.Name),
		"databaseName":       llx.StringData(rule.DatabaseName),
		"schemaName":         llx.StringData(rule.SchemaName),
		"owner":              llx.StringData(rule.Owner),
		"ownerRoleType":      llx.StringData(rule.OwnerRoleType),
		"comment":            llx.StringData(rule.Comment),
		"type":               llx.StringData(string(rule.Type)),
		"mode":               llx.StringData(string(rule.Mode)),
		"entriesInValueList": llx.IntData(rule.EntriesInValueList),
		"createdAt":          llx.TimeData(rule.CreatedOn),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlSnowflakeNetworkRule), nil
}

func (r *mqlSnowflakeNetworkRule) database() (*mqlSnowflakeDatabase, error) {
	return resolveDatabaseRef(r.MqlRuntime, r.DatabaseName.Data, &r.Database)
}

func (r *mqlSnowflakeNetworkRule) schema() (*mqlSnowflakeSchema, error) {
	return resolveSchemaRef(r.MqlRuntime, r.DatabaseName.Data, r.SchemaName.Data, &r.Schema)
}

func (r *mqlSnowflakeNetworkRule) valueList() ([]any, error) {
	if r.valuesLoaded {
		return r.values, r.valuesLoadErr
	}
	r.valuesLock.Lock()
	defer r.valuesLock.Unlock()
	if r.valuesLoaded {
		return r.values, r.valuesLoadErr
	}

	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	details, err := client.NetworkRules.Describe(ctx,
		sdk.NewSchemaObjectIdentifier(r.DatabaseName.Data, r.SchemaName.Data, r.Name.Data),
	)
	if err != nil {
		r.valuesLoaded = true
		r.valuesLoadErr = err
		return nil, err
	}

	r.values = convert.SliceAnyToInterface(details.ValueList)
	r.valuesLoaded = true
	return r.values, nil
}
