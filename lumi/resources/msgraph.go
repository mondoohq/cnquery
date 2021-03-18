package resources

import (
	"context"
	"errors"
	"strings"

	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/transports"
	ms365_transport "go.mondoo.io/mondoo/motor/transports/ms365"
)

func ms365transport(t transports.Transport) (*ms365_transport.Transport, error) {
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
	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	graphBetaClient, err := mt.GraphBetaClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	settings, err := graphBetaClient.Settings().Request().Get(ctx)
	if err != nil {
		return nil, err
	}
	return jsonToDictSlice(settings)
}

func (m *lumiMsgraphBeta) GetOrganizations() ([]interface{}, error) {
	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("Organization.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := mt.GraphBetaClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	orgs, err := graphBetaClient.Organization().Request().Get(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range orgs {
		org := orgs[i]

		assignedPlans, _ := jsonToDictSlice(org.AssignedPlans)
		verifiedDomains, _ := jsonToDictSlice(org.VerifiedDomains)

		lumiResource, err := m.Runtime.CreateResource("msgraph.beta.organization",
			"id", toString(org.ID),
			"assignedPlans", assignedPlans,
			"createdDateTime", org.CreatedDateTime,
			"displayName", toString(org.DisplayName),
			"verifiedDomains", verifiedDomains,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiResource)
	}

	return res, nil
}

func (m *lumiMsgraphBeta) GetUsers() ([]interface{}, error) {
	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("User.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := mt.GraphBetaClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	users, err := graphBetaClient.Users().Request().Get(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range users {
		user := users[i]

		settings, _ := jsonToDict(user.Settings)
		lumiResource, err := m.Runtime.CreateResource("msgraph.beta.user",
			"id", toString(user.ID),
			"accountEnabled", toBool(user.AccountEnabled),
			"city", toString(user.City),
			"companyName", toString(user.CompanyName),
			"country", toString(user.Country),
			"createdDateTime", user.CreatedDateTime,
			"department", toString(user.Department),
			"displayName", toString(user.DisplayName),
			"employeeId", toString(user.EmployeeID),
			"givenName", toString(user.GivenName),
			"jobTitle", toString(user.JobTitle),
			"mail", toString(user.Mail),
			"mobilePhone", toString(user.MobilePhone),
			"otherMails", strSliceToInterface(user.OtherMails),
			"officeLocation", toString(user.OfficeLocation),
			"postalCode", toString(user.PostalCode),
			"state", toString(user.State),
			"streetAddress", toString(user.StreetAddress),
			"surname", toString(user.Surname),
			"userPrincipalName", toString(user.UserPrincipalName),
			"userType", toString(user.UserType),
			"settings", settings,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiResource)
	}

	return res, nil
}

func (m *lumiMsgraphBeta) GetDomains() ([]interface{}, error) {
	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("Domain.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := mt.GraphBetaClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	domains, err := graphBetaClient.Domains().Request().Get(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range domains {
		domain := domains[i]

		lumiResource, err := m.Runtime.CreateResource("msgraph.beta.domain",
			"id", toString(domain.ID),
			"authenticationType", toString(domain.AuthenticationType),
			"availabilityStatus", toString(domain.AvailabilityStatus),
			"isAdminManaged", toBool(domain.IsAdminManaged),
			"isDefault", toBool(domain.IsDefault),
			"isInitial", toBool(domain.IsInitial),
			"isRoot", toBool(domain.IsRoot),
			"isVerified", toBool(domain.IsVerified),
			"passwordNotificationWindowInDays", toInt(domain.PasswordNotificationWindowInDays),
			"passwordValidityPeriodInDays", toInt(domain.PasswordValidityPeriodInDays),
			"supportedServices", sliceInterface(domain.SupportedServices),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiResource)
	}

	return res, nil
}

func (m *lumiMsgraphBeta) GetApplications() ([]interface{}, error) {
	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("Application.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := mt.GraphBetaClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	apps, err := graphBetaClient.Applications().Request().Get(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range apps {
		app := apps[i]

		lumiResource, err := m.Runtime.CreateResource("msgraph.beta.application",
			"id", toString(app.ID),
			"appId", toString(app.AppID),
			"createdDateTime", app.CreatedDateTime,
			"identifierUris", strSliceToInterface(app.IdentifierUris),
			"displayName", toString(app.DisplayName),
			"publisherDomain", toString(app.PublisherDomain),
			"signInAudience", toString(app.SignInAudience),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiResource)
	}

	return res, nil
}

