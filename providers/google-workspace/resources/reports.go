// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/google-workspace/connection"
	"go.mondoo.com/cnquery/v11/types"
	reports "google.golang.org/api/admin/reports/v1"
)

// https://developers.google.com/admin-sdk/reports/reference/rest/v1/activities/list#ApplicationName
const (
	appAccessTransparency = "access_transparency"
	appAdmin              = "admin"
	appCalendar           = "calendar"
	appChat               = "chat"
	appDrive              = "drive"
	appGcp                = "gcp"
	appGplus              = "gplus"
	appGroups             = "groups"
	appGroupsEnterprise   = "groups_enterprise"
	appJamboard           = "jamboard"
	appLogin              = "login"
	appMeet               = "meet"
	appMobile             = "mobile"
	appRules              = "rules"
	appSaml               = "saml"
	appToken              = "token"
	appUserAccounts       = "user_accounts"
	appContextAwareAccess = "context_aware_access"
	appChrome             = "chrome"
	appDataStudio         = "data_studio"
	appKeep               = "keep"
)

func (g *mqlGoogleworkspaceReportApps) id() (string, error) {
	return "googleworkspace.report.apps", nil
}

func (g *mqlGoogleworkspaceReportApps) drive() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GoogleWorkspaceConnection)
	reportsService, err := reportsService(conn)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	activities, err := reportsService.Activities.List("all", "drive").CustomerId(conn.CustomerID()).Do()
	if err != nil {
		return nil, err
	}

	for {
		for i := range activities.Items {
			r, err := newMqlGoogleWorkspaceReportActivity(g.MqlRuntime, activities.Items[i])
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}

		if activities.NextPageToken == "" {
			break
		}

		activities, err = reportsService.Activities.List("all", "drive").CustomerId(conn.CustomerID()).
			PageToken(activities.NextPageToken).Do()
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (g *mqlGoogleworkspaceReportApps) admin() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GoogleWorkspaceConnection)
	reportsService, err := reportsService(conn)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	activities, err := reportsService.Activities.List("all", "admin").CustomerId(conn.CustomerID()).Do()
	if err != nil {
		return nil, err
	}

	for {
		for i := range activities.Items {
			r, err := newMqlGoogleWorkspaceReportActivity(g.MqlRuntime, activities.Items[i])
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}

		if activities.NextPageToken == "" {
			break
		}

		activities, err = reportsService.Activities.List("all", "admin").CustomerId(conn.CustomerID()).
			PageToken(activities.NextPageToken).Do()
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (g *mqlGoogleworkspaceReportActivity) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "googleworkspace.report.activity/" + strconv.FormatInt(id, 10), nil
}

func newMqlGoogleWorkspaceReportActivity(runtime *plugin.Runtime, entry *reports.Activity) (interface{}, error) {
	actor, err := convert.JsonToDict(entry.Actor)
	if err != nil {
		return nil, err
	}
	events, err := convert.JsonToDictSlice(entry.Events)
	if err != nil {
		return nil, err
	}

	return CreateResource(runtime, "googleworkspace.report.activity", map[string]*llx.RawData{
		"id":          llx.IntData(entry.Id.UniqueQualifier),
		"ipAddress":   llx.StringData(entry.IpAddress),
		"ownerDomain": llx.StringData(entry.OwnerDomain),
		"actor":       llx.MapData(actor, types.Any),
		"events":      llx.ArrayData(events, types.Any),
	})
}

func (g *mqlGoogleworkspaceReportUsers) id() (string, error) {
	return "googleworkspace.report.users", nil
}

func (g *mqlGoogleworkspaceReportUsers) list() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GoogleWorkspaceConnection)
	reportsService, err := reportsService(conn)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	date := time.Now()
	expectedErr := "googleapi: Error 400: Data for dates later than"

	usageReports, err := fetchReportUsage(g.MqlRuntime, reportsService, conn.CustomerID(), date.Format(time.DateOnly))
	// we expect this error if there is no data for the current day, so we continue
	if err != nil && !strings.HasPrefix(err.Error(), expectedErr) {
		return nil, err
	}

	if len(usageReports) == 0 {
		// try and fetch usage for each of the past 7 days
		attempts := 7
		for attempts > 0 {
			date = date.Add(-24 * time.Hour)
			reports, err := fetchReportUsage(g.MqlRuntime, reportsService, conn.CustomerID(), date.Format(time.DateOnly))
			// we expect this error if there is no data for the current day, so we continue
			if err != nil && !strings.HasPrefix(err.Error(), expectedErr) {
				return nil, err
			}
			if len(reports) > 0 {
				res = append(res, reports...)
				break
			}
			attempts--
		}
	}

	return res, nil
}

