// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

// this package creates a copy of the msgraph object that we use for embedded struct. This is required since microsoft
// defines structs with lower case and does not attach json tags or implements the standard marshalling function

import (
	"encoding/json"
	"time"

	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
)

type AssignedPlan struct {
	AssignedDateTime *time.Time `json:"assignedDateTime"`
	CapabilityStatus string     `json:"capabilityStatus"`
	Service          string     `json:"service"`
	ServicePlanId    string     `json:"servicePlanId"`
}

func newAssignedPlans(p []models.AssignedPlanable) []AssignedPlan {
	res := []AssignedPlan{}
	for i := range p {
		res = append(res, newAssignedPlan(p[i]))
	}
	return res
}

func newAssignedPlan(p models.AssignedPlanable) AssignedPlan {
	return AssignedPlan{
		AssignedDateTime: p.GetAssignedDateTime(),
		CapabilityStatus: convert.ToString(p.GetCapabilityStatus()),
		Service:          convert.ToString(p.GetService()),
		ServicePlanId:    p.GetServicePlanId().String(),
	}
}

type VerifiedDomain struct {
	Capabilities string `json:"capabilities"`
	IsDefault    bool   `json:"isDefault"`
	IsInitial    bool   `json:"isInitial"`
	Name         string `json:"name"`
	Type         string `json:"type"`
}

func newVerifiedDomains(p []models.VerifiedDomainable) []VerifiedDomain {
	res := []VerifiedDomain{}
	for i := range p {
		res = append(res, newVerifiedDomain(p[i]))
	}
	return res
}

func newVerifiedDomain(p models.VerifiedDomainable) VerifiedDomain {
	return VerifiedDomain{
		Capabilities: convert.ToString(p.GetCapabilities()),
		IsDefault:    convert.ToBool(p.GetIsDefault()),
		IsInitial:    convert.ToBool(p.GetIsInitial()),
		Name:         convert.ToString(p.GetName()),
		Type:         convert.ToString(p.GetTypeEscaped()),
	}
}

type UnifiedRolePermission struct {
	AllowedResourceActions  []string `json:"allowedResourceActions"`
	Condition               string   `json:"condition"`
	ExcludedResourceActions []string `json:"excludedResourceActions"`
}

func newUnifiedRolePermissions(p []models.UnifiedRolePermissionable) []UnifiedRolePermission {
	res := []UnifiedRolePermission{}
	for i := range p {
		res = append(res, newUnifiedRolePermission(p[i]))
	}
	return res
}

func newUnifiedRolePermission(p models.UnifiedRolePermissionable) UnifiedRolePermission {
	return UnifiedRolePermission{
		AllowedResourceActions:  p.GetAllowedResourceActions(),
		Condition:               convert.ToString(p.GetCondition()),
		ExcludedResourceActions: p.GetExcludedResourceActions(),
	}
}

type GroupSetting struct {
	DisplayName string         `json:"displayName"`
	TemplateId  string         `json:"templateId"`
	Values      []SettingValue `json:"values"`
}

type SettingValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func newSettings(p []models.GroupSettingable) []GroupSetting {
	res := []GroupSetting{}
	for i := range p {
		res = append(res, newSetting(p[i]))
	}
	return res
}

func newSetting(p models.GroupSettingable) GroupSetting {
	values := []SettingValue{}
	entries := p.GetValues()
	for i := range entries {
		values = append(values, SettingValue{
			Name:  convert.ToString(entries[i].GetName()),
			Value: convert.ToString(entries[i].GetValue()),
		})
	}

	return GroupSetting{
		DisplayName: convert.ToString(p.GetDisplayName()),
		TemplateId:  convert.ToString(p.GetTemplateId()),
		Values:      values,
	}
}

// structs for AuthorizationPolicy

type Entity struct {
	Id *string `json:"id"`
}

type DirectoryObject struct {
	Entity
	DeletedDateTime *time.Time `json:"deletedDateTime"`
}

