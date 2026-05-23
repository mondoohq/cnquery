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

func (r *mqlSnowflakeAccount) resourceMonitors() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	monitors, err := client.ResourceMonitors.Show(ctx, &sdk.ShowResourceMonitorOptions{})
	if err != nil {
		return nil, err
	}

	list := make([]any, 0, len(monitors))
	for i := range monitors {
		mqlMonitor, err := newMqlSnowflakeResourceMonitor(r.MqlRuntime, monitors[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlMonitor)
	}

	return list, nil
}

func newMqlSnowflakeResourceMonitor(runtime *plugin.Runtime, monitor sdk.ResourceMonitor) (*mqlSnowflakeResourceMonitor, error) {
	level := ""
	if monitor.Level != nil {
		level = string(*monitor.Level)
	}

	notifyAt := make([]any, 0, len(monitor.NotifyAt))
	for _, n := range monitor.NotifyAt {
		notifyAt = append(notifyAt, int64(n))
	}

	notifyUsers := make([]any, 0, len(monitor.NotifyUsers))
	for _, u := range monitor.NotifyUsers {
		notifyUsers = append(notifyUsers, u)
	}

	args := map[string]*llx.RawData{
		"__id":             llx.StringData(monitor.ID().FullyQualifiedName()),
		"name":             llx.StringData(monitor.Name),
		"level":            llx.StringData(level),
		"creditQuota":      llx.FloatData(monitor.CreditQuota),
		"usedCredits":      llx.FloatData(monitor.UsedCredits),
		"remainingCredits": llx.FloatData(monitor.RemainingCredits),
		"frequency":        llx.StringData(string(monitor.Frequency)),
		"startTime":        llx.StringData(monitor.StartTime),
		"endTime":          llx.StringData(monitor.EndTime),
		"owner":            llx.StringData(monitor.Owner),
		"comment":          llx.StringData(monitor.Comment),
		"notifyAt":         llx.ArrayData(notifyAt, types.Int),
		"notifyUsers":      llx.ArrayData(notifyUsers, types.String),
		"createdAt":        llx.TimeData(monitor.CreatedOn),
	}

	if monitor.SuspendAt != nil {
		args["suspendAt"] = llx.IntData(int64(*monitor.SuspendAt))
	} else {
		args["suspendAt"] = llx.NilData
	}
	if monitor.SuspendImmediateAt != nil {
		args["suspendImmediateAt"] = llx.IntData(int64(*monitor.SuspendImmediateAt))
	} else {
		args["suspendImmediateAt"] = llx.NilData
	}

	r, err := CreateResource(runtime, "snowflake.resourceMonitor", args)
	if err != nil {
		return nil, err
	}
	return r.(*mqlSnowflakeResourceMonitor), nil
}
