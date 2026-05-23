// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"go.mondoo.com/mql/v13/providers/google-workspace/connection"
	directory "google.golang.org/api/admin/directory/v1"
	reports "google.golang.org/api/admin/reports/v1"
	"google.golang.org/api/calendar/v3"
	cloudidentity "google.golang.org/api/cloudidentity/v1"
	"google.golang.org/api/groupssettings/v1"
	"google.golang.org/api/option"
)

type mqlGoogleworkspaceInternal struct {
	usersByEmailOnce sync.Once
	usersByEmail     map[string]*mqlGoogleworkspaceUser
	usersByEmailErr  error
}

func (g *mqlGoogleworkspace) userByEmail(email string) (*mqlGoogleworkspaceUser, error) {
	g.usersByEmailOnce.Do(func() {
		if g.Users.Error != nil {
			g.usersByEmailErr = g.Users.Error
			return
		}
		m := make(map[string]*mqlGoogleworkspaceUser, len(g.Users.Data))
		for _, u := range g.Users.Data {
			user := u.(*mqlGoogleworkspaceUser)
			if user.PrimaryEmail.Error != nil {
				g.usersByEmailErr = user.PrimaryEmail.Error
				return
			}
			m[user.PrimaryEmail.Data] = user
		}
		g.usersByEmail = m
	})
	if g.usersByEmailErr != nil {
		return nil, g.usersByEmailErr
	}
	return g.usersByEmail[email], nil
}

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
