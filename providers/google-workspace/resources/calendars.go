// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers/google-workspace/connection"
	"google.golang.org/api/calendar/v3"
)

func (g *mqlGoogleworkspace) calendars() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GoogleWorkspaceConnection)
	calendarService, err := calendarService(conn, calendar.CalendarReadonlyScope, calendar.CalendarSettingsReadonlyScope)
	if err != nil {
		return nil, err
	}
	calendars, err := calendarService.CalendarList.List().Do()
	if err != nil {
		return nil, err
	}
	res := make([]interface{}, 0, len(calendars.Items))
	for _, c := range calendars.Items {
		r, err := CreateResource(g.MqlRuntime, "googleworkspace.calendar", map[string]*llx.RawData{
			"__id":            llx.StringData(c.Id),
			"summary":         llx.StringData(c.Summary),
			"summaryOverride": llx.StringData(c.SummaryOverride),
			"primary":         llx.BoolData(c.Primary),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (g *mqlGoogleworkspaceCalendar) acl() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GoogleWorkspaceConnection)
	calendarService, err := calendarService(conn, calendar.CalendarScope)
	if err != nil {
		return nil, err
	}
	acls, err := calendarService.Acl.List(g.__id).Do()
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, 0, len(acls.Items))
	for _, a := range acls.Items {
		scope, err := CreateResource(g.MqlRuntime, "googleworkspace.calendar.aclRule.scope", map[string]*llx.RawData{
			"__id":  llx.StringData(a.Id + a.Scope.Type + a.Scope.Value),
			"type":  llx.StringData(a.Scope.Type),
			"value": llx.StringData(a.Scope.Value),
		})
		if err != nil {
			return nil, err
		}

		r, err := CreateResource(g.MqlRuntime, "googleworkspace.calendar.aclRule", map[string]*llx.RawData{
			"__id":  llx.StringData(a.Id),
			"role":  llx.StringData(a.Role),
			"scope": llx.ResourceData(scope, "googleworkspace.calendar.aclRule.scope"),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}
