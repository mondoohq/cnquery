// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/cnquery/v11/providers/google-workspace/connection"
	directory "google.golang.org/api/admin/directory/v1"
	reports "google.golang.org/api/admin/reports/v1"
	"google.golang.org/api/calendar/v3"
	cloudidentity "google.golang.org/api/cloudidentity/v1"
	"google.golang.org/api/groupssettings/v1"
	"google.golang.org/api/option"
)

func (r *mqlGoogleworkspace) id() (string, error) {
	return "google-workspace", nil
}

func reportsService(conn *connection.GoogleWorkspaceConnection) (*reports.Service, error) {
	client, err := conn.Client(reports.AdminReportsAuditReadonlyScope, reports.AdminReportsUsageReadonlyScope)
	if err != nil {
		return nil, err
	}

	service, err := reports.NewService(context.Background(), option.WithHTTPClient(client))
	return service, err
}

func directoryService(conn *connection.GoogleWorkspaceConnection, scopes ...string) (*directory.Service, error) {
	client, err := conn.Client(scopes...)
	if err != nil {
		return nil, err
	}

	directoryService, err := directory.NewService(context.Background(), option.WithHTTPClient(client))
	return directoryService, err
}

func calendarService(conn *connection.GoogleWorkspaceConnection, scopes ...string) (*calendar.Service, error) {
	client, err := conn.Client(scopes...)
	if err != nil {
		return nil, err
	}

	calendarsService, err := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	return calendarsService, err
}

func cloudIdentityService(conn *connection.GoogleWorkspaceConnection, scopes ...string) (*cloudidentity.Service, error) {
	client, err := conn.Client(scopes...)
	if err != nil {
		return nil, err
	}

	cloudIdentityService, err := cloudidentity.NewService(context.Background(), option.WithHTTPClient(client))
	return cloudIdentityService, err
}

func groupSettingsService(conn *connection.GoogleWorkspaceConnection, scopes ...string) (*groupssettings.Service, error) {
	client, err := conn.Client(scopes...)
	if err != nil {
		return nil, err
	}

	groupssettingsService, err := groupssettings.NewService(context.Background(), option.WithHTTPClient(client))
	return groupssettingsService, err
}