func newDirectoryPrincipal(p models.DirectoryObjectable) *DirectoryObject {
	if p == nil {
		return nil
	}
	return &DirectoryObject{
		Entity: Entity{
			Id: p.GetId(),
		},
		DeletedDateTime: p.GetDeletedDateTime(),
	}
}

type PolicyBase struct {
	DirectoryObject
	// Description for this policy. Required.
	Description *string `json:"description"`
	// Display name for this policy. Required.
	DisplayName *string `json:"displayName"`
}

type AllowInvitesFrom int

func (a AllowInvitesFrom) String() string {
	return []string{"NONE", "ADMINSANDGUESTINVITERS", "ADMINSGUESTINVITERSANDALLMEMBERS", "EVERYONE", "UNKNOWNFUTUREVALUE"}[a]
}

func (a AllowInvitesFrom) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func newAllowInvitesFrom(a *models.AllowInvitesFrom) *AllowInvitesFrom {
	if a == nil {
		return nil
	}
	n := AllowInvitesFrom(int(*a))
	return &n
}

type DefaultUserRolePermissions struct {
	// Indicates whether the default user role can create applications.
	AllowedToCreateApps *bool `json:"allowedToCreateApps"`
	// Indicates whether the default user role can create security groups.
	AllowedToCreateSecurityGroups *bool `json:"allowedToCreateSecurityGroups"`
	// Indicates whether the default user role can read other users.
	AllowedToReadOtherUsers *bool `json:"allowedToReadOtherUsers"`
	// Indicates whether the default user role can create tenants.
	AllowedToCreateTenants *bool `json:"allowedToCreateTenants"`
}

func newDefaultUserRolePermissions(a models.DefaultUserRolePermissionsable) *DefaultUserRolePermissions {
	if a == nil {
		return nil
	}
	return &DefaultUserRolePermissions{
		AllowedToCreateApps:           a.GetAllowedToCreateApps(),
		AllowedToCreateSecurityGroups: a.GetAllowedToCreateSecurityGroups(),
		AllowedToReadOtherUsers:       a.GetAllowedToReadOtherUsers(),
		AllowedToCreateTenants:        a.GetAllowedToCreateTenants(),
	}
}

type AuthorizationPolicy struct {
	PolicyBase
	// Indicates whether users can sign up for email based subscriptions.
	AllowedToSignUpEmailBasedSubscriptions *bool `json:"allowedToSignUpEmailBasedSubscriptions"`
	// Indicates whether the Self-Serve Password Reset feature can be used by users on the tenant.
	AllowedToUseSSPR *bool `json:"allowedToUseSSPR"`
	// Indicates whether a user can join the tenant by email validation.
	AllowEmailVerifiedUsersToJoinOrganization *bool `json:"allowEmailVerifiedUsersToJoinOrganization"`
	// Indicates who can invite external users to the organization. Possible values are: none, adminsAndGuestInviters, adminsGuestInvitersAndAllMembers, everyone.  everyone is the default setting for all cloud environments except US Government. See more in the table below.
	AllowInvitesFrom *AllowInvitesFrom `json:"allowInvitesFrom"`
	// To disable the use of MSOL PowerShell set this property to true. This will also disable user-based access to the legacy service endpoint used by MSOL PowerShell. This does not affect Azure AD Connect or Microsoft Graph.
	BlockMsolPowerShell *bool `json:"blockMsolPowerShell"`
	//
	DefaultUserRolePermissions *DefaultUserRolePermissions `json:"defaultUserRolePermissions"`
	// Represents role templateId for the role that should be granted to guest user. Currently following roles are supported:  User (a0b1b346-4d3e-4e8b-98f8-753987be4970), Guest User (10dae51f-b6af-4016-8d66-8c2a99b929b3), and Restricted Guest User (2af84b1e-32c8-42b7-82bc-daa82404023b).
	GuestUserRoleId *string `json:"guestUserRoleId"`
}

func newAuthorizationPolicys(policies []models.AuthorizationPolicyable) []*AuthorizationPolicy {
	res := []*AuthorizationPolicy{}
	for i := range policies {
		res = append(res, newAuthorizationPolicy(policies[i]))
	}
	return res
}

