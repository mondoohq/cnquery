// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/oracle/oci-go-sdk/v65/identitydomains"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

// ociScimPageSize is how many SCIM resources to request per call.
//
// Identity Domains paginates with startIndex/count rather than a page token,
// so the loop has to drive the offset itself. 200 is well inside the service
// maximum and keeps the request count low on domains with many users.
const ociScimPageSize = 200

func (o *mqlOciIdentity) domains() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	ctx := context.Background()

	client, err := conn.IdentityClient()
	if err != nil {
		return nil, err
	}

	// Domains are an identity-plane resource and live in the home region, so
	// this is a single listing rather than a region fan-out. They can be
	// created in any compartment, though, so the compartment tree still has to
	// be walked.
	compartments, err := conn.GetCompartments(ctx)
	if err != nil {
		return nil, err
	}

	res := []any{}
	for i := range compartments {
		compartmentID := stringValue(compartments[i].Id)
		if compartmentID == "" {
			continue
		}

		domains := []identity.DomainSummary{}
		var page *string
		for {
			response, err := client.ListDomains(ctx, identity.ListDomainsRequest{
				CompartmentId: common.String(compartmentID),
				Page:          page,
			})
			if err != nil {
				// A compartment the caller cannot read is expected; anything
				// else is a real fault, matching the compartment fan-out.
				if ociCompartmentInaccessible(err) {
					break
				}
				return nil, err
			}

			domains = append(domains, response.Items...)

			if response.OpcNextPage == nil {
				break
			}
			page = response.OpcNextPage
		}

		for j := range domains {
			d := domains[j]

			replicaRegions := make([]string, 0, len(d.ReplicaRegions))
			for _, r := range d.ReplicaRegions {
				replicaRegions = append(replicaRegions, stringValue(r.Region))
			}

			mqlDomain, err := CreateResource(o.MqlRuntime, "oci.identity.domain", map[string]*llx.RawData{
				"id":              llx.StringDataPtr(d.Id),
				"name":            llx.StringDataPtr(d.DisplayName),
				"description":     llx.StringDataPtr(d.Description),
				"type":            llx.StringData(string(d.Type)),
				"licenseType":     llx.StringDataPtr(d.LicenseType),
				"homeRegion":      llx.StringDataPtr(d.HomeRegion),
				"replicaRegions":  llx.ArrayData(stringsToAny(replicaRegions), types.String),
				"isHiddenOnLogin": llx.BoolData(boolValue(d.IsHiddenOnLogin)),
				"state":           llx.StringData(string(d.LifecycleState)),
				"created":         sdkTimeData(d.TimeCreated),
				"freeformTags":    llx.MapData(strMapToAny(d.FreeformTags), types.String),
				"definedTags":     llx.MapData(definedTagsToAny(d.DefinedTags), types.Any),
			})
			if err != nil {
				return nil, err
			}
			mqlDomainTyped := mqlDomain.(*mqlOciIdentityDomain)
			mqlDomainTyped.cacheCompartmentId = stringValue(d.CompartmentId)
			mqlDomainTyped.cacheUrl = stringValue(d.Url)
			res = append(res, mqlDomainTyped)
		}
	}

	return res, nil
}

type mqlOciIdentityDomainInternal struct {
	cacheCompartmentId string
	// The domain's own SCIM endpoint. Every sub-collection is served from
	// here rather than from a regional endpoint.
	cacheUrl string
}

func (o *mqlOciIdentityDomain) id() (string, error) {
	return "oci.identity.domain/" + o.Id.Data, nil
}

func (o *mqlOciIdentityDomain) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentId, &o.Compartment)
}

func (o *mqlOciIdentityDomain) domainClient() (*identitydomains.IdentityDomainsClient, error) {
	if o.cacheUrl == "" {
		return nil, errors.New("identity domain has no endpoint url: " + o.Id.Data)
	}
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	return conn.IdentityDomainsClient(o.cacheUrl)
}

