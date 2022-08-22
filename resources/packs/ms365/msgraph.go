package ms365

import (
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/devicemanagement/devicecompliancepolicies"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/models"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/rolemanagement/directory/roleassignments"
	"go.mondoo.io/mondoo/motor/providers"
	ms365_provider "go.mondoo.io/mondoo/motor/providers/ms365"
	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/packs/core"
	"go.mondoo.io/mondoo/resources/packs/ms365/msgraphclient"
	"go.mondoo.io/mondoo/resources/packs/ms365/msgraphconv"
)

func ms365Provider(t providers.Instance) (*ms365_provider.Provider, error) {
	at, ok := t.(*ms365_provider.Provider)
	if !ok {
		return nil, errors.New("ms365 resource is not supported on this transport")
	}
	return at, nil
}

func (m *mqlMsgraphBeta) id() (string, error) {
	return "msgraph.beta", nil
}

func (m *mqlMsgraphBeta) GetSettings() ([]interface{}, error) {
	provider, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	graphBetaClient, err := graphBetaClient(provider)
	if err != nil {
		return nil, err
	}

	settings, err := graphBetaClient.Settings().Get()
	if err != nil {
		return nil, err
	}
	return core.JsonToDictSlice(msgraphconv.NewDirectorySettings(settings.GetValue()))
}

func (m *mqlMsgraphBetaOrganization) id() (string, error) {
	return m.Id()
}

