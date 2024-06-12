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

func (r *mqlSnowflakeAccount) warehouses() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	warehouses, err := client.Warehouses.Show(ctx, &sdk.ShowWarehouseOptions{})
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
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
		"comment":                         llx.StringData(warehouse.Comment),
		"enableQueryAcceleration":         llx.BoolData(warehouse.EnableQueryAcceleration),
		"queryAccelerationMaxScaleFactor": llx.IntData(warehouse.QueryAccelerationMaxScaleFactor),
		"resourceMonitor":                 llx.StringData(warehouse.ResourceMonitor),
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