func newAuthorizationPolicy(p models.AuthorizationPolicyable) *AuthorizationPolicy {
	if p == nil {
		return nil
	}

	var roleId string
	if p.GetGuestUserRoleId() != nil {
		roleId = p.GetGuestUserRoleId().String()
	}
	return &AuthorizationPolicy{
		AllowedToSignUpEmailBasedSubscriptions:    p.GetAllowedToSignUpEmailBasedSubscriptions(),
		AllowedToUseSSPR:                          p.GetAllowedToUseSSPR(),
		AllowEmailVerifiedUsersToJoinOrganization: p.GetAllowEmailVerifiedUsersToJoinOrganization(),
		AllowInvitesFrom:                          newAllowInvitesFrom(p.GetAllowInvitesFrom()),
		BlockMsolPowerShell:                       p.GetBlockMsolPowerShell(),
		DefaultUserRolePermissions:                newDefaultUserRolePermissions(p.GetDefaultUserRolePermissions()),
		GuestUserRoleId:                           &roleId,
	}
}

type AverageComparativeScore struct {
	// Average score within specified basis.
	AverageScore *float64 `json:"averageScore"`
	// Scope type. The possible values are: AllTenants, TotalSeats, IndustryTypes.
	Basis *string `json:"basis"`
}

func newAverageComparativeScore(p models.AverageComparativeScoreable) *AverageComparativeScore {
	if p == nil {
		return nil
	}
	return &AverageComparativeScore{
		AverageScore: p.GetAverageScore(),
		Basis:        p.GetBasis(),
	}
}

type ControlScore struct {
	// Control action category (Identity, Data, Device, Apps, Infrastructure).
	ControlCategory *string `json:"controlCategory"`
	// Control unique name.
	ControlName *string `json:"controlName"`
	// Description of the control.
	Description *string `json:"description"`
	// Tenant achieved score for the control (it varies day by day depending on tenant operations on the control).
	Score *float64 `json:"score"`
}

func newControlScore(p models.ControlScoreable) *ControlScore {
	if p == nil {
		return nil
	}
	return &ControlScore{
		ControlCategory: p.GetControlCategory(),
		ControlName:     p.GetControlName(),
		Description:     p.GetDescription(),
		Score:           p.GetScore(),
	}
}

type SecurityVendorInformation struct {
	// Specific provider (product/service - not vendor company); for example, WindowsDefenderATP.
	Provider *string `json:"provider"`
	// Version of the provider or subprovider, if it exists, that generated the alert. Required
	ProviderVersion *string `json:"providerVersion"`
	// Specific subprovider (under aggregating provider); for example, WindowsDefenderATP.SmartScreen.
	SubProvider *string `json:"subProvider"`
	// Name of the alert vendor (for example, Microsoft, Dell, FireEye). Required
	Vendor *string `json:"vendor"`
}

func newSecurityVendorInformation(p models.SecurityVendorInformationable) *SecurityVendorInformation {
	if p == nil {
		return nil
	}
	return &SecurityVendorInformation{
		Provider:        p.GetProvider(),
		ProviderVersion: p.GetProviderVersion(),
		SubProvider:     p.GetSubProvider(),
		Vendor:          p.GetVendorEscaped(),
	}
}

type IdentitySecurityDefaultsEnforcementPolicy struct {
	PolicyBase
	// If set to true, Azure Active Directory security defaults is enabled for the tenant.
	IsEnabled *bool `json:"isEnabled"`
}

func newIdentitySecurityDefaultsEnforcementPolicy(p models.IdentitySecurityDefaultsEnforcementPolicyable) *IdentitySecurityDefaultsEnforcementPolicy {
	if p == nil {
		return nil
	}
	return &IdentitySecurityDefaultsEnforcementPolicy{
		PolicyBase: PolicyBase{
			Description: p.GetDescription(),
			DisplayName: p.GetDisplayName(),
		},
		IsEnabled: p.GetIsEnabled(),
	}
}

