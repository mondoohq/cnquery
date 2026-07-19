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

// initSnowflakeWarehouse resolves a single warehouse by name so typed
// references (such as snowflake.task.warehouse) can hydrate a full warehouse
// from just its name. A caller that already supplied more than the name is left
// untouched.
func initSnowflakeWarehouse(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	nameRaw, ok := args["name"]
	if !ok {
		return args, nil, nil
	}
	name, _ := nameRaw.Value.(string)
	if name == "" {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	warehouses, err := client.Warehouses.Show(ctx, &sdk.ShowWarehouseOptions{Like: &sdk.Like{Pattern: sdk.String(name)}})
	if err != nil {
		return nil, nil, err
	}
	for i := range warehouses {
		if warehouses[i].Name == name {
			res, err := newMqlSnowflakeWarehouse(runtime, warehouses[i])
			if err != nil {
				return nil, nil, err
			}
			return nil, res, nil
		}
	}
	return nil, nil, fmt.Errorf("snowflake.warehouse %q not found", name)
}

func (r *mqlSnowflakeAccount) warehouses() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	warehouses, err := client.Warehouses.Show(ctx, &sdk.ShowWarehouseOptions{})
	if err != nil {
		return nil, err
	}

	list := []any{}
	for i := range warehouses {
		mqlResource, err := newMqlSnowflakeWarehouse(r.MqlRuntime, warehouses[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlResource)
	}

	return list, nil
}

func newMqlSnowflakeWarehouse(runtime *plugin.Runtime, warehouse sdk.Warehouse) (*mqlSnowflakeWarehouse, error) {
	r, err := CreateResource(runtime, "snowflake.warehouse", map[string]*llx.RawData{
		"__id":                            llx.StringData(warehouse.ID().FullyQualifiedName()),
		"name":                            llx.StringData(warehouse.Name),
		"state":                           llx.StringData(string(warehouse.State)),
		"type":                            llx.StringData(string(warehouse.Type)),
		"size":                            llx.StringData(string(warehouse.Size)),
		"minClusterCount":                 llx.IntData(warehouse.MinClusterCount),
		"maxClusterCount":                 llx.IntData(warehouse.MaxClusterCount),
		"startedClusterCount":             llx.IntData(warehouse.StartedClusters),
		"running":                         llx.IntData(warehouse.Running),
		"queued":                          llx.IntData(warehouse.Queued),
		"isDefault":                       llx.BoolData(warehouse.IsDefault),
		"isCurrent":                       llx.BoolData(warehouse.IsCurrent),
		"autoSuspend":                     llx.IntData(warehouse.AutoSuspend),
		"autoResume":                      llx.BoolData(warehouse.AutoResume),
		"available":                       llx.FloatData(warehouse.Available),
		"provisioning":                    llx.FloatData(warehouse.Provisioning),
		"quiescing":                       llx.FloatData(warehouse.Quiescing),
		"other":                           llx.FloatData(warehouse.Other),
		"owner":                           llx.StringData(warehouse.Owner),
		"ownerRoleType":                   llx.StringData(warehouse.OwnerRoleType),
		"comment":                         llx.StringData(warehouse.Comment),
		"enableQueryAcceleration":         llx.BoolData(warehouse.EnableQueryAcceleration),
		"queryAccelerationMaxScaleFactor": llx.IntData(warehouse.QueryAccelerationMaxScaleFactor),
		"resourceMonitor":                 llx.StringData(warehouse.ResourceMonitor.Name()),
		"scalingPolicy":                   llx.StringData(string(warehouse.ScalingPolicy)),
		"createdAt":                       llx.TimeData(warehouse.CreatedOn),
		"resumedAt":                       llx.TimeData(warehouse.ResumedOn),
		"updatedAt":                       llx.TimeData(warehouse.UpdatedOn),
	})
	if err != nil {
		return nil, err
	}
	mqlResource := r.(*mqlSnowflakeWarehouse)
	return mqlResource, nil
}

func (r *mqlSnowflakeWarehouse) resourceMonitorRef() (*mqlSnowflakeResourceMonitor, error) {
	return snowflakeResourceMonitorByName(r.MqlRuntime, r.ResourceMonitor.Data, &r.ResourceMonitorRef)
}
