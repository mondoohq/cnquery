package googleworkspace

import (
	"errors"
	"strconv"
	"time"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
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

// ISO8601 is a date format required by Google Workspace Reports API
const ISO8601 = "2006-01-02" // yyyy-mm-dd

func (g *mqlGoogleworkspaceReportApps) id() (string, error) {
	return "googleworkspace.report.apps", nil
}

func (g *mqlGoogleworkspaceReportApps) GetDrive() ([]interface{}, error) {
	provider, reportsService, err := reportsService(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	activities, err := reportsService.Activities.List("all", "drive").CustomerId(provider.GetCustomerID()).Do()
	if err != nil {
		return nil, err
	}

	for {
		for i := range activities.Items {
			r, err := newMqlGoogleWorkspaceReportActivity(g.MotorRuntime, activities.Items[i])
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}

		if activities.NextPageToken == "" {
			break
		}

		activities, err = reportsService.Activities.List("all", "drive").CustomerId(provider.GetCustomerID()).
			PageToken(activities.NextPageToken).Do()
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (g *mqlGoogleworkspaceReportActivity) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "googleworkspace.report.activity/" + strconv.FormatInt(id, 10), nil
}

func newMqlGoogleWorkspaceReportActivity(runtime *resources.Runtime, entry *reports.Activity) (interface{}, error) {
	actor, err := core.JsonToDict(entry.Actor)
	if err != nil {
		return nil, err
	}
	events, err := core.JsonToDictSlice(entry.Events)
	if err != nil {
		return nil, err
	}

	return runtime.CreateResource("googleworkspace.report.activity",
		"id", entry.Id.UniqueQualifier,
		"ipAddress", entry.IpAddress,
		"ownerDomain", entry.OwnerDomain,
		"actor", actor,
		"events", events,
	)
}

func (g *mqlGoogleworkspaceReportUsers) id() (string, error) {
	return "googleworkspace.report.users", nil
}

func (g *mqlGoogleworkspaceReportUsers) GetList() ([]interface{}, error) {
	provider, reportsService, err := reportsService(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	date := time.Now()
	usageReports, err := reportsService.UserUsageReport.Get("all", date.Format(ISO8601)).CustomerId(provider.GetCustomerID()).Do()
	if err != nil {
		return nil, err
	}
	for {
		if len(usageReports.UsageReports) == 0 {
			date = date.Add(-24 * time.Hour)
			// try fetching from a day before
			usageReports, err = reportsService.UserUsageReport.Get("all", date.Format(ISO8601)).CustomerId(provider.GetCustomerID()).Do()
			if err != nil {
				return nil, err
			}
			continue
		}

		for i := range usageReports.UsageReports {
			r, err := newMqlGoogleWorkspaceUsageReport(g.MotorRuntime, usageReports.UsageReports[i])
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}

		if usageReports.NextPageToken == "" {
			break
		}

		usageReports, err = reportsService.UserUsageReport.Get("all", date.Format(ISO8601)).CustomerId(provider.GetCustomerID()).
			PageToken(usageReports.NextPageToken).Do()
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func newMqlGoogleWorkspaceUsageReport(runtime *resources.Runtime, entry *reports.UsageReport) (interface{}, error) {
	var date *time.Time
	parsedDate, err := time.Parse(ISO8601, entry.Date)
	if err == nil {
		date = &parsedDate
	}

	parameters, err := core.JsonToDictSlice(entry.Parameters)
	if err != nil {
		return nil, err
	}

	r := parseUserReports(entry.Parameters)

	accountUsage, err := core.JsonToDict(r.Account)
	if err != nil {
		return nil, err
	}

	securityUsage, err := core.JsonToDict(r.Security)
	if err != nil {
		return nil, err
	}

	appUsage, err := core.JsonToDict(r.AppUsage)
	if err != nil {
		return nil, err
	}

	return runtime.CreateResource("googleworkspace.report.usage",
		"customerId", entry.Entity.CustomerId,
		"entityId", entry.Entity.EntityId,
		"profileId", entry.Entity.ProfileId,
		"type", entry.Entity.Type,
		"userEmail", entry.Entity.UserEmail,
		"date", date,
		"parameters", parameters,
		"account", accountUsage,
		"security", securityUsage,
		"appUsage", appUsage,
	)
}

func (g *mqlGoogleworkspaceReportUsage) id() (string, error) {
	customerId, err := g.CustomerId()
	if err != nil {
		return "", err
	}
	profileId, err := g.ProfileId()
	if err != nil {
		return "", err
	}
	date, err := g.Date()
	if err != nil {
		return "", err
	}

	return "googleworkspace.report.usage/" + customerId + "/" + profileId + "/" + date.Format(ISO8601), nil
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

func (g *mqlGoogleworkspaceReportUsage) GetAccount() (interface{}, error) {
	// is auto-computed during creation time
	return nil, errors.New("not implemented")
}

func (g *mqlGoogleworkspaceReportUsage) GetSecurity() (interface{}, error) {
	// is auto-computed during creation time
	return nil, errors.New("not implemented")
}

func (g *mqlGoogleworkspaceReportUsage) GetAppUsage() (interface{}, error) {
	// is auto-computed during creation time
	return nil, errors.New("not implemented")
}