type ContactMergeSuggestions struct {
	Entity
	IsEnabled *bool `json:"isEnabled"`
}

type LocaleInfo struct {
	// A name representing the user's locale in natural language, for example, 'English (United States)'.
	DisplayName *string `json:"displayName"`
	// A locale representation for the user, which includes the user's preferred language and country/region. For example, 'en-us'. The language component follows 2-letter codes as defined in ISO 639-1, and the country component follows 2-letter codes as defined in ISO 3166-1 alpha-2.
	Locale *string `json:"locale"`
}

func newLocalInfo(p models.LocaleInfoable) *LocaleInfo {
	if p == nil {
		return nil
	}
	return &LocaleInfo{
		DisplayName: p.GetDisplayName(),
		Locale:      p.GetLocale(),
	}
}

func newLocalInfoList(policies []models.LocaleInfoable) []LocaleInfo {
	res := []LocaleInfo{}
	for i := range policies {
		res = append(res, *newLocalInfo(policies[i]))
	}
	return res
}

type Identity struct {
	// The identity's display name. Note that this may not always be available or up to date. For example, if a user changes their display name, the API may show the new value in a future response, but the items associated with the user won't show up as having changed when using delta.
	DisplayName *string `json:"displayName"`
	// Unique identifier for the identity.
	Id *string `json:"id"`
}

func newIdentity(p models.Identityable) *Identity {
	if p == nil {
		return nil
	}
	return &Identity{
		DisplayName: p.GetDisplayName(),
		Id:          p.GetId(),
	}
}

type IdentitySet struct {
	// Optional. The application associated with this action.
	Application *Identity `json:"application"`
	// Optional. The device associated with this action.
	Device *Identity `json:"device"`
	// Optional. The user associated with this action.
	User *Identity `json:"user"`
}

func newIdentitySet(p models.IdentitySetable) *IdentitySet {
	if p == nil {
		return nil
	}
	return &IdentitySet{
		Application: newIdentity(p.GetApplication()),
		Device:      newIdentity(p.GetDevice()),
		User:        newIdentity(p.GetUser()),
	}
}

type ChangeTrackedEntity struct {
	Entity
	// The Timestamp type represents date and time information using ISO 8601 format and is always in UTC time. For example, midnight UTC on Jan 1, 2014 is 2014-01-01T00:00:00Z
	CreatedDateTime *time.Time `json:"createdDateTime"`
	// Identity of the person who last modified the entity.
	LastModifiedBy *IdentitySet `json:"lastModifiedBy"`
	// The Timestamp type represents date and time information using ISO 8601 format and is always in UTC time. For example, midnight UTC on Jan 1, 2014 is 2014-01-01T00:00:00Z
	LastModifiedDateTime *time.Time `json:"lastModifiedDateTime"`
}

type UserSettings struct {
	Entity
	// Reflects the Office Delve organization level setting. When set to true, the organization doesn't have access to Office Delve. This setting is read-only and can only be changed by administrators in the SharePoint admin center.
	ContributionToContentDiscoveryAsOrganizationDisabled *bool `json:"contributionToContentDiscoveryAsOrganizationDisabled"`
	// When set to true, documents in the user's Office Delve are disabled. Users can control this setting in Office Delve.
	ContributionToContentDiscoveryDisabled *bool `json:"contributionToContentDiscoveryDisabled"`
}

func newUserSettings(p models.UserSettingsable) *UserSettings {
	if p == nil {
		return nil
	}
	return &UserSettings{
		Entity: Entity{
			Id: p.GetId(),
		},
		ContributionToContentDiscoveryAsOrganizationDisabled: p.GetContributionToContentDiscoveryAsOrganizationDisabled(),
		ContributionToContentDiscoveryDisabled:               p.GetContributionToContentDiscoveryDisabled(),
	}
}

type (
	DeviceAndAppManagementAssignmentSource     int
	DeviceAndAppManagementAssignmentFilterType int
)

type DeviceCompliancePolicyAssignment struct {
	Entity
}

