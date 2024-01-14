// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"time"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/google-workspace/connection"
	"go.mondoo.com/cnquery/v10/types"
	directory "google.golang.org/api/admin/directory/v1"
)

func (g *mqlGoogleworkspace) users() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GoogleWorkspaceConnection)
	directoryService, err := directoryService(conn, directory.AdminDirectoryUserReadonlyScope)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	users, err := directoryService.Users.List().Customer(conn.CustomerID()).Do()
	if err != nil {
		return nil, err
	}

	for {
		for i := range users.Users {
			r, err := newMqlGoogleWorkspaceUser(g.MqlRuntime, users.Users[i])
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}

		if users.NextPageToken == "" {
			break
		}

		users, err = directoryService.Users.List().Customer(conn.CustomerID()).PageToken(users.NextPageToken).Do()
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func newMqlGoogleWorkspaceUser(runtime *plugin.Runtime, entry *directory.User) (interface{}, error) {
	var lastLoginTime *time.Time
	var creationTime *time.Time

	llt, err := time.Parse(time.RFC3339, entry.LastLoginTime)
	if err == nil {
		lastLoginTime = &llt
	}

	ct, err := time.Parse(time.RFC3339, entry.CreationTime)
	if err == nil {
		creationTime = &ct
	}

	return CreateResource(runtime, "googleworkspace.user", map[string]*llx.RawData{
		"id":               llx.StringData(entry.Id),
		"familyName":       llx.StringData(entry.Name.FamilyName),
		"givenName":        llx.StringData(entry.Name.GivenName),
		"fullName":         llx.StringData(entry.Name.FullName),
		"primaryEmail":     llx.StringData(entry.PrimaryEmail),
		"recoveryEmail":    llx.StringData(entry.RecoveryEmail),
		"recoveryPhone":    llx.StringData(entry.RecoveryPhone),
		"agreedToTerms":    llx.BoolData(entry.AgreedToTerms),
		"aliases":          llx.ArrayData(convert.SliceAnyToInterface[string](entry.Aliases), types.Any),
		"suspended":        llx.BoolData(entry.Suspended),
		"suspensionReason": llx.StringData(entry.SuspensionReason),
		"archived":         llx.BoolData(entry.Archived),
		"isAdmin":          llx.BoolData(entry.IsAdmin),
		"isEnforcedIn2Sv":  llx.BoolData(entry.IsEnforcedIn2Sv),
		"isEnrolledIn2Sv":  llx.BoolData(entry.IsEnrolledIn2Sv),
		"isMailboxSetup":   llx.BoolData(entry.IsMailboxSetup),
		"lastLoginTime":    llx.TimeDataPtr(lastLoginTime),
		"creationTime":     llx.TimeDataPtr(creationTime),
	})
}

func (g *mqlGoogleworkspaceUser) id() (string, error) {
	return "googleworkspace.user/" + g.Id.Data, g.Id.Error
}

func (g *mqlGoogleworkspaceUser) usageReport() (*mqlGoogleworkspaceReportUsage, error) {
	conn := g.MqlRuntime.Connection.(*connection.GoogleWorkspaceConnection)
	reportsService, err := reportsService(conn)
	if err != nil {
		return nil, err
	}

	if g.PrimaryEmail.Error != nil {
		return nil, g.PrimaryEmail.Error
	}
	primaryEmail := g.PrimaryEmail.Data

	date := time.Now()
	report, err := reportsService.UserUsageReport.Get(primaryEmail, date.Format(time.DateOnly)).CustomerId(conn.CustomerID()).Do()
	if err != nil {
		return nil, err
	}

	i := 0
	for {
		if len(report.UsageReports) > 1 {
			return nil, errors.New("unexpected result for user usage report")
		}

		if len(report.UsageReports) == 0 {
			date = date.Add(-24 * time.Hour)
			// try fetching from a day before
			report, err = reportsService.UserUsageReport.Get(primaryEmail, date.Format(time.DateOnly)).CustomerId(conn.CustomerID()).Do()
			if err != nil {
				return nil, err
			}
			i++
			if i > 10 {
				return nil, errors.New("could not find user report within last 10 days")
			}
			continue
		}

		// if we reach here, we have exactly one report
		break
	}

	return newMqlGoogleWorkspaceUsageReport(g.MqlRuntime, report.UsageReports[0])
}

func (g *mqlGoogleworkspaceUser) tokens() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GoogleWorkspaceConnection)
	directoryService, err := directoryService(conn, directory.AdminDirectoryUserSecurityScope)
	if err != nil {
		return nil, err
	}

	if g.PrimaryEmail.Error != nil {
		return nil, g.PrimaryEmail.Error
	}
	primaryEmail := g.PrimaryEmail.Data

	tokenList, err := directoryService.Tokens.List(primaryEmail).Do()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range tokenList.Items {
		r, err := newMqlGoogleWorkspaceToken(g.MqlRuntime, tokenList.Items[i])
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func newMqlGoogleWorkspaceToken(runtime *plugin.Runtime, entry *directory.Token) (interface{}, error) {
	return CreateResource(runtime, "googleworkspace.token", map[string]*llx.RawData{
		"anonymous":   llx.BoolData(entry.Anonymous),
		"clientId":    llx.StringData(entry.ClientId),
		"displayText": llx.StringData(entry.DisplayText),
		"nativeApp":   llx.BoolData(entry.NativeApp),
		"scopes":      llx.ArrayData(convert.SliceAnyToInterface[string](entry.Scopes), types.Any),
		"userKey":     llx.StringData(entry.UserKey),
	})
}

func (g *mqlGoogleworkspaceToken) id() (string, error) {
	if g.ClientId.Error != nil {
		return "", g.ClientId.Error
	}
	clientID := g.ClientId.Data

	if g.UserKey.Error != nil {
		return "", g.UserKey.Error
	}
	userKey := g.UserKey.Data

	return "googleworkspace.token/" + userKey + "/" + clientID, nil
}