func (o *mqlOciIdentityDomain) users() ([]any, error) {
	client, err := o.domainClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	users := []identitydomains.User{}
	startIndex := 1
	for {
		response, err := client.ListUsers(ctx, identitydomains.ListUsersRequest{
			StartIndex: common.Int(startIndex),
			Count:      common.Int(ociScimPageSize),
		})
		if err != nil {
			return nil, err
		}

		users = append(users, response.Users.Resources...)

		next, more := ociScimNextIndex(startIndex, len(response.Users.Resources), response.Users.TotalResults)
		if !more {
			break
		}
		startIndex = next
	}

	res := make([]any, 0, len(users))
	for i := range users {
		user := users[i]

		primaryEmail, primaryVerified := ociPrimaryEmail(user.Emails)
		emails, err := convert.JsonToDictSlice(user.Emails)
		if err != nil {
			return nil, err
		}
		groups, err := convert.JsonToDictSlice(user.Groups)
		if err != nil {
			return nil, err
		}

		var (
			mfaStatus, preferredFactor, mfaEnabledOn string
			isLocked                                 bool
			loginAttempts                            int64
			lastSuccessful, prevSuccessful, lastFail string
			isFederated                              bool
		)

		if mfa := user.UrnIetfParamsScimSchemasOracleIdcsExtensionMfaUser; mfa != nil {
			mfaStatus = string(mfa.MfaStatus)
			preferredFactor = string(mfa.PreferredAuthenticationFactor)
			mfaEnabledOn = stringValue(mfa.MfaEnabledOn)
		}

		if state := user.UrnIetfParamsScimSchemasOracleIdcsExtensionUserStateUser; state != nil {
			loginAttempts = int64(intValue(state.LoginAttempts))
			lastSuccessful = stringValue(state.LastSuccessfulLoginDate)
			prevSuccessful = stringValue(state.PreviousSuccessfulLoginDate)
			lastFail = stringValue(state.LastFailedLoginDate)
			if state.Locked != nil {
				isLocked = boolValue(state.Locked.On)
			}
		}

		if ext := user.UrnIetfParamsScimSchemasOracleIdcsExtensionUserUser; ext != nil {
			isFederated = boolValue(ext.IsFederatedUser)
		}

		capabilities := map[string]any{}
		if caps := user.UrnIetfParamsScimSchemasOracleIdcsExtensionCapabilitiesUser; caps != nil {
			capabilities["canUseApiKeys"] = boolValue(caps.CanUseApiKeys)
			capabilities["canUseAuthTokens"] = boolValue(caps.CanUseAuthTokens)
			capabilities["canUseConsolePassword"] = boolValue(caps.CanUseConsolePassword)
			capabilities["canUseCustomerSecretKeys"] = boolValue(caps.CanUseCustomerSecretKeys)
			capabilities["canUseSmtpCredentials"] = boolValue(caps.CanUseSmtpCredentials)
			capabilities["canUseDbCredentials"] = boolValue(caps.CanUseDbCredentials)
			capabilities["canUseConsole"] = boolValue(caps.CanUseConsole)
		}

		mqlUser, err := CreateResource(o.MqlRuntime, "oci.identity.domain.user", map[string]*llx.RawData{
			"__id":                          llx.StringData(o.Id.Data + "/user/" + stringValue(user.Id)),
			"id":                            llx.StringDataPtr(user.Id),
			"ocid":                          llx.StringDataPtr(user.Ocid),
			"userName":                      llx.StringDataPtr(user.UserName),
			"displayName":                   llx.StringDataPtr(user.DisplayName),
			"description":                   llx.StringDataPtr(user.Description),
			"active":                        llx.BoolData(boolValue(user.Active)),
			"userType":                      llx.StringData(string(user.UserType)),
			"primaryEmail":                  llx.StringData(primaryEmail),
			"primaryEmailVerified":          llx.BoolData(primaryVerified),
			"emails":                        llx.ArrayData(emails, types.Dict),
			"isFederated":                   llx.BoolData(isFederated),
			"mfaStatus":                     llx.StringData(mfaStatus),
			"preferredAuthenticationFactor": llx.StringData(preferredFactor),
			"mfaEnabledOn":                  llx.StringData(mfaEnabledOn),
			"isLocked":                      llx.BoolData(isLocked),
			"loginAttempts":                 llx.IntData(loginAttempts),
			"lastSuccessfulLogin":           llx.StringData(lastSuccessful),
			"previousSuccessfulLogin":       llx.StringData(prevSuccessful),
			"lastFailedLogin":               llx.StringData(lastFail),
			"capabilities":                  llx.MapData(capabilities, types.Bool),
			"groups":                        llx.ArrayData(groups, types.Dict),
			"created":                       llx.TimeDataPtr(ociScimCreatedAt(user.Meta)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlUser)
	}

	return res, nil
}

func (o *mqlOciIdentityDomain) groups() ([]any, error) {
	client, err := o.domainClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	groups := []identitydomains.Group{}
	startIndex := 1
	for {
		response, err := client.ListGroups(ctx, identitydomains.ListGroupsRequest{
			StartIndex: common.Int(startIndex),
			Count:      common.Int(ociScimPageSize),
		})
		if err != nil {
			return nil, err
		}

		groups = append(groups, response.Groups.Resources...)

		next, more := ociScimNextIndex(startIndex, len(response.Groups.Resources), response.Groups.TotalResults)
		if !more {
			break
		}
		startIndex = next
	}

	res := make([]any, 0, len(groups))
	for i := range groups {
		group := groups[i]

		isRequestable := false
		if ext := group.UrnIetfParamsScimSchemasOracleIdcsExtensionRequestableGroup; ext != nil {
			isRequestable = boolValue(ext.Requestable)
		}

		mqlGroup, err := CreateResource(o.MqlRuntime, "oci.identity.domain.group", map[string]*llx.RawData{
			"__id":          llx.StringData(o.Id.Data + "/group/" + stringValue(group.Id)),
			"id":            llx.StringDataPtr(group.Id),
			"ocid":          llx.StringDataPtr(group.Ocid),
			"displayName":   llx.StringDataPtr(group.DisplayName),
			"description":   llx.StringData(ociGroupDescription(group)),
			"memberCount":   llx.IntData(int64(len(group.Members))),
			"isRequestable": llx.BoolData(isRequestable),
			"created":       llx.TimeDataPtr(ociScimCreatedAt(group.Meta)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlGroup)
	}

	return res, nil
}

func (o *mqlOciIdentityDomain) passwordPolicies() ([]any, error) {
	client, err := o.domainClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	policies := []identitydomains.PasswordPolicy{}
	startIndex := 1
	for {
		response, err := client.ListPasswordPolicies(ctx, identitydomains.ListPasswordPoliciesRequest{
			StartIndex: common.Int(startIndex),
			Count:      common.Int(ociScimPageSize),
		})
		if err != nil {
			return nil, err
		}

		policies = append(policies, response.PasswordPolicies.Resources...)

		next, more := ociScimNextIndex(startIndex, len(response.PasswordPolicies.Resources), response.PasswordPolicies.TotalResults)
		if !more {
			break
		}
		startIndex = next
	}

	res := make([]any, 0, len(policies))
	for i := range policies {
		p := policies[i]

		mqlPolicy, err := CreateResource(o.MqlRuntime, "oci.identity.domain.passwordPolicy", map[string]*llx.RawData{
			"__id":                     llx.StringData(o.Id.Data + "/passwordPolicy/" + stringValue(p.Id)),
			"id":                       llx.StringDataPtr(p.Id),
			"ocid":                     llx.StringDataPtr(p.Ocid),
			"name":                     llx.StringDataPtr(p.Name),
			"description":              llx.StringDataPtr(p.Description),
			"minLength":                llx.IntDataDefault(p.MinLength, 0),
			"maxLength":                llx.IntDataDefault(p.MaxLength, 0),
			"minAlphas":                llx.IntDataDefault(p.MinAlphas, 0),
			"minNumerals":              llx.IntDataDefault(p.MinNumerals, 0),
			"minAlphaNumerals":         llx.IntDataDefault(p.MinAlphaNumerals, 0),
			"minSpecialChars":          llx.IntDataDefault(p.MinSpecialChars, 0),
			"maxSpecialChars":          llx.IntDataDefault(p.MaxSpecialChars, 0),
			"minLowerCase":             llx.IntDataDefault(p.MinLowerCase, 0),
			"minUpperCase":             llx.IntDataDefault(p.MinUpperCase, 0),
			"minUniqueChars":           llx.IntDataDefault(p.MinUniqueChars, 0),
			"maxRepeatedChars":         llx.IntDataDefault(p.MaxRepeatedChars, 0),
			"passwordExpiresAfter":     llx.IntDataDefault(p.PasswordExpiresAfter, 0),
			"passwordExpireWarning":    llx.IntDataDefault(p.PasswordExpireWarning, 0),
			"minPasswordAge":           llx.IntDataDefault(p.MinPasswordAge, 0),
			"numPasswordsInHistory":    llx.IntDataDefault(p.NumPasswordsInHistory, 0),
			"maxIncorrectAttempts":     llx.IntDataDefault(p.MaxIncorrectAttempts, 0),
			"lockoutDuration":          llx.IntDataDefault(p.LockoutDuration, 0),
			"dictionaryWordDisallowed": llx.BoolData(boolValue(p.DictionaryWordDisallowed)),
			"userNameDisallowed":       llx.BoolData(boolValue(p.UserNameDisallowed)),
			"firstNameDisallowed":      llx.BoolData(boolValue(p.FirstNameDisallowed)),
			"lastNameDisallowed":       llx.BoolData(boolValue(p.LastNameDisallowed)),
			"startsWithAlphabet":       llx.BoolData(boolValue(p.StartsWithAlphabet)),
			"disallowedSubstrings":     llx.ArrayData(stringsToAny(p.DisallowedSubstrings), types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlPolicy)
	}

	return res, nil
}

func (o *mqlOciIdentityDomain) authenticationFactorSettings() (*mqlOciIdentityDomainAuthenticationFactorSettings, error) {
	client, err := o.domainClient()
	if err != nil {
		return nil, err
	}

	response, err := client.ListAuthenticationFactorSettings(context.Background(),
		identitydomains.ListAuthenticationFactorSettingsRequest{})
	if err != nil {
		return nil, err
	}

	// The service models this as a collection, but a domain has exactly one
	// settings resource. An empty collection means the domain returned no
	// multi-factor configuration at all, which is a null answer rather than a
	// domain with every factor disabled - reporting the latter would assert
	// something the API never said.
	if len(response.AuthenticationFactorSettings.Resources) == 0 {
		o.AuthenticationFactorSettings.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	s := response.AuthenticationFactorSettings.Resources[0]

	res, err := CreateResource(o.MqlRuntime, "oci.identity.domain.authenticationFactorSettings", map[string]*llx.RawData{
		"__id":                     llx.StringData(o.Id.Data + "/authenticationFactorSettings"),
		"id":                       llx.StringDataPtr(s.Id),
		"smsEnabled":               llx.BoolData(boolValue(s.SmsEnabled)),
		"emailEnabled":             llx.BoolData(boolValue(s.EmailEnabled)),
		"totpEnabled":              llx.BoolData(boolValue(s.TotpEnabled)),
		"pushEnabled":              llx.BoolData(boolValue(s.PushEnabled)),
		"phoneCallEnabled":         llx.BoolData(boolValue(s.PhoneCallEnabled)),
		"fidoAuthenticatorEnabled": llx.BoolData(boolValue(s.FidoAuthenticatorEnabled)),
		"yubicoOtpEnabled":         llx.BoolData(boolValue(s.YubicoOtpEnabled)),
		"securityQuestionsEnabled": llx.BoolData(boolValue(s.SecurityQuestionsEnabled)),
		"bypassCodeEnabled":        llx.BoolData(boolValue(s.BypassCodeEnabled)),
		"mfaEnrollmentType":        llx.StringDataPtr(s.MfaEnrollmentType),
		"mfaEnabledCategory":       llx.StringDataPtr(s.MfaEnabledCategory),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOciIdentityDomainAuthenticationFactorSettings), nil
}

func (o *mqlOciIdentityDomain) apps() ([]any, error) {
	client, err := o.domainClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	apps := []identitydomains.App{}
	startIndex := 1
	for {
		response, err := client.ListApps(ctx, identitydomains.ListAppsRequest{
			StartIndex: common.Int(startIndex),
			Count:      common.Int(ociScimPageSize),
		})
		if err != nil {
			return nil, err
		}

		apps = append(apps, response.Apps.Resources...)

		next, more := ociScimNextIndex(startIndex, len(response.Apps.Resources), response.Apps.TotalResults)
		if !more {
			break
		}
		startIndex = next
	}

	res := make([]any, 0, len(apps))
	for i := range apps {
		app := apps[i]

		mqlApp, err := CreateResource(o.MqlRuntime, "oci.identity.domain.app", map[string]*llx.RawData{
			"__id":            llx.StringData(o.Id.Data + "/app/" + stringValue(app.Id)),
			"id":              llx.StringDataPtr(app.Id),
			"name":            llx.StringDataPtr(app.Name),
			"displayName":     llx.StringDataPtr(app.DisplayName),
			"description":     llx.StringDataPtr(app.Description),
			"active":          llx.BoolData(boolValue(app.Active)),
			"isOAuthClient":   llx.BoolData(boolValue(app.IsOAuthClient)),
			"isOAuthResource": llx.BoolData(boolValue(app.IsOAuthResource)),
			"clientType":      llx.StringData(string(app.ClientType)),
			"allowedGrants":   llx.ArrayData(stringsToAny(app.AllowedGrants), types.String),
			"landingPageUrl":  llx.StringDataPtr(app.LandingPageUrl),
			"isWebTierPolicy": llx.BoolData(boolValue(app.IsWebTierPolicy)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlApp)
	}

	return res, nil
}

// ociScimNextIndex advances a SCIM startIndex/count pagination loop.
//
// SCIM indexes are 1-based and the service reports the total separately from
// the page, so the loop cannot stop on an absent page token the way the rest of
// OCI does. It returns the next start index and whether another page is due.
//
// A page that comes back empty always terminates the loop: without that guard a
// service reporting a total larger than it will actually return - or a page
// truncated for any reason - would spin forever.
func ociScimNextIndex(startIndex int, returned int, total *int) (int, bool) {
	if returned == 0 {
		return 0, false
	}
	next := startIndex + returned
	if total == nil {
		// No total to compare against: keep going only while pages come back
		// full, since a short page means the collection is exhausted.
		return next, returned >= ociScimPageSize
	}
	if next > *total {
		return 0, false
	}
	return next, true
}

// ociPrimaryEmail picks the primary address out of a SCIM emails list, falling
// back to the first entry when none is flagged primary.
func ociPrimaryEmail(emails []identitydomains.UserEmails) (string, bool) {
	if len(emails) == 0 {
		return "", false
	}
	for i := range emails {
		if boolValue(emails[i].Primary) {
			return stringValue(emails[i].Value), boolValue(emails[i].Verified)
		}
	}
	return stringValue(emails[0].Value), boolValue(emails[0].Verified)
}

// ociScimCreatedAt pulls the creation timestamp out of a SCIM meta block.
func ociScimCreatedAt(meta *identitydomains.Meta) *time.Time {
	if meta == nil || meta.Created == nil {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, *meta.Created)
	if err != nil {
		// SCIM timestamps are RFC3339, but a malformed one is not worth
		// failing the whole listing over.
		return nil
	}
	return &parsed
}

// ociGroupDescription reads a group's description, which SCIM carries on an
// extension rather than on the group itself.
func ociGroupDescription(group identitydomains.Group) string {
	if ext := group.UrnIetfParamsScimSchemasOracleIdcsExtensionGroupGroup; ext != nil {
		return stringValue(ext.Description)
	}
	return ""
}