func newDeviceCompliancePolicyAssignment(p models.DeviceCompliancePolicyAssignmentable) *DeviceCompliancePolicyAssignment {
	if p == nil {
		return nil
	}

	return &DeviceCompliancePolicyAssignment{
		Entity: Entity{
			Id: p.GetId(),
		},
	}
}

func newDeviceCompliancePolicyAssignments(entries []models.DeviceCompliancePolicyAssignmentable) []*DeviceCompliancePolicyAssignment {
	res := []*DeviceCompliancePolicyAssignment{}
	for i := range entries {
		res = append(res, newDeviceCompliancePolicyAssignment(entries[i]))
	}
	return res
}

type PermissionType int

type PermissionGrantConditionSet struct {
	Entity
	// A list of appId values for the client applications to match with, or a list with the single value all to match any client application. Default is the single value all.
	ClientApplicationIds []string `json:"clientApplicationIds"`
	// A list of Microsoft Partner Network (MPN) IDs for verified publishers of the client application, or a list with the single value all to match with client apps from any publisher. Default is the single value all.
	ClientApplicationPublisherIds []string `json:"clientApplicationPublisherIds"`
	// Set to true to only match on client applications with a verified publisher. Set to false to match on any client app, even if it does not have a verified publisher. Default is false.
	ClientApplicationsFromVerifiedPublisherOnly *bool `json:"clientApplicationsFromVerifiedPublisherOnly"`
	// A list of Azure Active Directory tenant IDs in which the client application is registered, or a list with the single value all to match with client apps registered in any tenant. Default is the single value all.
	ClientApplicationTenantIds []string `json:"clientApplicationTenantIds"`
	// The permission classification for the permission being granted, or all to match with any permission classification (including permissions which are not classified). Default is all.
	PermissionClassification *string `json:"permissionClassification"`
	// The list of id values for the specific permissions to match with, or a list with the single value all to match with any permission. The id of delegated permissions can be found in the oauth2PermissionScopes property of the API's **servicePrincipal** object. The id of application permissions can be found in the appRoles property of the API's **servicePrincipal** object. The id of resource-specific application permissions can be found in the resourceSpecificApplicationPermissions property of the API's **servicePrincipal** object. Default is the single value all.
	Permissions []string `json:"permissions"`
	// The permission type of the permission being granted. Possible values: application for application permissions (e.g. app roles), or delegated for delegated permissions. The value delegatedUserConsentable indicates delegated permissions which have not been configured by the API publisher to require admin consent—this value may be used in built-in permission grant policies, but cannot be used in custom permission grant policies. Required.
	PermissionType *PermissionType `json:"permissionType"`
	// The appId of the resource application (e.g. the API) for which a permission is being granted, or any to match with any resource application or API. Default is any.
	ResourceApplication *string `json:"resourceApplication"`
}

func newPermissionGrantConditionSet(p models.PermissionGrantConditionSetable) PermissionGrantConditionSet {
	t := PermissionType(*p.GetPermissionType())

	return PermissionGrantConditionSet{
		Entity: Entity{
			Id: p.GetId(),
		},
		ClientApplicationIds:                        p.GetClientApplicationIds(),
		ClientApplicationPublisherIds:               p.GetClientApplicationPublisherIds(),
		ClientApplicationsFromVerifiedPublisherOnly: p.GetClientApplicationsFromVerifiedPublisherOnly(),
		ClientApplicationTenantIds:                  p.GetClientApplicationTenantIds(),
		PermissionClassification:                    p.GetPermissionClassification(),
		Permissions:                                 p.GetPermissions(),
		PermissionType:                              &t,
		ResourceApplication:                         p.GetResourceApplication(),
	}
}

func newPermissionGrantConditionSets(set []models.PermissionGrantConditionSetable) []PermissionGrantConditionSet {
	res := []PermissionGrantConditionSet{}
	for i := range set {
		res = append(res, newPermissionGrantConditionSet(set[i]))
	}
	return res
}

