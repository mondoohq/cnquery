package ms365

import (
	"context"
	"strings"

	"errors"
	"github.com/microsoft/kiota-abstractions-go/authentication"
	a "github.com/microsoft/kiota-authentication-azure-go"
	msgraphclient "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/applications"
	"github.com/microsoftgraph/msgraph-sdk-go/devicemanagement"
	"github.com/microsoftgraph/msgraph-sdk-go/domains"
	"github.com/microsoftgraph/msgraph-sdk-go/groups"
	"github.com/microsoftgraph/msgraph-sdk-go/groupsettings"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/organization"
	"github.com/microsoftgraph/msgraph-sdk-go/policies"
	"github.com/microsoftgraph/msgraph-sdk-go/rolemanagement"
	"github.com/microsoftgraph/msgraph-sdk-go/security"
	"github.com/microsoftgraph/msgraph-sdk-go/serviceprincipals"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
	microsoft "go.mondoo.com/cnquery/motor/providers/microsoft"
	msgraphadapter "go.mondoo.com/cnquery/motor/providers/microsoft/msgraph/msgraphclient"
	msgraphconv "go.mondoo.com/cnquery/motor/providers/microsoft/msgraph/msgraphconv"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (m *mqlMicrosoft) id() (string, error) {
	return "microsoft", nil
}

func (m *mqlMicrosoftOrganization) id() (string, error) {
	return m.Id()
}

func (m *mqlMicrosoft) GetSettings() ([]interface{}, error) {
	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	settings, err := graphClient.GroupSettings().Get(ctx, &groupsettings.GroupSettingsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}
	return core.JsonToDictSlice(msgraphconv.NewSettings(settings.GetValue()))
}

func (m *mqlMicrosoft) GetOrganizations() ([]interface{}, error) {
	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Organization.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.Organization().Get(ctx, &organization.OrganizationRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}

	res := []interface{}{}
	orgs := resp.GetValue()
	for i := range orgs {
		org := orgs[i]

		assignedPlans, _ := core.JsonToDictSlice(msgraphconv.NewAssignedPlans(org.GetAssignedPlans()))
		verifiedDomains, _ := core.JsonToDictSlice(msgraphconv.NewVerifiedDomains(org.GetVerifiedDomains()))

		mqlResource, err := m.MotorRuntime.CreateResource("microsoft.organization",
			"id", core.ToString(org.GetId()),
			"assignedPlans", assignedPlans,
			"createdDateTime", org.GetCreatedDateTime(),
			"displayName", core.ToString(org.GetDisplayName()),
			"verifiedDomains", verifiedDomains,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

func (a *mqlMicrosoftGroup) id() (string, error) {
	return a.Id()
}

func (a *mqlMicrosoftGroup) GetMembers() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (m *mqlMicrosoft) GetGroups() ([]interface{}, error) {
	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Group.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	resp, err := graphClient.Groups().Get(ctx, &groups.GroupsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}

	res := []interface{}{}
	grps := resp.GetValue()
	for _, grp := range grps {
		graphGrp, err := m.MotorRuntime.CreateResource("microsoft.group",
			"id", core.ToString(grp.GetId()),
			"displayName", core.ToString(grp.GetDisplayName()),
			"securityEnabled", core.ToBool(grp.GetSecurityEnabled()),
			"mailEnabled", core.ToBool(grp.GetMailEnabled()),
			"mailNickname", core.ToString(grp.GetMailNickname()),
			"mail", core.ToString(grp.GetMail()),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, graphGrp)
	}

	return res, nil
}

func (m *mqlMicrosoftUser) id() (string, error) {
	return m.Id()
}

func (m *mqlMicrosoft) GetUsers() ([]interface{}, error) {
	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("User.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}

	selectFields := []string{
		"id", "accountEnabled", "city", "companyName", "country", "createdDateTime", "department", "displayName", "employeeId", "givenName",
		"jobTitle", "mail", "mobilePhone", "otherMails", "officeLocation", "postalCode", "state", "streetAddress", "surname", "userPrincipalName", "userType",
	}
	ctx := context.Background()
	resp, err := graphClient.Users().Get(ctx, &users.UsersRequestBuilderGetRequestConfiguration{QueryParameters: &users.UsersRequestBuilderGetQueryParameters{
		Select: selectFields,
	}})
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}

	res := []interface{}{}
	users := resp.GetValue()
	for i := range users {
		user := users[i]

		mqlResource, err := m.MotorRuntime.CreateResource("microsoft.user",
			"id", core.ToString(user.GetId()),
			"accountEnabled", core.ToBool(user.GetAccountEnabled()),
			"city", core.ToString(user.GetCity()),
			"companyName", core.ToString(user.GetCompanyName()),
			"country", core.ToString(user.GetCountry()),
			"createdDateTime", user.GetCreatedDateTime(),
			"department", core.ToString(user.GetDepartment()),
			"displayName", core.ToString(user.GetDisplayName()),
			"employeeId", core.ToString(user.GetEmployeeId()),
			"givenName", core.ToString(user.GetGivenName()),
			"jobTitle", core.ToString(user.GetJobTitle()),
			"mail", core.ToString(user.GetMail()),
			"mobilePhone", core.ToString(user.GetMobilePhone()),
			"otherMails", core.StrSliceToInterface(user.GetOtherMails()),
			"officeLocation", core.ToString(user.GetOfficeLocation()),
			"postalCode", core.ToString(user.GetPostalCode()),
			"state", core.ToString(user.GetState()),
			"streetAddress", core.ToString(user.GetStreetAddress()),
			"surname", core.ToString(user.GetSurname()),
			"userPrincipalName", core.ToString(user.GetUserPrincipalName()),
			"userType", core.ToString(user.GetUserType()),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

func (m *mqlMicrosoftServiceprincipal) id() (string, error) {
	return m.Id()
}

func (m *mqlMicrosoft) GetServiceprincipals() ([]interface{}, error) {
	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Application.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}

	// TODO: we need to use Top, there are more than 100 SPs.
	ctx := context.Background()
	resp, err := graphClient.ServicePrincipals().Get(ctx, &serviceprincipals.ServicePrincipalsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}

	res := []interface{}{}
	sps := resp.GetValue()
	for _, sp := range sps {
		mqlResource, err := m.MotorRuntime.CreateResource("microsoft.serviceprincipal",
			"id", core.ToString(sp.GetId()),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

func (m *mqlMicrosoftDomain) id() (string, error) {
	return m.Id()
}

func (m *mqlMicrosoft) GetDomains() ([]interface{}, error) {
	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Domain.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.Domains().Get(ctx, &domains.DomainsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}

	res := []interface{}{}
	domains := resp.GetValue()
	for i := range domains {
		domain := domains[i]

		mqlResource, err := m.MotorRuntime.CreateResource("microsoft.domain",
			"id", core.ToString(domain.GetId()),
			"authenticationType", core.ToString(domain.GetAuthenticationType()),
			"availabilityStatus", core.ToString(domain.GetAvailabilityStatus()),
			"isAdminManaged", core.ToBool(domain.GetIsAdminManaged()),
			"isDefault", core.ToBool(domain.GetIsDefault()),
			"isInitial", core.ToBool(domain.GetIsInitial()),
			"isRoot", core.ToBool(domain.GetIsRoot()),
			"isVerified", core.ToBool(domain.GetIsVerified()),
			"passwordNotificationWindowInDays", core.ToInt64From32(domain.GetPasswordNotificationWindowInDays()),
			"passwordValidityPeriodInDays", core.ToInt64From32(domain.GetPasswordValidityPeriodInDays()),
			"supportedServices", core.StrSliceToInterface(domain.GetSupportedServices()),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

func (m *mqlMicrosoftDomaindnsrecord) id() (string, error) {
	return m.Id()
}

func (m *mqlMicrosoftDomain) GetServiceConfigurationRecords() ([]interface{}, error) {
	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Domain.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	id, err := m.Id()
	if err != nil {
		return nil, err
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.DomainsById(id).ServiceConfigurationRecords().Get(ctx, &domains.ItemServiceConfigurationRecordsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}

	res := []interface{}{}
	records := resp.GetValue()
	for i := range records {
		record := records[i]
		properties := getDomainsDnsRecordProperties(record)

		mqlResource, err := m.MotorRuntime.CreateResource("microsoft.domaindnsrecord",
			"id", core.ToString(record.GetId()),
			"isOptional", core.ToBool(record.GetIsOptional()),
			"label", core.ToString(record.GetLabel()),
			"recordType", core.ToString(record.GetRecordType()),
			"supportedService", core.ToString(record.GetSupportedService()),
			"ttl", core.ToInt64From32(record.GetTtl()),
			"properties", properties,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

func getDomainsDnsRecordProperties(record models.DomainDnsRecordable) map[string]interface{} {
	props := map[string]interface{}{}
	if record.GetOdataType() != nil {
		props["@odata.type"] = *record.GetOdataType()
	}
	txtRecord, ok := record.(*models.DomainDnsTxtRecord)
	if ok {
		if txtRecord.GetText() != nil {
			props["text"] = *txtRecord.GetText()
		}
	}
	mxRecord, ok := record.(*models.DomainDnsMxRecord)
	if ok {
		if mxRecord.GetMailExchange() != nil {
			props["mailExchange"] = *mxRecord.GetMailExchange()
		}
		if mxRecord.GetPreference() != nil {
			props["preference"] = *mxRecord.GetPreference()
		}
	}
	cNameRecord, ok := record.(*models.DomainDnsCnameRecord)
	if ok {
		if cNameRecord.GetCanonicalName() != nil {
			props["canonicalName"] = *cNameRecord.GetCanonicalName()
		}
	}
	srvRecord, ok := record.(*models.DomainDnsSrvRecord)
	if ok {
		if srvRecord.GetNameTarget() != nil {
			props["nameTarget"] = *srvRecord.GetNameTarget()
		}
		if srvRecord.GetPort() != nil {
			props["port"] = *srvRecord.GetPort()
		}
		if srvRecord.GetPriority() != nil {
			props["priority"] = *srvRecord.GetPriority()
		}
		if srvRecord.GetProtocol() != nil {
			props["protocol"] = *srvRecord.GetProtocol()
		}
		if srvRecord.GetService() != nil {
			props["service"] = *srvRecord.GetService()
		}
		if srvRecord.GetWeight() != nil {
			props["weight"] = *srvRecord.GetWeight()
		}
	}
	return props
}

func (m *mqlMicrosoftApplication) id() (string, error) {
	return m.Id()
}

func (m *mqlMicrosoft) GetApplications() ([]interface{}, error) {
	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Application.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.Applications().Get(ctx, &applications.ApplicationsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}

	res := []interface{}{}
	apps := resp.GetValue()
	for i := range apps {
		app := apps[i]

		mqlResource, err := m.MotorRuntime.CreateResource("microsoft.application",
			"id", core.ToString(app.GetId()),
			"appId", core.ToString(app.GetAppId()),
			"createdDateTime", app.GetCreatedDateTime(),
			"identifierUris", core.StrSliceToInterface(app.GetIdentifierUris()),
			"displayName", core.ToString(app.GetDisplayName()),
			"publisherDomain", core.ToString(app.GetPublisherDomain()),
			"signInAudience", core.ToString(app.GetSignInAudience()),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

func (m *mqlMicrosoftUser) GetSettings() (interface{}, error) {
	id, err := m.Id()
	if err != nil {
		return nil, err
	}

	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("User.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	userSettings, err := graphClient.UsersById(id).Settings().Get(ctx, &users.ItemSettingsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}

	return core.JsonToDict(msgraphconv.NewUserSettings(userSettings))
}

func (m *mqlMicrosoftSecurity) id() (string, error) {
	return "microsoft.security", nil
}

func msSecureScoreToMql(runtime *resources.Runtime, score models.SecureScoreable) (interface{}, error) {
	if score == nil {
		return nil, nil
	}
	averageComparativeScores := []interface{}{}
	graphAverageComparativeScores := score.GetAverageComparativeScores()
	for j := range graphAverageComparativeScores {
		entry, err := core.JsonToDict(msgraphconv.NewAverageComparativeScore(graphAverageComparativeScores[j]))
		if err != nil {
			return nil, err
		}
		averageComparativeScores = append(averageComparativeScores, entry)
	}

	controlScores := []interface{}{}
	graphControlScores := score.GetControlScores()
	for j := range graphControlScores {
		entry, err := core.JsonToDict(msgraphconv.NewControlScore(graphControlScores[j]))
		if err != nil {
			return nil, err
		}
		controlScores = append(controlScores, entry)
	}

	vendorInformation, err := core.JsonToDict(msgraphconv.NewSecurityVendorInformation(score.GetVendorInformation()))
	if err != nil {
		return nil, err
	}

	mqlResource, err := runtime.CreateResource("microsoft.security.securityscore",
		"id", core.ToString(score.GetId()),
		"activeUserCount", core.ToInt64From32(score.GetActiveUserCount()),
		"averageComparativeScores", averageComparativeScores,
		"azureTenantId", core.ToString(score.GetAzureTenantId()),
		"controlScores", controlScores,
		"createdDateTime", score.GetCreatedDateTime(),
		"currentScore", core.ToFloat64(score.GetCurrentScore()),
		"enabledServices", core.StrSliceToInterface(score.GetEnabledServices()),
		"licensedUserCount", core.ToInt64From32(score.GetLicensedUserCount()),
		"maxScore", core.ToFloat64(score.GetMaxScore()),
		"vendorInformation", vendorInformation,
	)
	if err != nil {
		return nil, err
	}
	return mqlResource, nil
}

func (m *mqlMicrosoftSecurity) GetLatestSecureScores() (interface{}, error) {
	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("SecurityEvents.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.Security().SecureScores().Get(ctx, &security.SecureScoresRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}

	scores := resp.GetValue()
	if len(scores) == 0 {
		return nil, errors.New("could not retrieve any score")
	}

	latestScore := scores[0]
	for i := range scores {
		score := scores[i]
		if score.GetCreatedDateTime() != nil && (latestScore.GetCreatedDateTime() == nil || score.GetCreatedDateTime().Before(*latestScore.GetCreatedDateTime())) {
			latestScore = score
		}
	}

	return msSecureScoreToMql(m.MotorRuntime, latestScore)
}

// see https://docs.microsoft.com/en-us/graph/api/securescore-get?view=graph-rest-1.0&tabs=http
func (m *mqlMicrosoftSecurity) GetSecureScores() ([]interface{}, error) {
	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("SecurityEvents.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	resp, err := graphClient.Security().SecureScores().Get(ctx, &security.SecureScoresRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}

	res := []interface{}{}
	scores := resp.GetValue()
	for i := range scores {
		score := scores[i]
		mqlResource, err := msSecureScoreToMql(m.MotorRuntime, score)
		if err != nil {
			return nil, err
		}

		res = append(res, mqlResource)
	}
	return res, nil
}

func (s *mqlMicrosoftSecuritySecurityscore) id() (string, error) {
	return s.Id()
}

func (s *mqlMicrosoftPolicies) id() (string, error) {
	return "microsoft.policies", nil
}

func (m *mqlMicrosoftPolicies) GetAuthorizationPolicy() (interface{}, error) {
	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Policy.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	resp, err := graphClient.Policies().AuthorizationPolicy().Get(ctx, &policies.AuthorizationPolicyRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}

	return core.JsonToDict(msgraphconv.NewAuthorizationPolicy(resp))
}

func (m *mqlMicrosoftPolicies) GetIdentitySecurityDefaultsEnforcementPolicy() (interface{}, error) {
	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Policy.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	policy, err := graphClient.Policies().IdentitySecurityDefaultsEnforcementPolicy().Get(ctx, &policies.IdentitySecurityDefaultsEnforcementPolicyRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}

	return core.JsonToDict(msgraphconv.NewIdentitySecurityDefaultsEnforcementPolicy(policy))
}

// https://docs.microsoft.com/en-us/graph/api/adminconsentrequestpolicy-get?view=graph-rest-
func (m *mqlMicrosoftPolicies) GetAdminConsentRequestPolicy() (interface{}, error) {
	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Policy.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	policy, err := graphClient.Policies().AdminConsentRequestPolicy().Get(ctx, &policies.AdminConsentRequestPolicyRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}
	return core.JsonToDict(msgraphconv.NewAdminConsentRequestPolicy(policy))
}

// https://docs.microsoft.com/en-us/azure/active-directory/manage-apps/configure-user-consent?tabs=azure-powershell
// https://docs.microsoft.com/en-us/graph/api/permissiongrantpolicy-list?view=graph-rest-1.0&tabs=http
func (m *mqlMicrosoftPolicies) GetPermissionGrantPolicies() (interface{}, error) {
	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Policy.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	resp, err := graphClient.Policies().PermissionGrantPolicies().Get(ctx, &policies.PermissionGrantPoliciesRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}
	return core.JsonToDictSlice(msgraphconv.NewPermissionGrantPolicies(resp.GetValue()))
}

func (m *mqlMicrosoftRolemanagement) id() (string, error) {
	return "microsoft.rolemanagement", nil
}

func (m *mqlMicrosoftRolemanagement) GetRoleDefinitions() (interface{}, error) {
	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Directory.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.RoleManagement().Directory().RoleDefinitions().Get(ctx, &rolemanagement.DirectoryRoleDefinitionsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}

	res := []interface{}{}
	roles := resp.GetValue()
	for i := range roles {
		role := roles[i]

		rolePermissions, _ := core.JsonToDictSlice(msgraphconv.NewUnifiedRolePermissions(role.GetRolePermissions()))

		mqlResource, err := m.MotorRuntime.CreateResource("microsoft.rolemanagement.roledefinition",
			"id", core.ToString(role.GetId()),
			"description", core.ToString(role.GetDescription()),
			"displayName", core.ToString(role.GetDisplayName()),
			"isBuiltIn", core.ToBool(role.GetIsBuiltIn()),
			"isEnabled", core.ToBool(role.GetIsEnabled()),
			"rolePermissions", rolePermissions,
			"templateId", core.ToString(role.GetTemplateId()),
			"version", core.ToString(role.GetVersion()),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

func (m *mqlMicrosoftRolemanagementRoledefinition) id() (string, error) {
	return m.Id()
}

func (m *mqlMicrosoftRolemanagementRoledefinition) GetAssignments() ([]interface{}, error) {
	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Directory.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}

	roleDefinitionID, err := m.Id()
	if err != nil {
		return nil, err
	}
	filter := "roleDefinitionId eq '" + roleDefinitionID + "'"

	ctx := context.Background()
	requestConfig := &rolemanagement.DirectoryRoleAssignmentsRequestBuilderGetRequestConfiguration{
		QueryParameters: &rolemanagement.DirectoryRoleAssignmentsRequestBuilderGetQueryParameters{
			Filter: &filter,
			Expand: []string{"principal"},
		},
	}
	resp, err := graphClient.RoleManagement().Directory().RoleAssignments().Get(ctx, requestConfig)
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}

	roleAssignments := resp.GetValue()

	res := []interface{}{}
	for i := range roleAssignments {
		roleAssignment := roleAssignments[i]
		principal, _ := core.JsonToDict(msgraphconv.NewDirectoryPrincipal(roleAssignment.GetPrincipal()))
		mqlResource, err := m.MotorRuntime.CreateResource("microsoft.rolemanagement.roleassignment",
			"id", core.ToString(roleAssignment.GetId()),
			"roleDefinitionId", core.ToString(roleAssignment.GetRoleDefinitionId()),
			"principalId", core.ToString(roleAssignment.GetPrincipalId()),
			"principal", principal,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

func (m *mqlMicrosoftRolemanagementRoleassignment) id() (string, error) {
	return m.Id()
}

func (m *mqlMicrosoftDevicemanagement) id() (string, error) {
	return "microsoft.devicemanagement", nil
}

func (m *mqlMicrosoftDevicemanagement) GetDeviceConfigurations() ([]interface{}, error) {
	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("DeviceManagementConfiguration.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.DeviceManagement().DeviceConfigurations().Get(ctx, &devicemanagement.DeviceConfigurationsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}

	res := []interface{}{}
	configurations := resp.GetValue()
	for i := range configurations {
		configuration := configurations[i]
		properties := getConfigurationProperties(configuration)
		mqlResource, err := m.MotorRuntime.CreateResource("microsoft.devicemanagement.deviceconfiguration",
			"id", core.ToString(configuration.GetId()),
			"lastModifiedDateTime", configuration.GetLastModifiedDateTime(),
			"createdDateTime", configuration.GetCreatedDateTime(),
			"description", core.ToString(configuration.GetDescription()),
			"displayName", core.ToString(configuration.GetDisplayName()),
			"version", core.ToInt64From32(configuration.GetVersion()),
			"properties", properties,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

// TODO: androidDeviceOwnerGeneralDeviceConfiguration missing
func getConfigurationProperties(config models.DeviceConfigurationable) map[string]interface{} {
	props := map[string]interface{}{}
	if config.GetOdataType() != nil {
		props["@odata.type"] = *config.GetOdataType()
	}

	agdc, ok := config.(*models.AndroidGeneralDeviceConfiguration)
	if ok {
		if agdc.GetPasswordRequired() != nil {
			props["passwordRequired"] = *agdc.GetPasswordRequired()
		}
		if agdc.GetPasswordSignInFailureCountBeforeFactoryReset() != nil {
			props["passwordSignInFailureCountBeforeFactoryReset"] = *agdc.GetPasswordSignInFailureCountBeforeFactoryReset()
		}
		if agdc.GetPasswordMinimumLength() != nil {
			props["passwordMinimumLength"] = *agdc.GetPasswordMinimumLength()
		}
		if agdc.GetStorageRequireDeviceEncryption() != nil {
			props["storageRequireDeviceEncryption"] = *agdc.GetStorageRequireDeviceEncryption()
		}
		if agdc.GetPasswordRequiredType() != nil {
			props["passwordRequiredType"] = agdc.GetPasswordRequiredType().String()
		}
		if agdc.GetPasswordExpirationDays() != nil {
			props["passwordExpirationDays"] = *agdc.GetPasswordExpirationDays()
		}
		if agdc.GetPasswordMinutesOfInactivityBeforeScreenTimeout() != nil {
			props["passwordMinutesOfInactivityBeforeScreenTimeout"] = *agdc.GetPasswordMinutesOfInactivityBeforeScreenTimeout()
		}
	}
	w10gc, ok := config.(*models.Windows10GeneralConfiguration)
	if ok {
		if w10gc.GetPasswordRequired() != nil {
			props["passwordRequired"] = *w10gc.GetPasswordRequired()
		}
		if w10gc.GetPasswordBlockSimple() != nil {
			props["passwordBlockSimple"] = *w10gc.GetPasswordBlockSimple()
		}
		if w10gc.GetPasswordMinutesOfInactivityBeforeScreenTimeout() != nil {
			props["passwordMinutesOfInactivityBeforeScreenTimeout"] = *w10gc.GetPasswordMinutesOfInactivityBeforeScreenTimeout()
		}
		if w10gc.GetPasswordSignInFailureCountBeforeFactoryReset() != nil {
			props["passwordSignInFailureCountBeforeFactoryReset"] = *w10gc.GetPasswordSignInFailureCountBeforeFactoryReset()
		}
		if w10gc.GetPasswordMinimumLength() != nil {
			props["passwordMinimumLength"] = *w10gc.GetPasswordMinimumLength()
		}
		if w10gc.GetPasswordRequiredType() != nil {
			props["passwordRequiredType"] = w10gc.GetPasswordRequiredType().String()
		}
		if w10gc.GetPasswordExpirationDays() != nil {
			props["passwordExpirationDays"] = *w10gc.GetPasswordExpirationDays()
		}
		if w10gc.GetPasswordExpirationDays() != nil {
			props["passwordExpirationDays"] = *w10gc.GetPasswordExpirationDays()
		}
	}
	macdc, ok := config.(*models.MacOSGeneralDeviceConfiguration)
	if ok {
		if macdc.GetPasswordMinutesOfInactivityBeforeScreenTimeout() != nil {
			props["passwordMinutesOfInactivityBeforeScreenTimeout"] = *macdc.GetPasswordMinutesOfInactivityBeforeScreenTimeout()
		}
		if macdc.GetPasswordMinimumLength() != nil {
			props["passwordMinimumLength"] = *macdc.GetPasswordMinimumLength()
		}
		if macdc.GetPasswordMinutesOfInactivityBeforeLock() != nil {
			props["passwordMinutesOfInactivityBeforeLock"] = *macdc.GetPasswordMinutesOfInactivityBeforeLock()
		}
		if macdc.GetPasswordRequiredType() != nil {
			props["passwordRequiredType"] = macdc.GetPasswordRequiredType().String()
		}
		if macdc.GetPasswordBlockSimple() != nil {
			props["passwordBlockSimple"] = *macdc.GetPasswordBlockSimple()
		}
		if macdc.GetPasswordRequired() != nil {
			props["passwordRequired"] = *macdc.GetPasswordRequired()
		}
		if macdc.GetPasswordExpirationDays() != nil {
			props["passwordExpirationDays"] = *macdc.GetPasswordExpirationDays()
		}
	}

	iosdc, ok := config.(*models.IosGeneralDeviceConfiguration)
	if ok {
		if iosdc.GetPasscodeSignInFailureCountBeforeWipe() != nil {
			props["passcodeSignInFailureCountBeforeWipe"] = *iosdc.GetPasscodeSignInFailureCountBeforeWipe()
		}
		if iosdc.GetPasscodeMinimumLength() != nil {
			props["passcodeMinimumLength"] = *iosdc.GetPasscodeMinimumLength()
		}
		if iosdc.GetPasscodeMinutesOfInactivityBeforeLock() != nil {
			props["passcodeMinutesOfInactivityBeforeLock"] = *iosdc.GetPasscodeMinutesOfInactivityBeforeLock()
		}
		if iosdc.GetPasscodeMinutesOfInactivityBeforeScreenTimeout() != nil {
			props["passcodeMinutesOfInactivityBeforeScreenTimeout"] = *iosdc.GetPasscodeMinutesOfInactivityBeforeScreenTimeout()
		}
		if iosdc.GetPasscodeRequiredType() != nil {
			props["passcodeRequiredType"] = iosdc.GetPasscodeRequiredType().String()
		}
		if iosdc.GetPasscodeBlockSimple() != nil {
			props["passcodeBlockSimple"] = *iosdc.GetPasscodeBlockSimple()
		}
		if iosdc.GetPasscodeRequired() != nil {
			props["passcodeRequired"] = *iosdc.GetPasscodeRequired()
		}
		if iosdc.GetPasscodeExpirationDays() != nil {
			props["passcodeExpirationDays"] = *iosdc.GetPasscodeExpirationDays()
		}
	}
	awpgdc, ok := config.(*models.AndroidWorkProfileGeneralDeviceConfiguration)
	if ok {
		if awpgdc.GetPasswordSignInFailureCountBeforeFactoryReset() != nil {
			props["passwordSignInFailureCountBeforeFactoryReset"] = *awpgdc.GetPasswordSignInFailureCountBeforeFactoryReset()
		}
		if awpgdc.GetPasswordMinimumLength() != nil {
			props["passwordMinimumLength"] = *awpgdc.GetPasswordMinimumLength()
		}
		if awpgdc.GetPasswordMinutesOfInactivityBeforeScreenTimeout() != nil {
			props["passwordMinutesOfInactivityBeforeScreenTimeout"] = *awpgdc.GetPasswordMinutesOfInactivityBeforeScreenTimeout()
		}
		if awpgdc.GetWorkProfilePasswordMinutesOfInactivityBeforeScreenTimeout() != nil {
			props["workProfilePasswordMinutesOfInactivityBeforeScreenTimeout"] = *awpgdc.GetWorkProfilePasswordMinutesOfInactivityBeforeScreenTimeout()
		}
		if awpgdc.GetPasswordRequiredType() != nil {
			props["passwordRequiredType"] = awpgdc.GetPasswordRequiredType().String()
		}
		if awpgdc.GetWorkProfilePasswordRequiredType() != nil {
			props["workProfilePasswordRequiredType"] = awpgdc.GetWorkProfilePasswordRequiredType().String()
		}
		if awpgdc.GetPasswordExpirationDays() != nil {
			props["passwordExpirationDays"] = *awpgdc.GetPasswordExpirationDays()
		}
	}
	return props
}

func (m *mqlMicrosoftDevicemanagement) GetDeviceCompliancePolicies() ([]interface{}, error) {
	provider, err := microsoftProvider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("DeviceManagementConfiguration.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphClient, err := graphClient(provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	requestConfig := &devicemanagement.DeviceCompliancePoliciesRequestBuilderGetRequestConfiguration{
		QueryParameters: &devicemanagement.DeviceCompliancePoliciesRequestBuilderGetQueryParameters{
			Expand: []string{"assignments"},
		},
	}
	resp, err := graphClient.DeviceManagement().DeviceCompliancePolicies().Get(ctx, requestConfig)
	if err != nil {
		return nil, msgraphconv.TransformError(err)
	}

	compliancePolicies := resp.GetValue()
	res := []interface{}{}
	for i := range compliancePolicies {
		compliancePolicy := compliancePolicies[i]

		assignments, _ := core.JsonToDictSlice(msgraphconv.NewDeviceCompliancePolicyAssignments(compliancePolicy.GetAssignments()))
		properties := getComplianceProperties(compliancePolicy)

		mqlResource, err := m.MotorRuntime.CreateResource("microsoft.devicemanagement.devicecompliancepolicy",
			"id", core.ToString(compliancePolicy.GetId()),
			"createdDateTime", compliancePolicy.GetCreatedDateTime(),
			"description", core.ToString(compliancePolicy.GetDescription()),
			"displayName", core.ToString(compliancePolicy.GetDisplayName()),
			"lastModifiedDateTime", compliancePolicy.GetLastModifiedDateTime(),
			"version", core.ToInt64From32(compliancePolicy.GetVersion()),
			"assignments", assignments,
			"properties", properties,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

// TODO: windows 10 props missing.
func getComplianceProperties(compliance models.DeviceCompliancePolicyable) map[string]interface{} {
	props := map[string]interface{}{}
	if compliance.GetOdataType() != nil {
		props["@odata.type"] = *compliance.GetOdataType()
	}

	ioscp, ok := compliance.(*models.IosCompliancePolicy)
	if ok {
		if ioscp.GetSecurityBlockJailbrokenDevices() != nil {
			props["securityBlockJailbrokenDevices"] = *ioscp.GetSecurityBlockJailbrokenDevices()
		}
		if ioscp.GetManagedEmailProfileRequired() != nil {
			props["managedEmailProfileRequired"] = *ioscp.GetManagedEmailProfileRequired()
		}
	}
	androidcp, ok := compliance.(*models.AndroidCompliancePolicy)
	if ok {
		if androidcp.GetSecurityBlockJailbrokenDevices() != nil {
			props["securityBlockJailbrokenDevices"] = *androidcp.GetSecurityBlockJailbrokenDevices()
		}
	}
	androidworkcp, ok := compliance.(*models.AndroidWorkProfileCompliancePolicy)
	if ok {
		if androidworkcp.GetSecurityBlockJailbrokenDevices() != nil {
			props["securityBlockJailbrokenDevices"] = *androidworkcp.GetSecurityBlockJailbrokenDevices()
		}
	}
	return props
}

func (m *mqlMicrosoftDevicemanagementDeviceconfiguration) id() (string, error) {
	return m.Id()
}

func (m *mqlMicrosoftDevicemanagementDevicecompliancepolicy) id() (string, error) {
	return m.Id()
}

func graphClient(t *microsoft.Provider) (*msgraphclient.GraphServiceClient, error) {
	auth, err := t.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	providerFunc := func() (authentication.AuthenticationProvider, error) {
		return a.NewAzureIdentityAuthenticationProviderWithScopes(auth, msgraphadapter.DefaultMSGraphScopes)
	}
	adapter, err := msgraphadapter.NewGraphRequestAdapterWithFn(providerFunc)
	if err != nil {
		return nil, err
	}
	graphClient := msgraphclient.NewGraphServiceClient(adapter)
	return graphClient, nil
}