func (m *lumiMsgraphBetaOrganization) id() (string, error) {
	return m.Id()
}

func (m *lumiMsgraphBetaUser) id() (string, error) {
	return m.Id()
}

func (m *lumiMsgraphBetaDomain) id() (string, error) {
	return m.Id()
}

func (m *lumiMsgraphBetaApplication) id() (string, error) {
	return m.Id()
}

func (m *lumiMsgraphBetaUser) GetSettings() (interface{}, error) {
	id, err := m.Id()
	if err != nil {
		return nil, err
	}

	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("User.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := mt.GraphBetaClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	userSettings, err := graphBetaClient.Users().ID(id).Settings().Request().Get(ctx)
	if err != nil {
		return nil, err
	}
	return jsonToDict(userSettings)
}

func (m *lumiMsgraphBetaSecurity) id() (string, error) {
	return "msgraph.beta.security", nil
}

func msSecureScoreToLumi(runtime *lumi.Runtime, score msgraphbeta.SecureScore) (interface{}, error) {
	averageComparativeScores := []interface{}{}
	for j := range score.AverageComparativeScores {
		entry, err := jsonToDict(score.AverageComparativeScores[j])
		if err != nil {
			return nil, err
		}
		averageComparativeScores = append(averageComparativeScores, entry)
	}

	controlScores := []interface{}{}
	for j := range score.ControlScores {
		entry, err := jsonToDict(score.ControlScores[j])
		if err != nil {
			return nil, err
		}
		controlScores = append(controlScores, entry)
	}

	vendorInformation, err := jsonToDict(score.VendorInformation)
	if err != nil {
		return nil, err
	}

	lumiResource, err := runtime.CreateResource("msgraph.beta.security.securityscore",
		"id", toString(score.ID),
		"activeUserCount", toInt(score.ActiveUserCount),
		"averageComparativeScores", averageComparativeScores,
		"azureTenantId", toString(score.AzureTenantID),
		"controlScores", controlScores,
		"createdDateTime", score.CreatedDateTime,
		"currentScore", toFloat64(score.CurrentScore),
		"enabledServices", strSliceToInterface(score.EnabledServices),
		"licensedUserCount", toInt(score.LicensedUserCount),
		"maxScore", toFloat64(score.MaxScore),
		"vendorInformation", vendorInformation,
	)
	if err != nil {
		return nil, err
	}
	return lumiResource, nil
}

func (m *lumiMsgraphBetaSecurity) GetLatestSecureScores() (interface{}, error) {
	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("SecurityEvents.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := mt.GraphBetaClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	scores, err := graphBetaClient.Security().SecureScores().Request().Get(ctx)
	if err != nil {
		return nil, err
	}

	if len(scores) == 0 {
		return nil, errors.New("could not retrieve any score")
	}

	latestScore := scores[0]
	for i := range scores {
		score := scores[i]
		if score.CreatedDateTime != nil && (latestScore.CreatedDateTime == nil || score.CreatedDateTime.Before(*latestScore.CreatedDateTime)) {
			latestScore = score
		}
	}

	return msSecureScoreToLumi(m.Runtime, latestScore)
}

// see https://docs.microsoft.com/en-us/graph/api/securescore-get?view=graph-rest-1.0&tabs=http
func (m *lumiMsgraphBetaSecurity) GetSecureScores() ([]interface{}, error) {
	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("SecurityEvents.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := mt.GraphBetaClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	scores, err := graphBetaClient.Security().SecureScores().Request().Get(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range scores {
		score := scores[i]
		lumiResource, err := msSecureScoreToLumi(m.Runtime, score)
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

func (s *lumiMsgraphBetaProfiles) id() (string, error) {
	return "msgraph.beta.profiles", nil
}

func (m *lumiMsgraphBetaProfiles) GetAuthorizationPolicy() (interface{}, error) {
	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("Policy.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := mt.GraphBetaClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	policy, err := graphBetaClient.Policies().ID("authorizationPolicy").Request().Get(ctx)
	if err != nil {
		return nil, err
	}

	additionalDataRaw, ok := policy.AdditionalData["value"]
	if ok {
		additonalDataSlice, ok := additionalDataRaw.([]interface{})
		if ok && len(additonalDataSlice) > 0 {
			return jsonToDict(additonalDataSlice[0])
		}
	}
	return nil, nil
}

func (m *lumiMsgraphBetaProfiles) GetIdentitySecurityDefaultsEnforcementPolicy() (interface{}, error) {
	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("Policy.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := mt.GraphBetaClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	policy, err := graphBetaClient.Policies().ID("identitySecurityDefaultsEnforcementPolicy").Request().Get(ctx)
	if err != nil {
		return nil, err
	}
	return jsonToDict(policy.AdditionalData)
}

func (m *lumiMsgraphBetaRolemanagement) id() (string, error) {
	return "msgraph.rolemanagement", nil
}

func (m *lumiMsgraphBetaRolemanagement) GetRoleDefinitions() (interface{}, error) {
	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("Directory.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := mt.GraphBetaClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	roles, err := graphBetaClient.RoleManagement().Directory().RoleDefinitions().Request().Get(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range roles {
		role := roles[i]

		rolePermissions, _ := jsonToDictSlice(role.RolePermissions)

		lumiResource, err := m.Runtime.CreateResource("msgraph.beta.rolemanagement.roledefinition",
			"id", toString(role.ID),
			"description", toString(role.Description),
			"displayName", toString(role.DisplayName),
			"isBuiltIn", toBool(role.IsBuiltIn),
			"isEnabled", toBool(role.IsEnabled),
			"rolePermissions", rolePermissions,
			"templateId", toString(role.TemplateID),
			"version", toString(role.Version),
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
	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("Directory.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := mt.GraphBetaClient()
	if err != nil {
		return nil, err
	}

	roleDefinitionID, err := m.Id()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	r := graphBetaClient.RoleManagement().Directory().RoleAssignments().Request()
	r.Filter("roleDefinitionId eq '" + roleDefinitionID + "'")
	r.Expand("principal")
	roleAssignments, err := r.Get(ctx)

	res := []interface{}{}
	for i := range roleAssignments {
		roleAssignment := roleAssignments[i]

		principal, _ := jsonToDict(roleAssignment.Principal)

		lumiResource, err := m.Runtime.CreateResource("msgraph.beta.rolemanagement.roleassignment",
			"id", toString(roleAssignment.ID),
			"roleDefinitionId", toString(roleAssignment.RoleDefinitionID),
			"principalId", toString(roleAssignment.PrincipalID),
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
	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("DeviceManagementConfiguration.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := mt.GraphBetaClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	configurations, err := graphBetaClient.DeviceManagement().DeviceConfigurations().Request().Get(ctx)

	res := []interface{}{}
	for i := range configurations {
		configuration := configurations[i]

		properties, _ := jsonToDict(configuration.AdditionalData)

		lumiResource, err := m.Runtime.CreateResource("msgraph.beta.devicemanagement.deviceconfiguration",
			"id", toString(configuration.ID),
			"lastModifiedDateTime", configuration.LastModifiedDateTime,
			"roleScopeTagIds", sliceInterface(configuration.RoleScopeTagIDs),
			"supportsScopeTags", toBool(configuration.SupportsScopeTags),
			"createdDateTime", configuration.CreatedDateTime,
			"description", toString(configuration.Description),
			"displayName", toString(configuration.DisplayName),
			"version", toInt(configuration.Version),
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
	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	missingPermissions := mt.MissingRoles("DeviceManagementConfiguration.Read.All")
	if len(missingPermissions) > 0 {
		return nil, errors.New("current credentials have insufficient privileges: " + strings.Join(missingPermissions, ","))
	}

	graphBetaClient, err := mt.GraphBetaClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	r := graphBetaClient.DeviceManagement().DeviceCompliancePolicies().Request()
	r.Expand("assignments")
	compliancePolicies, err := r.Get(ctx)

	res := []interface{}{}
	for i := range compliancePolicies {
		compliancePolicy := compliancePolicies[i]

		properties, _ := jsonToDict(compliancePolicy.AdditionalData)
		assignments, _ := jsonToDictSlice(compliancePolicy.Assignments)

		lumiResource, err := m.Runtime.CreateResource("msgraph.beta.devicemanagement.devicecompliancepolicy",
			"id", toString(compliancePolicy.ID),
			"createdDateTime", compliancePolicy.CreatedDateTime,
			"description", toString(compliancePolicy.Description),
			"displayName", toString(compliancePolicy.DisplayName),
			"lastModifiedDateTime", compliancePolicy.LastModifiedDateTime,
			"roleScopeTagIds", sliceInterface(compliancePolicy.RoleScopeTagIDs),
			"version", toInt(compliancePolicy.Version),
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
