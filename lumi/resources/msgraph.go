package resources

import (
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/devicemanagement/devicecompliancepolicies"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/models"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/rolemanagement/directory/roleassignments"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/msgraphclient"
	"go.mondoo.io/mondoo/lumi/resources/msgraphconv"
	"go.mondoo.io/mondoo/motor/providers"
	ms365_transport "go.mondoo.io/mondoo/motor/providers/ms365"
)

func ms365transport(t providers.Transport) (*ms365_transport.Transport, error) {
	at, ok := t.(*ms365_transport.Transport)
	if !ok {
		return nil, errors.New("ms365 resource is not supported on this transport")
	}
	return at, nil
}

func (m *lumiMsgraphBeta) id() (string, error) {
	return "msgraph.beta", nil
}

func (m *lumiMsgraphBeta) GetSettings() ([]interface{}, error) {
	mt, err := ms365transport(m.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	graphBetaClient, err := graphBetaClient(mt)
	if err != nil {
		return nil, err
	}

	settings, err := graphBetaClient.Settings().Get()
	if err != nil {
		return nil, err
	}
	return jsonToDictSlice(msgraphconv.NewDirectorySettings(settings.GetValue()))
}

func (m *lumiMsgraphBetaOrganization) id() (string, error) {
	return m.Id()
}

func (m *lumiMsgraphBeta) GetOrganizations() ([]interface{}, error) {
	mt, err := ms365transport(m.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("Organization.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(mt)
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

		assignedPlans, _ := jsonToDictSlice(msgraphconv.NewAssignedPlans(org.GetAssignedPlans()))
		verifiedDomains, _ := jsonToDictSlice(msgraphconv.NewVerifiedDomains(org.GetVerifiedDomains()))

		lumiResource, err := m.MotorRuntime.CreateResource("msgraph.beta.organization",
			"id", toString(org.GetId()),
			"assignedPlans", assignedPlans,
			"createdDateTime", org.GetCreatedDateTime(),
			"displayName", toString(org.GetDisplayName()),
			"verifiedDomains", verifiedDomains,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiResource)
	}

	return res, nil
}

func (m *lumiMsgraphBetaUser) id() (string, error) {
	return m.Id()
}

func (m *lumiMsgraphBeta) GetUsers() ([]interface{}, error) {
	mt, err := ms365transport(m.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("User.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(mt)
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

		settings, _ := jsonToDict(user.GetSettings())
		lumiResource, err := m.MotorRuntime.CreateResource("msgraph.beta.user",
			"id", toString(user.GetId()),
			"accountEnabled", toBool(user.GetAccountEnabled()),
			"city", toString(user.GetCity()),
			"companyName", toString(user.GetCompanyName()),
			"country", toString(user.GetCountry()),
			"createdDateTime", user.GetCreatedDateTime(),
			"department", toString(user.GetDepartment()),
			"displayName", toString(user.GetDisplayName()),
			"employeeId", toString(user.GetEmployeeId()),
			"givenName", toString(user.GetGivenName()),
			"jobTitle", toString(user.GetJobTitle()),
			"mail", toString(user.GetMail()),
			"mobilePhone", toString(user.GetMobilePhone()),
			"otherMails", strSliceToInterface(user.GetOtherMails()),
			"officeLocation", toString(user.GetOfficeLocation()),
			"postalCode", toString(user.GetPostalCode()),
			"state", toString(user.GetState()),
			"streetAddress", toString(user.GetStreetAddress()),
			"surname", toString(user.GetSurname()),
			"userPrincipalName", toString(user.GetUserPrincipalName()),
			"userType", toString(user.GetUserType()),
			"settings", settings,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiResource)
	}

	return res, nil
}

func (m *lumiMsgraphBetaDomain) id() (string, error) {
	return m.Id()
}

func (m *lumiMsgraphBeta) GetDomains() ([]interface{}, error) {
	mt, err := ms365transport(m.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("Domain.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(mt)
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

		lumiResource, err := m.MotorRuntime.CreateResource("msgraph.beta.domain",
			"id", toString(domain.GetId()),
			"authenticationType", toString(domain.GetAuthenticationType()),
			"availabilityStatus", toString(domain.GetAvailabilityStatus()),
			"isAdminManaged", toBool(domain.GetIsAdminManaged()),
			"isDefault", toBool(domain.GetIsDefault()),
			"isInitial", toBool(domain.GetIsInitial()),
			"isRoot", toBool(domain.GetIsRoot()),
			"isVerified", toBool(domain.GetIsVerified()),
			"passwordNotificationWindowInDays", toInt64From32(domain.GetPasswordNotificationWindowInDays()),
			"passwordValidityPeriodInDays", toInt64From32(domain.GetPasswordValidityPeriodInDays()),
			"supportedServices", sliceInterface(domain.GetSupportedServices()),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiResource)
	}

	return res, nil
}

func (m *lumiMsgraphBetaDomaindnsrecord) id() (string, error) {
	return m.Id()
}

func (m *lumiMsgraphBetaDomain) GetServiceConfigurationRecords() ([]interface{}, error) {
	mt, err := ms365transport(m.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("Domain.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	id, err := m.Id()
	if err != nil {
		return nil, err
	}

	graphBetaClient, err := graphBetaClient(mt)
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
		properties, _ := jsonToDict(record.GetAdditionalData())

		lumiResource, err := m.MotorRuntime.CreateResource("msgraph.beta.domaindnsrecord",
			"id", toString(record.GetId()),
			"isOptional", toBool(record.GetIsOptional()),
			"label", toString(record.GetLabel()),
			"recordType", toString(record.GetRecordType()),
			"supportedService", toString(record.GetSupportedService()),
			"ttl", toInt64From32(record.GetTtl()),
			"properties", properties,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiResource)
	}

	return res, nil
}

func (m *lumiMsgraphBetaApplication) id() (string, error) {
	return m.Id()
}

func (m *lumiMsgraphBeta) GetApplications() ([]interface{}, error) {
	mt, err := ms365transport(m.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("Application.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(mt)
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

		lumiResource, err := m.MotorRuntime.CreateResource("msgraph.beta.application",
			"id", toString(app.GetId()),
			"appId", toString(app.GetAppId()),
			"createdDateTime", app.GetCreatedDateTime(),
			"identifierUris", strSliceToInterface(app.GetIdentifierUris()),
			"displayName", toString(app.GetDisplayName()),
			"publisherDomain", toString(app.GetPublisherDomain()),
			"signInAudience", toString(app.GetSignInAudience()),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiResource)
	}

	return res, nil
}

func (m *lumiMsgraphBetaUser) GetSettings() (interface{}, error) {
	id, err := m.Id()
	if err != nil {
		return nil, err
	}

	mt, err := ms365transport(m.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("User.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(mt)
	if err != nil {
		return nil, err
	}

	userSettings, err := graphBetaClient.UsersById(id).Settings().Get()
	if err != nil {
		return nil, err
	}

	return jsonToDict(msgraphconv.NewUserSettings(userSettings))
}

func (m *lumiMsgraphBetaSecurity) id() (string, error) {
	return "msgraph.beta.security", nil
}

func msSecureScoreToLumi(runtime *lumi.Runtime, score models.SecureScoreable) (interface{}, error) {
	if score == nil {
		return nil, nil
	}
	averageComparativeScores := []interface{}{}
	graphAverageComparativeScores := score.GetAverageComparativeScores()
	for j := range graphAverageComparativeScores {
		entry, err := jsonToDict(msgraphconv.NewAverageComparativeScore(graphAverageComparativeScores[j]))
		if err != nil {
			return nil, err
		}
		averageComparativeScores = append(averageComparativeScores, entry)
	}

	controlScores := []interface{}{}
	graphControlScores := score.GetControlScores()
	for j := range graphControlScores {
		entry, err := jsonToDict(msgraphconv.NewControlScore(graphControlScores[j]))
		if err != nil {
			return nil, err
		}
		controlScores = append(controlScores, entry)
	}

	vendorInformation, err := jsonToDict(msgraphconv.NewSecurityVendorInformation(score.GetVendorInformation()))
	if err != nil {
		return nil, err
	}

	lumiResource, err := runtime.CreateResource("msgraph.beta.security.securityscore",
		"id", toString(score.GetId()),
		"activeUserCount", toInt64From32(score.GetActiveUserCount()),
		"averageComparativeScores", averageComparativeScores,
		"azureTenantId", toString(score.GetAzureTenantId()),
		"controlScores", controlScores,
		"createdDateTime", score.GetCreatedDateTime(),
		"currentScore", toFloat64(score.GetCurrentScore()),
		"enabledServices", strSliceToInterface(score.GetEnabledServices()),
		"licensedUserCount", toInt64From32(score.GetLicensedUserCount()),
		"maxScore", toFloat64(score.GetMaxScore()),
		"vendorInformation", vendorInformation,
	)
	if err != nil {
		return nil, err
	}
	return lumiResource, nil
}

func (m *lumiMsgraphBetaSecurity) GetLatestSecureScores() (interface{}, error) {
	mt, err := ms365transport(m.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("SecurityEvents.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(mt)
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

	return msSecureScoreToLumi(m.MotorRuntime, latestScore)
}

// see https://docs.microsoft.com/en-us/graph/api/securescore-get?view=graph-rest-1.0&tabs=http
func (m *lumiMsgraphBetaSecurity) GetSecureScores() ([]interface{}, error) {
	mt, err := ms365transport(m.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("SecurityEvents.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(mt)
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
		lumiResource, err := msSecureScoreToLumi(m.MotorRuntime, score)
		if err != nil {
			return nil, err
		}

		res = append(res, lumiResource)
	}
	return res, nil
}

func (s *lumiMsgraphBetaSecuritySecurityscore) id() (string, error) {
	return s.Id()
}

func (s *lumiMsgraphBetaPolicies) id() (string, error) {
	return "msgraph.beta.policies", nil
}

func (m *lumiMsgraphBetaPolicies) GetAuthorizationPolicy() (interface{}, error) {
	mt, err := ms365transport(m.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("Policy.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(mt)
	if err != nil {
		return nil, err
	}

	resp, err := graphBetaClient.Policies().AuthorizationPolicy().Get()
	if err != nil {
		return nil, err
	}

	policies := resp.GetValue()
	if len(policies) > 0 {
		// TODO: we need to change the lumi resource to return more than one
		return jsonToDict(msgraphconv.NewAuthorizationPolicy(policies[0]))
	}
	return nil, nil
}

func (m *lumiMsgraphBetaPolicies) GetIdentitySecurityDefaultsEnforcementPolicy() (interface{}, error) {
	mt, err := ms365transport(m.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("Policy.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(mt)
	if err != nil {
		return nil, err
	}

	policy, err := graphBetaClient.Policies().IdentitySecurityDefaultsEnforcementPolicy().Get()
	if err != nil {
		return nil, err
	}

	return jsonToDict(msgraphconv.NewIdentitySecurityDefaultsEnforcementPolicy(policy))
}

// https://docs.microsoft.com/en-us/graph/api/adminconsentrequestpolicy-get?view=graph-rest-beta
func (m *lumiMsgraphBetaPolicies) GetAdminConsentRequestPolicy() (interface{}, error) {
	mt, err := ms365transport(m.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("Policy.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(mt)
	if err != nil {
		return nil, err
	}

	policy, err := graphBetaClient.Policies().AdminConsentRequestPolicy().Get()
	if err != nil {
		return nil, err
	}
	return jsonToDict(msgraphconv.NewAdminConsentRequestPolicy(policy))
}

// https://docs.microsoft.com/en-us/azure/active-directory/manage-apps/configure-user-consent?tabs=azure-powershell
// https://docs.microsoft.com/en-us/graph/api/permissiongrantpolicy-list?view=graph-rest-1.0&tabs=http
func (m *lumiMsgraphBetaPolicies) GetPermissionGrantPolicies() (interface{}, error) {
	mt, err := ms365transport(m.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("Policy.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(mt)
	if err != nil {
		return nil, err
	}

	resp, err := graphBetaClient.Policies().PermissionGrantPolicies().Get()
	if err != nil {
		return nil, err
	}
	return jsonToDictSlice(msgraphconv.NewPermissionGrantPolicys(resp.GetValue()))
}

func (m *lumiMsgraphBetaRolemanagement) id() (string, error) {
	return "msgraph.rolemanagement", nil
}

func (m *lumiMsgraphBetaRolemanagement) GetRoleDefinitions() (interface{}, error) {
	mt, err := ms365transport(m.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("Directory.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(mt)
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

		rolePermissions, _ := jsonToDictSlice(msgraphconv.NewUnifiedRolePermissions(role.GetRolePermissions()))

		lumiResource, err := m.MotorRuntime.CreateResource("msgraph.beta.rolemanagement.roledefinition",
			"id", toString(role.GetId()),
			"description", toString(role.GetDescription()),
			"displayName", toString(role.GetDisplayName()),
			"isBuiltIn", toBool(role.GetIsBuiltIn()),
			"isEnabled", toBool(role.GetIsEnabled()),
			"rolePermissions", rolePermissions,
			"templateId", toString(role.GetTemplateId()),
			"version", toString(role.GetVersion()),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiResource)
	}

	return res, nil
}

func (m *lumiMsgraphBetaRolemanagementRoledefinition) id() (string, error) {
	return m.Id()
}

func (m *lumiMsgraphBetaRolemanagementRoledefinition) GetAssignments() ([]interface{}, error) {
	mt, err := ms365transport(m.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("Directory.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(mt)
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
		principal, _ := jsonToDict(msgraphconv.NewDirectoryPricipal(roleAssignment.GetPrincipal()))
		lumiResource, err := m.MotorRuntime.CreateResource("msgraph.beta.rolemanagement.roleassignment",
			"id", toString(roleAssignment.GetId()),
			"roleDefinitionId", toString(roleAssignment.GetRoleDefinitionId()),
			"principalId", toString(roleAssignment.GetPrincipalId()),
			"principal", principal,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiResource)
	}

	return res, nil
}

func (m *lumiMsgraphBetaRolemanagementRoleassignment) id() (string, error) {
	return m.Id()
}

func (m *lumiMsgraphBetaDevicemanagement) id() (string, error) {
	return "msgraph.beta.devicemanagement", nil
}

func (m *lumiMsgraphBetaDevicemanagement) GetDeviceConfigurations() ([]interface{}, error) {
	mt, err := ms365transport(m.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("DeviceManagementConfiguration.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(mt)
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
		properties, _ := jsonToDict(configuration.GetAdditionalData())
		lumiResource, err := m.MotorRuntime.CreateResource("msgraph.beta.devicemanagement.deviceconfiguration",
			"id", toString(configuration.GetId()),
			"lastModifiedDateTime", configuration.GetLastModifiedDateTime(),
			"roleScopeTagIds", sliceInterface(configuration.GetRoleScopeTagIds()),
			"supportsScopeTags", toBool(configuration.GetSupportsScopeTags()),
			"createdDateTime", configuration.GetCreatedDateTime(),
			"description", toString(configuration.GetDescription()),
			"displayName", toString(configuration.GetDisplayName()),
			"version", toInt64From32(configuration.GetVersion()),
			"properties", properties,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiResource)
	}

	return res, nil
}

func (m *lumiMsgraphBetaDevicemanagement) GetDeviceCompliancePolicies() ([]interface{}, error) {
	mt, err := ms365transport(m.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("DeviceManagementConfiguration.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := graphBetaClient(mt)
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
		properties, _ := jsonToDict(compliancePolicy.GetAdditionalData())
		assignments, _ := jsonToDictSlice(msgraphconv.NewDeviceCompliancePolicyAssignments(compliancePolicy.GetAssignments()))

		lumiResource, err := m.MotorRuntime.CreateResource("msgraph.beta.devicemanagement.devicecompliancepolicy",
			"id", toString(compliancePolicy.GetId()),
			"createdDateTime", compliancePolicy.GetCreatedDateTime(),
			"description", toString(compliancePolicy.GetDescription()),
			"displayName", toString(compliancePolicy.GetDisplayName()),
			"lastModifiedDateTime", compliancePolicy.GetLastModifiedDateTime(),
			"roleScopeTagIds", sliceInterface(compliancePolicy.GetRoleScopeTagIds()),
			"version", toInt64From32(compliancePolicy.GetVersion()),
			"properties", properties,
			"assignments", assignments,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiResource)
	}

	return res, nil
}

func (m *lumiMsgraphBetaDevicemanagementDeviceconfiguration) id() (string, error) {
	return m.Id()
}

func (m *lumiMsgraphBetaDevicemanagementDevicecompliancepolicy) id() (string, error) {
	return m.Id()
}

func graphBetaAdapter(t *ms365_transport.Transport) (*msgraphclient.GraphRequestAdapter, error) {
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

func graphBetaClient(t *ms365_transport.Transport) (*msgraphclient.GraphServiceClient, error) {
	adapter, err := graphBetaAdapter(t)
	if err != nil {
		return nil, err
	}
	graphBetaClient := msgraphclient.NewGraphServiceClient(adapter)
	return graphBetaClient, nil
}