func (m *mqlMsgraphBeta) GetOrganizations() ([]interface{}, error) {
	provider, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Organization.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(provider)
	if err != nil {
		return nil, err
	}

	resp, err := graphBetaClient.Organization().Get()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	orgs := resp.GetValue()
	for i := range orgs {
		org := orgs[i]

		assignedPlans, _ := core.JsonToDictSlice(msgraphconv.NewAssignedPlans(org.GetAssignedPlans()))
		verifiedDomains, _ := core.JsonToDictSlice(msgraphconv.NewVerifiedDomains(org.GetVerifiedDomains()))

		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.beta.organization",
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

func (m *mqlMsgraphBetaUser) id() (string, error) {
	return m.Id()
}

func (m *mqlMsgraphBeta) GetUsers() ([]interface{}, error) {
	provider, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("User.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(provider)
	if err != nil {
		return nil, err
	}

	resp, err := graphBetaClient.Users().Get()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	users := resp.GetValue()
	for i := range users {
		user := users[i]

		settings, _ := core.JsonToDict(user.GetSettings())
		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.beta.user",
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
			"settings", settings,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

func (m *mqlMsgraphBetaDomain) id() (string, error) {
	return m.Id()
}

func (m *mqlMsgraphBeta) GetDomains() ([]interface{}, error) {
	provider, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Domain.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(provider)
	if err != nil {
		return nil, err
	}

	resp, err := graphBetaClient.Domains().Get()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	domains := resp.GetValue()
	for i := range domains {
		domain := domains[i]

		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.beta.domain",
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

func (m *mqlMsgraphBetaDomaindnsrecord) id() (string, error) {
	return m.Id()
}

func (m *mqlMsgraphBetaDomain) GetServiceConfigurationRecords() ([]interface{}, error) {
	provider, err := ms365Provider(m.MotorRuntime.Motor.Provider)
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

	graphBetaClient, err := graphBetaClient(provider)
	if err != nil {
		return nil, err
	}

	resp, err := graphBetaClient.DomainsById(id).ServiceConfigurationRecords().Get()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	records := resp.GetValue()
	for i := range records {
		record := records[i]

		// TODO: do not return additional data, it is used to gather the text
		properties, _ := core.JsonToDict(record.GetAdditionalData())

		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.beta.domaindnsrecord",
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

func (m *mqlMsgraphBetaApplication) id() (string, error) {
	return m.Id()
}

func (m *mqlMsgraphBeta) GetApplications() ([]interface{}, error) {
	provider, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Application.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(provider)
	if err != nil {
		return nil, err
	}

	resp, err := graphBetaClient.Applications().Get()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	apps := resp.GetValue()
	for i := range apps {
		app := apps[i]

		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.beta.application",
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

func (m *mqlMsgraphBetaUser) GetSettings() (interface{}, error) {
	id, err := m.Id()
	if err != nil {
		return nil, err
	}

	provider, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("User.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(provider)
	if err != nil {
		return nil, err
	}

	userSettings, err := graphBetaClient.UsersById(id).Settings().Get()
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(msgraphconv.NewUserSettings(userSettings))
}

func (m *mqlMsgraphBetaSecurity) id() (string, error) {
	return "msgraph.beta.security", nil
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

	mqlResource, err := runtime.CreateResource("msgraph.beta.security.securityscore",
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

func (m *mqlMsgraphBetaSecurity) GetLatestSecureScores() (interface{}, error) {
	provider, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("SecurityEvents.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(provider)
	if err != nil {
		return nil, err
	}

	resp, err := graphBetaClient.Security().SecureScores().Get()
	if err != nil {
		return nil, err
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
func (m *mqlMsgraphBetaSecurity) GetSecureScores() ([]interface{}, error) {
	provider, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("SecurityEvents.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(provider)
	if err != nil {
		return nil, err
	}

	resp, err := graphBetaClient.Security().SecureScores().Get()
	if err != nil {
		return nil, err
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

func (s *mqlMsgraphBetaSecuritySecurityscore) id() (string, error) {
	return s.Id()
}

func (s *mqlMsgraphBetaPolicies) id() (string, error) {
	return "msgraph.beta.policies", nil
}

func (m *mqlMsgraphBetaPolicies) GetAuthorizationPolicy() (interface{}, error) {
	provider, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Policy.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(provider)
	if err != nil {
		return nil, err
	}

	resp, err := graphBetaClient.Policies().AuthorizationPolicy().Get()
	if err != nil {
		return nil, err
	}

	policies := resp.GetValue()
	if len(policies) > 0 {
		// TODO: we need to change the MQL resource to return more than one
		return core.JsonToDict(msgraphconv.NewAuthorizationPolicy(policies[0]))
	}
	return nil, nil
}

func (m *mqlMsgraphBetaPolicies) GetIdentitySecurityDefaultsEnforcementPolicy() (interface{}, error) {
	provider, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Policy.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(provider)
	if err != nil {
		return nil, err
	}

	policy, err := graphBetaClient.Policies().IdentitySecurityDefaultsEnforcementPolicy().Get()
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(msgraphconv.NewIdentitySecurityDefaultsEnforcementPolicy(policy))
}

// https://docs.microsoft.com/en-us/graph/api/adminconsentrequestpolicy-get?view=graph-rest-beta
func (m *mqlMsgraphBetaPolicies) GetAdminConsentRequestPolicy() (interface{}, error) {
	provider, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Policy.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(provider)
	if err != nil {
		return nil, err
	}

	policy, err := graphBetaClient.Policies().AdminConsentRequestPolicy().Get()
	if err != nil {
		return nil, err
	}
	return core.JsonToDict(msgraphconv.NewAdminConsentRequestPolicy(policy))
}

// https://docs.microsoft.com/en-us/azure/active-directory/manage-apps/configure-user-consent?tabs=azure-powershell
// https://docs.microsoft.com/en-us/graph/api/permissiongrantpolicy-list?view=graph-rest-1.0&tabs=http
func (m *mqlMsgraphBetaPolicies) GetPermissionGrantPolicies() (interface{}, error) {
	provider, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Policy.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(provider)
	if err != nil {
		return nil, err
	}

	resp, err := graphBetaClient.Policies().PermissionGrantPolicies().Get()
	if err != nil {
		return nil, err
	}
	return core.JsonToDictSlice(msgraphconv.NewPermissionGrantPolicys(resp.GetValue()))
}

func (m *mqlMsgraphBetaRolemanagement) id() (string, error) {
	return "msgraph.rolemanagement", nil
}

func (m *mqlMsgraphBetaRolemanagement) GetRoleDefinitions() (interface{}, error) {
	provider, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Directory.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(provider)
	if err != nil {
		return nil, err
	}

	resp, err := graphBetaClient.RoleManagement().Directory().RoleDefinitions().Get()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	roles := resp.GetValue()
	for i := range roles {
		role := roles[i]

		rolePermissions, _ := core.JsonToDictSlice(msgraphconv.NewUnifiedRolePermissions(role.GetRolePermissions()))

		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.beta.rolemanagement.roledefinition",
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

func (m *mqlMsgraphBetaRolemanagementRoledefinition) id() (string, error) {
	return m.Id()
}

func (m *mqlMsgraphBetaRolemanagementRoledefinition) GetAssignments() ([]interface{}, error) {
	provider, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("Directory.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(provider)
	if err != nil {
		return nil, err
	}

	roleDefinitionID, err := m.Id()
	if err != nil {
		return nil, err
	}
	filter := "roleDefinitionId eq '" + roleDefinitionID + "'"

	resp, err := graphBetaClient.RoleManagement().Directory().RoleAssignments().
		GetWithRequestConfigurationAndResponseHandler(&roleassignments.RoleAssignmentsRequestBuilderGetRequestConfiguration{
			QueryParameters: &roleassignments.RoleAssignmentsRequestBuilderGetQueryParameters{
				Filter: &filter,
				Expand: []string{"principal"},
			},
		}, nil)

	roleAssignments := resp.GetValue()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range roleAssignments {
		roleAssignment := roleAssignments[i]
		principal, _ := core.JsonToDict(msgraphconv.NewDirectoryPricipal(roleAssignment.GetPrincipal()))
		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.beta.rolemanagement.roleassignment",
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

func (m *mqlMsgraphBetaRolemanagementRoleassignment) id() (string, error) {
	return m.Id()
}

func (m *mqlMsgraphBetaDevicemanagement) id() (string, error) {
	return "msgraph.beta.devicemanagement", nil
}

func (m *mqlMsgraphBetaDevicemanagement) GetDeviceConfigurations() ([]interface{}, error) {
	provider, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("DeviceManagementConfiguration.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(provider)
	if err != nil {
		return nil, err
	}

	resp, err := graphBetaClient.DeviceManagement().DeviceConfigurations().Get()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	configurations := resp.GetValue()
	for i := range configurations {
		configuration := configurations[i]
		// TODO: do not return additional data
		properties, _ := core.JsonToDict(configuration.GetAdditionalData())
		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.beta.devicemanagement.deviceconfiguration",
			"id", core.ToString(configuration.GetId()),
			"lastModifiedDateTime", configuration.GetLastModifiedDateTime(),
			"roleScopeTagIds", core.StrSliceToInterface(configuration.GetRoleScopeTagIds()),
			"supportsScopeTags", core.ToBool(configuration.GetSupportsScopeTags()),
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

func (m *mqlMsgraphBetaDevicemanagement) GetDeviceCompliancePolicies() ([]interface{}, error) {
	provider, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	missingPermissions := provider.MissingRoles("DeviceManagementConfiguration.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(provider)
	if err != nil {
		return nil, err
	}

	resp, err := graphBetaClient.DeviceManagement().DeviceCompliancePolicies().
		GetWithRequestConfigurationAndResponseHandler(&devicecompliancepolicies.DeviceCompliancePoliciesRequestBuilderGetRequestConfiguration{
			QueryParameters: &devicecompliancepolicies.DeviceCompliancePoliciesRequestBuilderGetQueryParameters{
				Expand: []string{"assignments"},
			},
		}, nil)
	if err != nil {
		return nil, err
	}

	compliancePolicies := resp.GetValue()
	res := []interface{}{}
	for i := range compliancePolicies {
		compliancePolicy := compliancePolicies[i]

		// TODO: revisit if we really need to expose the additional data
		// expose the struct better
		properties, _ := core.JsonToDict(compliancePolicy.GetAdditionalData())
		assignments, _ := core.JsonToDictSlice(msgraphconv.NewDeviceCompliancePolicyAssignments(compliancePolicy.GetAssignments()))

		mqlResource, err := m.MotorRuntime.CreateResource("msgraph.beta.devicemanagement.devicecompliancepolicy",
			"id", core.ToString(compliancePolicy.GetId()),
			"createdDateTime", compliancePolicy.GetCreatedDateTime(),
			"description", core.ToString(compliancePolicy.GetDescription()),
			"displayName", core.ToString(compliancePolicy.GetDisplayName()),
			"lastModifiedDateTime", compliancePolicy.GetLastModifiedDateTime(),
			"roleScopeTagIds", core.StrSliceToInterface(compliancePolicy.GetRoleScopeTagIds()),
			"version", core.ToInt64From32(compliancePolicy.GetVersion()),
			"properties", properties,
			"assignments", assignments,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

func (m *mqlMsgraphBetaDevicemanagementDeviceconfiguration) id() (string, error) {
	return m.Id()
}

func (m *mqlMsgraphBetaDevicemanagementDevicecompliancepolicy) id() (string, error) {
	return m.Id()
}

func graphBetaAdapter(t *ms365_provider.Provider) (*msgraphclient.GraphRequestAdapter, error) {
	auth, err := t.Auth()
	if err != nil {
		return nil, errors.Wrap(err, "authentication provider error")
	}

	adapter, err := msgraphclient.NewGraphRequestAdapter(auth)
	if err != nil {
		return nil, err
	}
	return adapter, nil
}

func graphBetaClient(t *ms365_provider.Provider) (*msgraphclient.GraphServiceClient, error) {
	adapter, err := graphBetaAdapter(t)
	if err != nil {
		return nil, err
	}
	graphBetaClient := msgraphclient.NewGraphServiceClient(adapter)
	return graphBetaClient, nil
}
