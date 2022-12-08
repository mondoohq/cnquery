package ms365

import (
	"context"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/microsoft/kiota-abstractions-go/authentication"
	a "github.com/microsoft/kiota-authentication-azure-go"
	"github.com/microsoftgraph/msgraph-sdk-go/applications"
	"github.com/microsoftgraph/msgraph-sdk-go/devicemanagement"
	"github.com/microsoftgraph/msgraph-sdk-go/domains"
	"github.com/microsoftgraph/msgraph-sdk-go/groups"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/organization"
	"github.com/microsoftgraph/msgraph-sdk-go/policies"
	"github.com/microsoftgraph/msgraph-sdk-go/rolemanagement"
	"github.com/microsoftgraph/msgraph-sdk-go/security"
	"github.com/microsoftgraph/msgraph-sdk-go/serviceprincipals"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
	microsoft "go.mondoo.com/cnquery/motor/providers/microsoft"
	"go.mondoo.com/cnquery/motor/providers/microsoft/msgraph/msgraphclient"
	"go.mondoo.com/cnquery/motor/providers/microsoft/msgraph/msgraphconv"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (m *mqlMsgraph) id() (string, error) {
	return "msgraph", nil
}

func (m *mqlMsgraphOrganization) id() (string, error) {
	return m.Id()
}

func (m *mqlMsgraph) GetSettings() ([]interface{}, error) {
	return []interface{}{}, nil
}

func (m *mqlMsgraphDomaindnsrecord) GetProperties() ([]interface{}, error) {
	return []interface{}{}, nil
}

func (m *mqlMsgraphDevicemanagementDeviceconfiguration) GetProperties() ([]interface{}, error) {
	return []interface{}{}, nil
}

func (m *mqlMsgraphDevicemanagementDevicecompliancepolicy) GetProperties() ([]interface{}, error) {
	return []interface{}{}, nil
}

func (m *mqlMsgraph) GetOrganizations() ([]interface{}, error) {
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
		return nil, msgraphclient.TransformODataError(err)
	}

	res := []interface{}{}
	orgs := resp.GetValue()
	for i := range orgs {
		org := orgs[i]

		assignedPlans, _ := core.JsonToDictSlice(msgraphconv.NewAssignedPlans(org.GetAssignedPlans()))
		verifiedDomains, _ := core.JsonToDictSlice(msgraphconv.NewVerifiedDomains(org.GetVerifiedDomains()))

		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.organization",
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

func (a *mqlMsgraphGroup) id() (string, error) {
	return a.Id()
}

func (a *mqlMsgraphGroup) GetMembers() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (m *mqlMsgraph) GetGroups() ([]interface{}, error) {
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
		return nil, msgraphclient.TransformODataError(err)
	}

	res := []interface{}{}
	grps := resp.GetValue()
	for _, grp := range grps {
		graphGrp, err := m.MotorRuntime.CreateResource("msgraph.group",
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

func (m *mqlMsgraphUser) id() (string, error) {
	return m.Id()
}

func (m *mqlMsgraph) GetUsers() ([]interface{}, error) {
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
	resp, err := graphClient.Users().Get(ctx, &users.UsersRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphclient.TransformODataError(err)
	}

	res := []interface{}{}
	users := resp.GetValue()
	for i := range users {
		user := users[i]

		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.user",
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

func (m *mqlMsgraphServiceprincipal) id() (string, error) {
	return m.Id()
}

func (m *mqlMsgraph) GetServiceprincipals() ([]interface{}, error) {
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
		return nil, msgraphclient.TransformODataError(err)
	}

	res := []interface{}{}
	sps := resp.GetValue()
	for _, sp := range sps {
		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.serviceprincipal",
			"id", core.ToString(sp.GetId()),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

func (m *mqlMsgraphDomain) id() (string, error) {
	return m.Id()
}

func (m *mqlMsgraph) GetDomains() ([]interface{}, error) {
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
		return nil, msgraphclient.TransformODataError(err)
	}

	res := []interface{}{}
	domains := resp.GetValue()
	for i := range domains {
		domain := domains[i]

		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.domain",
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

func (m *mqlMsgraphDomaindnsrecord) id() (string, error) {
	return m.Id()
}

func (m *mqlMsgraphDomain) GetServiceConfigurationRecords() ([]interface{}, error) {
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
	resp, err := graphClient.DomainsById(id).ServiceConfigurationRecords().Get(ctx, &domains.DomainsItemServiceConfigurationRecordsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphclient.TransformODataError(err)
	}

	res := []interface{}{}
	records := resp.GetValue()
	for i := range records {
		record := records[i]

		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.domaindnsrecord",
			"id", core.ToString(record.GetId()),
			"isOptional", core.ToBool(record.GetIsOptional()),
			"label", core.ToString(record.GetLabel()),
			"recordType", core.ToString(record.GetRecordType()),
			"supportedService", core.ToString(record.GetSupportedService()),
			"ttl", core.ToInt64From32(record.GetTtl()),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

func (m *mqlMsgraphApplication) id() (string, error) {
	return m.Id()
}

func (m *mqlMsgraph) GetApplications() ([]interface{}, error) {
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
		return nil, msgraphclient.TransformODataError(err)
	}

	res := []interface{}{}
	apps := resp.GetValue()
	for i := range apps {
		app := apps[i]

		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.application",
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

func (m *mqlMsgraphUser) GetSettings() (interface{}, error) {
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
	userSettings, err := graphClient.UsersById(id).Settings().Get(ctx, &users.UsersItemSettingsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphclient.TransformODataError(err)
	}

	return core.JsonToDict(msgraphconv.NewUserSettings(userSettings))
}

func (m *mqlMsgraphSecurity) id() (string, error) {
	return "msgraph.security", nil
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

	mqlResource, err := runtime.CreateResource("msgraph.security.securityscore",
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

func (m *mqlMsgraphSecurity) GetLatestSecureScores() (interface{}, error) {
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
	resp, err := graphClient.Security().SecureScores().Get(ctx, &security.SecuritySecureScoresRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphclient.TransformODataError(err)
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
func (m *mqlMsgraphSecurity) GetSecureScores() ([]interface{}, error) {
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
	resp, err := graphClient.Security().SecureScores().Get(ctx, &security.SecuritySecureScoresRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphclient.TransformODataError(err)
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

func (s *mqlMsgraphSecuritySecurityscore) id() (string, error) {
	return s.Id()
}

func (s *mqlMsgraphPolicies) id() (string, error) {
	return "msgraph.policies", nil
}

func (m *mqlMsgraphPolicies) GetAuthorizationPolicy() (interface{}, error) {
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
	resp, err := graphClient.Policies().AuthorizationPolicy().Get(ctx, &policies.PoliciesAuthorizationPolicyRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphclient.TransformODataError(err)
	}

	return core.JsonToDict(msgraphconv.NewAuthorizationPolicy(resp))
}

func (m *mqlMsgraphPolicies) GetIdentitySecurityDefaultsEnforcementPolicy() (interface{}, error) {
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
	policy, err := graphClient.Policies().IdentitySecurityDefaultsEnforcementPolicy().Get(ctx, &policies.PoliciesIdentitySecurityDefaultsEnforcementPolicyRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphclient.TransformODataError(err)
	}

	return core.JsonToDict(msgraphconv.NewIdentitySecurityDefaultsEnforcementPolicy(policy))
}

// https://docs.microsoft.com/en-us/graph/api/adminconsentrequestpolicy-get?view=graph-rest-
func (m *mqlMsgraphPolicies) GetAdminConsentRequestPolicy() (interface{}, error) {
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
	policy, err := graphClient.Policies().AdminConsentRequestPolicy().Get(ctx, &policies.PoliciesAdminConsentRequestPolicyRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphclient.TransformODataError(err)
	}
	return core.JsonToDict(msgraphconv.NewAdminConsentRequestPolicy(policy))
}

// https://docs.microsoft.com/en-us/azure/active-directory/manage-apps/configure-user-consent?tabs=azure-powershell
// https://docs.microsoft.com/en-us/graph/api/permissiongrantpolicy-list?view=graph-rest-1.0&tabs=http
func (m *mqlMsgraphPolicies) GetPermissionGrantPolicies() (interface{}, error) {
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
	resp, err := graphClient.Policies().PermissionGrantPolicies().Get(ctx, &policies.PoliciesPermissionGrantPoliciesRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphclient.TransformODataError(err)
	}
	return core.JsonToDictSlice(msgraphconv.NewPermissionGrantPolicies(resp.GetValue()))
}

func (m *mqlMsgraphRolemanagement) id() (string, error) {
	return "msgraph.rolemanagement", nil
}

func (m *mqlMsgraphRolemanagement) GetRoleDefinitions() (interface{}, error) {
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
	resp, err := graphClient.RoleManagement().Directory().RoleDefinitions().Get(ctx, &rolemanagement.RoleManagementDirectoryRoleDefinitionsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphclient.TransformODataError(err)
	}

	res := []interface{}{}
	roles := resp.GetValue()
	for i := range roles {
		role := roles[i]

		rolePermissions, _ := core.JsonToDictSlice(msgraphconv.NewUnifiedRolePermissions(role.GetRolePermissions()))

		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.rolemanagement.roledefinition",
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

func (m *mqlMsgraphRolemanagementRoledefinition) id() (string, error) {
	return m.Id()
}

func (m *mqlMsgraphRolemanagementRoledefinition) GetAssignments() ([]interface{}, error) {
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
	requestConfig := &rolemanagement.RoleManagementDirectoryRoleAssignmentsRequestBuilderGetRequestConfiguration{
		QueryParameters: &rolemanagement.RoleManagementDirectoryRoleAssignmentsRequestBuilderGetQueryParameters{
			Filter: &filter,
			Expand: []string{"principal"},
		},
	}
	resp, err := graphClient.RoleManagement().Directory().RoleAssignments().Get(ctx, requestConfig)
	if err != nil {
		return nil, msgraphclient.TransformODataError(err)
	}

	roleAssignments := resp.GetValue()

	res := []interface{}{}
	for i := range roleAssignments {
		roleAssignment := roleAssignments[i]
		principal, _ := core.JsonToDict(msgraphconv.NewDirectoryPrincipal(roleAssignment.GetPrincipal()))
		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.rolemanagement.roleassignment",
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

func (m *mqlMsgraphRolemanagementRoleassignment) id() (string, error) {
	return m.Id()
}

func (m *mqlMsgraphDevicemanagement) id() (string, error) {
	return "msgraph.devicemanagement", nil
}

func (m *mqlMsgraphDevicemanagement) GetDeviceConfigurations() ([]interface{}, error) {
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
	resp, err := graphClient.DeviceManagement().DeviceConfigurations().Get(ctx, &devicemanagement.DeviceManagementDeviceConfigurationsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, msgraphclient.TransformODataError(err)
	}

	res := []interface{}{}
	configurations := resp.GetValue()
	for i := range configurations {
		configuration := configurations[i]
		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.devicemanagement.deviceconfiguration",
			"id", core.ToString(configuration.GetId()),
			"lastModifiedDateTime", configuration.GetLastModifiedDateTime(),
			"createdDateTime", configuration.GetCreatedDateTime(),
			"description", core.ToString(configuration.GetDescription()),
			"displayName", core.ToString(configuration.GetDisplayName()),
			"version", core.ToInt64From32(configuration.GetVersion()),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

func (m *mqlMsgraphDevicemanagement) GetDeviceCompliancePolicies() ([]interface{}, error) {
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
	requestConfig := &devicemanagement.DeviceManagementDeviceCompliancePoliciesRequestBuilderGetRequestConfiguration{
		QueryParameters: &devicemanagement.DeviceManagementDeviceCompliancePoliciesRequestBuilderGetQueryParameters{
			Expand: []string{"assignments"},
		},
	}
	resp, err := graphClient.DeviceManagement().DeviceCompliancePolicies().Get(ctx, requestConfig)
	if err != nil {
		return nil, msgraphclient.TransformODataError(err)
	}

	compliancePolicies := resp.GetValue()
	res := []interface{}{}
	for i := range compliancePolicies {
		compliancePolicy := compliancePolicies[i]

		assignments, _ := core.JsonToDictSlice(msgraphconv.NewDeviceCompliancePolicyAssignments(compliancePolicy.GetAssignments()))

		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.devicemanagement.devicecompliancepolicy",
			"id", core.ToString(compliancePolicy.GetId()),
			"createdDateTime", compliancePolicy.GetCreatedDateTime(),
			"description", core.ToString(compliancePolicy.GetDescription()),
			"displayName", core.ToString(compliancePolicy.GetDisplayName()),
			"lastModifiedDateTime", compliancePolicy.GetLastModifiedDateTime(),
			"version", core.ToInt64From32(compliancePolicy.GetVersion()),
			"assignments", assignments,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

func (m *mqlMsgraphDevicemanagementDeviceconfiguration) id() (string, error) {
	return m.Id()
}

func (m *mqlMsgraphDevicemanagementDevicecompliancepolicy) id() (string, error) {
	return m.Id()
}

func graphClient(t *microsoft.Provider) (*msgraphclient.GraphServiceClient, error) {
	auth, err := t.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	providerFunc := func() (authentication.AuthenticationProvider, error) {
		return a.NewAzureIdentityAuthenticationProviderWithScopes(auth, msgraphclient.DefaultMSGraphScopes)
	}
	adapter, err := msgraphclient.NewGraphRequestAdapterWithFn(providerFunc)
	if err != nil {
		return nil, err
	}
	graphClient := msgraphclient.NewGraphServiceClient(adapter)
	return graphClient, nil
}
