// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"github.com/stackitcloud/stackit-sdk-go/services/serverupdate"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// ------------------------- server updates -------------------------

func (r *mqlStackitServer) updates() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.ServerUpdate()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListUpdatesExecute(bgctx(), c.ProjectID(), r.Id.Data, c.Region())
	if err != nil {
		if isAccessDenied(err) || isNotFound(err) {
			// A 404 means the Server Update service is not enabled for this
			// server, a legitimate "no updates" state rather than an error.
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildServerUpdate(r.MqlRuntime, r.Id.Data, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func buildServerUpdate(runtime *plugin.Runtime, serverID string, u *serverupdate.Update) (plugin.Resource, error) {
	args := map[string]*llx.RawData{
		"id":               llx.IntData(u.GetId()),
		"serverId":         llx.StringData(serverID),
		"status":           llx.StringData(u.GetStatus()),
		"startDate":        llx.TimeDataPtr(parseRFC3339(u.GetStartDate())),
		"endDate":          llx.TimeDataPtr(parseRFC3339(u.GetEndDate())),
		"installedUpdates": llx.IntData(u.GetInstalledUpdates()),
		"failedUpdates":    llx.IntData(u.GetFailedUpdates()),
		"failReason":       llx.StringData(u.GetFailReason()),
	}
	return CreateResource(runtime, "stackit.server.update", args)
}

func (r *mqlStackitServerUpdate) id() (string, error) {
	return "stackit.server.update/" + r.ServerId.Data + "/" + strconv.FormatInt(r.Id.Data, 10), nil
}

func (r *mqlStackitServerUpdate) server() (*mqlStackitServer, error) {
	return serverRef(r.MqlRuntime, r.ServerId.Data, &r.Server)
}

// ------------------------- server update schedules -------------------------

func (r *mqlStackitServer) updateSchedules() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.ServerUpdate()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListUpdateSchedulesExecute(bgctx(), c.ProjectID(), r.Id.Data, c.Region())
	if err != nil {
		if isAccessDenied(err) || isNotFound(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildUpdateSchedule(r.MqlRuntime, r.Id.Data, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func buildUpdateSchedule(runtime *plugin.Runtime, serverID string, s *serverupdate.UpdateSchedule) (plugin.Resource, error) {
	args := map[string]*llx.RawData{
		"id":                llx.IntData(s.GetId()),
		"serverId":          llx.StringData(serverID),
		"name":              llx.StringData(s.GetName()),
		"enabled":           llx.BoolData(s.GetEnabled()),
		"rrule":             llx.StringData(s.GetRrule()),
		"maintenanceWindow": llx.IntData(s.GetMaintenanceWindow()),
	}
	return CreateResource(runtime, "stackit.server.updateSchedule", args)
}

func (r *mqlStackitServerUpdateSchedule) id() (string, error) {
	return "stackit.server.updateSchedule/" + r.ServerId.Data + "/" + strconv.FormatInt(r.Id.Data, 10), nil
}

func (r *mqlStackitServerUpdateSchedule) server() (*mqlStackitServer, error) {
	return serverRef(r.MqlRuntime, r.ServerId.Data, &r.Server)
}
