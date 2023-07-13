package googleworkspace

import (
	"errors"
	"time"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	directory "google.golang.org/api/admin/directory/v1"
)

func (g *mqlGoogleworkspace) GetUsers() ([]interface{}, error) {
	provider, directoryService, err := directoryService(g.MotorRuntime.Motor.Provider, directory.AdminDirectoryUserReadonlyScope)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	users, err := directoryService.Users.List().Customer(provider.GetCustomerID()).Do()
	if err != nil {
		return nil, err
	}

	for {
		for i := range users.Users {
			r, err := newMqlGoogleWorkspaceUser(g.MotorRuntime, users.Users[i])
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}

		if users.NextPageToken == "" {
			break
		}

		users, err = directoryService.Users.List().Customer(provider.GetCustomerID()).PageToken(users.NextPageToken).Do()
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func newMqlGoogleWorkspaceUser(runtime *resources.Runtime, entry *directory.User) (interface{}, error) {
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

	return runtime.CreateResource("googleworkspace.user",
		"id", entry.Id,
		"familyName", entry.Name.FamilyName,
		"givenName", entry.Name.GivenName,
		"fullName", entry.Name.FullName,
		"primaryEmail", entry.PrimaryEmail,
		"recoveryEmail", entry.RecoveryEmail,
		"recoveryPhone", entry.RecoveryPhone,
		"agreedToTerms", entry.AgreedToTerms,
		"aliases", core.StrSliceToInterface(entry.Aliases),
		"suspended", entry.Suspended,
		"suspensionReason", entry.SuspensionReason,
		"archived", entry.Archived,
		"isAdmin", entry.IsAdmin,
		"isEnforcedIn2Sv", entry.IsEnforcedIn2Sv,
		"isEnrolledIn2Sv", entry.IsEnrolledIn2Sv,
		"isMailboxSetup", entry.IsMailboxSetup,
		"lastLoginTime", lastLoginTime,
		"creationTime", creationTime,
	)
}

func (g *mqlGoogleworkspaceUser) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "googleworkspace.user/" + id, nil
}

func (g *mqlGoogleworkspaceUser) GetUsageReport() (interface{}, error) {
	provider, reportsService, err := reportsService(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	primaryEmail, err := g.PrimaryEmail()
	if err != nil {
		return nil, err
	}

	date := time.Now()
	report, err := reportsService.UserUsageReport.Get(primaryEmail, date.Format(ISO8601)).CustomerId(provider.GetCustomerID()).Do()
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
			report, err = reportsService.UserUsageReport.Get(primaryEmail, date.Format(ISO8601)).CustomerId(provider.GetCustomerID()).Do()
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

	return newMqlGoogleWorkspaceUsageReport(g.MotorRuntime, report.UsageReports[0])
}

func (g *mqlGoogleworkspaceUser) GetTokens() ([]interface{}, error) {
	_, directoryService, err := directoryService(g.MotorRuntime.Motor.Provider, directory.AdminDirectoryUserSecurityScope)
	if err != nil {
		return nil, err
	}

	primaryEmail, err := g.PrimaryEmail()
	if err != nil {
		return nil, err
	}

	tokenList, err := directoryService.Tokens.List(primaryEmail).Do()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range tokenList.Items {
		r, err := newMqlGoogleWorkspaceToken(g.MotorRuntime, tokenList.Items[i])
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func newMqlGoogleWorkspaceToken(runtime *resources.Runtime, entry *directory.Token) (interface{}, error) {
	return runtime.CreateResource("googleworkspace.token",
		"anonymous", entry.Anonymous,
		"clientId", entry.ClientId,
		"displayText", entry.DisplayText,
		"nativeApp", entry.NativeApp,
		"scopes", core.StrSliceToInterface(entry.Scopes),
		"userKey", entry.UserKey,
	)
}

func (g *mqlGoogleworkspaceToken) id() (string, error) {
	clientID, err := g.ClientId()
	if err != nil {
		return "", err
	}

	userKey, err := g.UserKey()
	if err != nil {
		return "", err
	}

	return "googleworkspace.token/" + userKey + "/" + clientID, nil
}
