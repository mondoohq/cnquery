// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/snowflake/connection"
	"go.mondoo.com/mql/v13/types"
)

type mqlSnowflakeTaskInternal struct {
	cacheOwner     string
	cacheWarehouse string
}

func (r *mqlSnowflakeAccount) tasks() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	tasks, err := client.Tasks.Show(ctx, sdk.NewShowTaskRequest())
	if err != nil {
		return nil, err
	}

	list := []any{}
	for i := range tasks {
		mqlTask, err := newMqlSnowflakeTask(r.MqlRuntime, tasks[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlTask)
	}
	return list, nil
}

func newMqlSnowflakeTask(runtime *plugin.Runtime, task sdk.Task) (*mqlSnowflakeTask, error) {
	predecessors := []any{}
	for _, p := range task.Predecessors {
		predecessors = append(predecessors, p.FullyQualifiedName())
	}
	warehouse := ""
	if task.Warehouse != nil {
		warehouse = task.Warehouse.Name()
	}

	r, err := CreateResource(runtime, "snowflake.task", map[string]*llx.RawData{
		"__id":                      llx.StringData(task.ID().FullyQualifiedName()),
		"name":                      llx.StringData(task.Name),
		"databaseName":              llx.StringData(task.DatabaseName),
		"schemaName":                llx.StringData(task.SchemaName),
		"ownerRoleType":             llx.StringData(task.OwnerRoleType),
		"schedule":                  llx.StringData(task.Schedule),
		"state":                     llx.StringData(string(task.State)),
		"definition":                llx.StringData(task.Definition),
		"condition":                 llx.StringData(task.Condition),
		"allowOverlappingExecution": llx.BoolData(task.AllowOverlappingExecution),
		"predecessors":              llx.ArrayData(predecessors, types.String),
		"comment":                   llx.StringData(task.Comment),
	})
	if err != nil {
		return nil, err
	}
	mqlTask := r.(*mqlSnowflakeTask)
	mqlTask.cacheOwner = task.Owner
	mqlTask.cacheWarehouse = warehouse
	return mqlTask, nil
}

func (r *mqlSnowflakeTask) owner() (*mqlSnowflakeRole, error) {
	if r.cacheOwner == "" {
		r.Owner.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return snowflakeRoleByName(r.MqlRuntime, r.cacheOwner)
}

func (r *mqlSnowflakeTask) warehouse() (*mqlSnowflakeWarehouse, error) {
	if r.cacheWarehouse == "" {
		r.Warehouse.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	wh, err := NewResource(r.MqlRuntime, "snowflake.warehouse", map[string]*llx.RawData{
		"name": llx.StringData(r.cacheWarehouse),
	})
	if err != nil {
		return nil, err
	}
	return wh.(*mqlSnowflakeWarehouse), nil
}