func fetchReportUsage(runtime *plugin.Runtime, service *reports.Service, customerId, date string) ([]interface{}, error) {
	res := []interface{}{}

	usageReports, err := service.UserUsageReport.Get("all", date).CustomerId(customerId).Do()
	if err != nil {
		return nil, err
	}
	for _, u := range usageReports.UsageReports {
		r, err := newMqlGoogleWorkspaceUsageReport(runtime, u)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	for usageReports != nil && usageReports.NextPageToken != "" {
		usageReports, err := service.UserUsageReport.Get("all", date).CustomerId(customerId).PageToken(usageReports.NextPageToken).Do()
		if err != nil {
			return nil, err
		}
		for _, u := range usageReports.UsageReports {
			r, err := newMqlGoogleWorkspaceUsageReport(runtime, u)
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}
	}
	return res, nil
}

func newMqlGoogleWorkspaceUsageReport(runtime *plugin.Runtime, entry *reports.UsageReport) (*mqlGoogleworkspaceReportUsage, error) {
	var date *time.Time
	parsedDate, err := time.Parse(time.DateOnly, entry.Date)
	if err == nil {
		date = &parsedDate
	}

	parameters, err := convert.JsonToDictSlice(entry.Parameters)
	if err != nil {
		return nil, err
	}

	r := parseUserReports(entry.Parameters)

	accountUsage, err := convert.JsonToDict(r.Account)
	if err != nil {
		return nil, err
	}

	securityUsage, err := convert.JsonToDict(r.Security)
	if err != nil {
		return nil, err
	}

	appUsage, err := convert.JsonToDict(r.AppUsage)
	if err != nil {
		return nil, err
	}

	report, err := CreateResource(runtime, "googleworkspace.report.usage", map[string]*llx.RawData{
		"customerId": llx.StringData(entry.Entity.CustomerId),
		"entityId":   llx.StringData(entry.Entity.EntityId),
		"profileId":  llx.StringData(entry.Entity.ProfileId),
		"type":       llx.StringData(entry.Entity.Type),
		"userEmail":  llx.StringData(entry.Entity.UserEmail),
		"date":       llx.TimeDataPtr(date),
		"parameters": llx.ArrayData(parameters, types.Any),
		"account":    llx.MapData(accountUsage, types.Any),
		"security":   llx.MapData(securityUsage, types.Any),
		"appUsage":   llx.MapData(appUsage, types.Any),
	})
	return report.(*mqlGoogleworkspaceReportUsage), err
}

func (g *mqlGoogleworkspaceReportUsage) id() (string, error) {
	if g.CustomerId.Error != nil {
		return "", g.CustomerId.Error
	}
	customerId := g.CustomerId.Data
	if g.ProfileId.Error != nil {
		return "", g.ProfileId.Error
	}
	profileId := g.ProfileId.Data
	if g.Date.Error != nil {
		return "", g.Date.Error
	}
	date := g.Date.Data

	return "googleworkspace.report.usage/" + customerId + "/" + profileId + "/" + date.Format(time.DateOnly), nil
}

type userReport struct {
	Account  userReportAccount
	Security userReportSecurity
	AppUsage userReportAppUsage
}

type userReportAccount struct {
	IsDisabled                    bool   `json:"isDisabled"`
	IsSuperAdmin                  bool   `json:"isSuperAdmin"`
	IsS2svEnrolled                bool   `json:"isS2SvEnrolled"`
	Is2svEnforced                 bool   `json:"is2SvEnforced"`
	PasswordLengthCompliance      string `json:"passwordLengthCompliance"`
	PasswordStrength              string `json:"passwordStrength"`
	IsLessSecureAppsAccessAllowed bool   `json:"isLessSecureAppsAccessAllowed"`
	GmailUsedQuotaInMb            int64  `json:"gmailUsedQuotaInMb"`
	DriveUsedQuotaInMb            int64  `json:"driveUsedQuotaInMb"`
	UsedQuotaInMb                 int64  `json:"usedQuotaInMb"`
	AdminSetName                  string `json:"adminSetName"`
}

type userReportSecurity struct {
	NumAuthorizedApps             int64  `json:"numAuthorizedApps"`
	IsS2svEnrolled                bool   `json:"isS2SvEnrolled"`
	Is2svEnforced                 bool   `json:"is2SvEnforced"`
	PasswordLengthCompliance      string `json:"passwordLengthCompliance"`
	PasswordStrength              string `json:"passwordStrength"`
	IsDisabled                    bool   `json:"isDisabled"`
	IsSuperAdmin                  bool   `json:"isSuperAdmin"`
	NumSecurityKeys               int64  `json:"numSecurityKeys"`
	IsLessSecureAppsAccessAllowed bool   `json:"isLessSecureAppsAccessAllowed"`
}

type userReportAppUsage struct {
	GmailUsedQuotaInMb       int64      `json:"gmailUsedQuotaInMb"`
	DriveUsedQuotaInMb       int64      `json:"driveUsedQuotaInMb"`
	GPlusPhotosUsedQuotaInMb int64      `json:"gPlusPhotosUsedQuotaInMb"`
	UsedQuotaInMb            int64      `json:"usedQuotaInMb"`
	NumEmailsExchanged       int64      `json:"numEmailsExchanged"`
	NumEmailSent             int64      `json:"numEmailSent"`
	NumEmailsReceived        int64      `json:"numEmailsReceived"`
	LastImapTime             *time.Time `json:"lastImapTime"`
	LastWebmailTime          *time.Time `json:"lastWebmailTime"`
	NumOwnedItemsEdited      int64      `json:"numOwnedItemsEdited"`
	NumOwnedItemsViewed      int64      `json:"numOwnedItemsViewed"`
	DriveLastActiveUsageTime *time.Time `json:"driveLastActiveUsageTime"`
}

func parseUserReports(params []*reports.UsageReportParameters) *userReport {
	r := &userReport{}

	for i := range params {
		param := params[i]
		switch param.Name {
		// account
		case "accounts:is_disabled":
			r.Account.IsDisabled = param.BoolValue
			r.Security.IsDisabled = param.BoolValue
		case "accounts:is_super_admin":
			r.Account.IsSuperAdmin = param.BoolValue
			r.Security.IsSuperAdmin = param.BoolValue
		case "accounts:is_2sv_enrolled":
			r.Account.IsS2svEnrolled = param.BoolValue
			r.Security.IsS2svEnrolled = param.BoolValue
		case "accounts:is_2sv_enforced":
			r.Account.Is2svEnforced = param.BoolValue
			r.Security.Is2svEnforced = param.BoolValue
		case "accounts:password_length_compliance":
			r.Account.PasswordLengthCompliance = param.StringValue
			r.Security.PasswordLengthCompliance = param.StringValue
		case "accounts:password_strength":
			r.Account.PasswordStrength = param.StringValue
			r.Security.PasswordStrength = param.StringValue
		case "accounts:is_less_secure_apps_access_allowed":
			r.Account.IsLessSecureAppsAccessAllowed = param.BoolValue
			r.Security.IsLessSecureAppsAccessAllowed = param.BoolValue
		case "accounts:admin_set_name":
			r.Account.AdminSetName = param.StringValue
			// security
		case "accounts:num_authorized_apps":
			r.Security.NumAuthorizedApps = param.IntValue
		case "accounts:num_security_keys":
			r.Security.NumSecurityKeys = param.IntValue
			// usage
		case "accounts:gmail_used_quota_in_mb":
			r.Account.GmailUsedQuotaInMb = param.IntValue
			r.AppUsage.GmailUsedQuotaInMb = param.IntValue
		case "accounts:drive_used_quota_in_mb":
			r.Account.DriveUsedQuotaInMb = param.IntValue
			r.AppUsage.DriveUsedQuotaInMb = param.IntValue
		case "gplus_photos_used_quota_in_mb":
			r.AppUsage.GPlusPhotosUsedQuotaInMb = param.IntValue
		case "accounts:used_quota_in_mb":
			r.Account.UsedQuotaInMb = param.IntValue
			r.AppUsage.UsedQuotaInMb = param.IntValue
		case "gmail:num_emails_exchanged":
			r.AppUsage.NumEmailsExchanged = param.IntValue
		case "gmail:num_emails_sent":
			r.AppUsage.NumEmailSent = param.IntValue
		case "gmail:num_emails_received":
			r.AppUsage.NumEmailsReceived = param.IntValue
		case "gmail:last_imap_time":
			var datetime *time.Time
			parseDateTime, err := time.Parse(time.RFC3339, param.DatetimeValue)
			if err == nil {
				datetime = &parseDateTime
			}
			r.AppUsage.LastImapTime = datetime
		case "gmail:last_webmail_time":
			var datetime *time.Time
			parseDateTime, err := time.Parse(time.RFC3339, param.DatetimeValue)
			if err == nil {
				datetime = &parseDateTime
			}
			r.AppUsage.LastWebmailTime = datetime
		case "docs:num_owned_items_edited":
			r.AppUsage.NumOwnedItemsEdited = param.IntValue
		case "docs:num_owned_items_viewed":
			r.AppUsage.NumOwnedItemsViewed = param.IntValue
		case "drive:last_active_usage_time":
			var datetime *time.Time
			parseDateTime, err := time.Parse(time.RFC3339, param.DatetimeValue)
			if err == nil {
				datetime = &parseDateTime
			}
			r.AppUsage.DriveLastActiveUsageTime = datetime
		}
	}

	return r
}

func (g *mqlGoogleworkspaceReportUsage) account() (interface{}, error) {
	// is auto-computed during creation time
	return nil, errors.New("not implemented")
}

func (g *mqlGoogleworkspaceReportUsage) security() (interface{}, error) {
	// is auto-computed during creation time
	return nil, errors.New("not implemented")
}

func (g *mqlGoogleworkspaceReportUsage) appUsage() (interface{}, error) {
	// is auto-computed during creation time
	return nil, errors.New("not implemented")
}