type PermissionGrantPolicy struct {
	PolicyBase
	// Condition sets which are excluded in this permission grant policy. Automatically expanded on GET.
	Excludes []PermissionGrantConditionSet `json:"excludes"`
	// Condition sets which are included in this permission grant policy. Automatically expanded on GET.
	Includes []PermissionGrantConditionSet `json:"includes"`
}

func newPermissionGrantPolicy(p models.PermissionGrantPolicyable) *PermissionGrantPolicy {
	if p == nil {
		return nil
	}
	return &PermissionGrantPolicy{
		PolicyBase: PolicyBase{
			DirectoryObject: DirectoryObject{
				Entity: Entity{
					Id: p.GetId(),
				},
				DeletedDateTime: p.GetDeletedDateTime(),
			},
			DisplayName: p.GetDisplayName(),
			Description: p.GetDescription(),
		},
		Excludes: newPermissionGrantConditionSets(p.GetExcludes()),
		Includes: newPermissionGrantConditionSets(p.GetIncludes()),
	}
}

func newPermissionGrantPolicies(policies []models.PermissionGrantPolicyable) []*PermissionGrantPolicy {
	res := []*PermissionGrantPolicy{}
	for i := range policies {
		res = append(res, newPermissionGrantPolicy(policies[i]))
	}
	return res
}

type AccessReviewScope struct {
	// Stores additional data not described in the OpenAPI description found when deserializing. Can be used for serialization as well.
	AdditionalData map[string]interface{} `json:"additionalData"`
}

type AccessReviewReviewerScope struct {
	AccessReviewScope
	// The query specifying who will be the reviewer. See table for examples.
	Query *string `json:"query"`
	// In the scenario where reviewers need to be specified dynamically, this property is used to indicate the relative source of the query. This property is only required if a relative query, for example, ./manager, is specified. Possible value: decisions.
	QueryRoot *string `json:"queryRoot"`
	// The type of query. Examples include MicrosoftGraph and ARM.
	QueryType *string `json:"queryType"`
}

func newAccessReviewReviewerScope(p models.AccessReviewReviewerScopeable) AccessReviewReviewerScope {
	return AccessReviewReviewerScope{
		Query:     p.GetQuery(),
		QueryRoot: p.GetQueryRoot(),
		QueryType: p.GetQueryType(),
	}
}

func newAccessReviewReviewerScopes(policies []models.AccessReviewReviewerScopeable) []AccessReviewReviewerScope {
	res := []AccessReviewReviewerScope{}
	for i := range policies {
		res = append(res, newAccessReviewReviewerScope(policies[i]))
	}
	return res
}

type AdminConsentRequestPolicy struct {
	Entity
	// Specifies whether the admin consent request feature is enabled or disabled. Required.
	IsEnabled *bool `json:"isEnabled"`
	// Specifies whether reviewers will receive notifications. Required.
	NotifyReviewers *bool `json:"notifyReviewers"`
	// Specifies whether reviewers will receive reminder emails. Required.
	RemindersEnabled *bool `json:"remindersEnabled"`
	// Specifies the duration the request is active before it automatically expires if no decision is applied.
	RequestDurationInDays *int32 `json:"requestDurationInDays"`
	// The list of reviewers for the admin consent. Required.
	Reviewers []AccessReviewReviewerScope `json:"reviewers"`
	// Specifies the version of this policy. When the policy is updated, this version is updated. Read-only.
	Version *int32 `json:"version"`
}

func newAdminConsentRequestPolicy(p models.AdminConsentRequestPolicyable) *AdminConsentRequestPolicy {
	if p != nil {
		return nil
	}

	return &AdminConsentRequestPolicy{
		Entity: Entity{
			Id: p.GetId(),
		},
		IsEnabled:             p.GetIsEnabled(),
		NotifyReviewers:       p.GetNotifyReviewers(),
		RemindersEnabled:      p.GetRemindersEnabled(),
		RequestDurationInDays: p.GetRequestDurationInDays(),
		Reviewers:             newAccessReviewReviewerScopes(p.GetReviewers()),
		Version:               p.GetVersion(),
	}
}
