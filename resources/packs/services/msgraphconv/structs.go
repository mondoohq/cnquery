package msgraphconv

// this package creates a copy of the msgraph object that we use for embedded struct. This is required since microsoft
// defines structs with lower case and does not attach json tags or implements the standard marshalling function

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/microsoftgraph/msgraph-beta-sdk-go/models"
	"go.mondoo.io/mondoo/resources/packs/core"
)

type AssignedPlan struct {
	AssignedDateTime *time.Time `json:"assignedDateTime"`
	CapabilityStatus string     `json:"capabilityStatus"`
	Service          string     `json:"service"`
	ServicePlanId    string     `json:"servicePlanId"`
}

func NewAssignedPlans(p []models.AssignedPlanable) []AssignedPlan {
	res := []AssignedPlan{}
	for i := range p {
		res = append(res, NewAssignedPlan(p[i]))
	}
	return res
}

func NewAssignedPlan(p models.AssignedPlanable) AssignedPlan {
	return AssignedPlan{
		AssignedDateTime: p.GetAssignedDateTime(),
		CapabilityStatus: core.ToString(p.GetCapabilityStatus()),
		Service:          core.ToString(p.GetService()),
		ServicePlanId:    core.ToString(p.GetServicePlanId()),
	}
}

type VerifiedDomain struct {
	Capabilities string `json:"capabilities"`
	IsDefault    bool   `json:"isDefault"`
	IsInitial    bool   `json:"isInitial"`
	Name         string `json:"name"`
	Type         string `json:"type"`
}

func NewVerifiedDomains(p []models.VerifiedDomainable) []VerifiedDomain {
	res := []VerifiedDomain{}
	for i := range p {
		res = append(res, NewVerifiedDomain(p[i]))
	}
	return res
}

func NewVerifiedDomain(p models.VerifiedDomainable) VerifiedDomain {
	return VerifiedDomain{
		Capabilities: core.ToString(p.GetCapabilities()),
		IsDefault:    core.ToBool(p.GetIsDefault()),
		IsInitial:    core.ToBool(p.GetIsInitial()),
		Name:         core.ToString(p.GetName()),
		Type:         core.ToString(p.GetType()),
	}
}

type UnifiedRolePermission struct {
	AllowedResourceActions  []string `json:"allowedResourceActions"`
	Condition               string   `json:"condition"`
	ExcludedResourceActions []string `json:"excludedResourceActions"`
}

func NewUnifiedRolePermissions(p []models.UnifiedRolePermissionable) []UnifiedRolePermission {
	res := []UnifiedRolePermission{}
	for i := range p {
		res = append(res, NewUnifiedRolePermission(p[i]))
	}
	return res
}

func NewUnifiedRolePermission(p models.UnifiedRolePermissionable) UnifiedRolePermission {
	return UnifiedRolePermission{
		AllowedResourceActions:  p.GetAllowedResourceActions(),
		Condition:               core.ToString(p.GetCondition()),
		ExcludedResourceActions: p.GetExcludedResourceActions(),
	}
}

type DirectorySetting struct {
	DisplayName string         `json:"displayName"`
	TemplateId  string         `json:"templateId"`
	Values      []SettingValue `json:"values"`
}

type SettingValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func NewDirectorySettings(p []models.DirectorySettingable) []DirectorySetting {
	res := []DirectorySetting{}
	for i := range p {
		res = append(res, NewDirectorySetting(p[i]))
	}
	return res
}

func NewDirectorySetting(p models.DirectorySettingable) DirectorySetting {
	values := []SettingValue{}
	entries := p.GetValues()
	for i := range entries {
		values = append(values, SettingValue{
			Name:  core.ToString(entries[i].GetName()),
			Value: core.ToString(entries[i].GetValue()),
		})
	}

	return DirectorySetting{
		DisplayName: core.ToString(p.GetDisplayName()),
		TemplateId:  core.ToString(p.GetTemplateId()),
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

func NewDirectoryPricipal(p models.DirectoryObjectable) *DirectoryObject {
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

func NewAllowInvitesFrom(a *models.AllowInvitesFrom) *AllowInvitesFrom {
	if a == nil {
		return nil
	}
	n := AllowInvitesFrom(int(*a))
	return &n
}

type DefaultUserRoleOverride struct {
	Entity
	IsDefault       *bool                   `json:"isDefault"`
	RolePermissions []UnifiedRolePermission `json:"rolePermissions"`
}

func NewDefaultUserRoleOverride(a models.DefaultUserRoleOverrideable) DefaultUserRoleOverride {
	return DefaultUserRoleOverride{
		Entity: Entity{
			Id: a.GetId(),
		},
		IsDefault:       a.GetIsDefault(),
		RolePermissions: NewUnifiedRolePermissions(a.GetRolePermissions()),
	}
}

func NewDefaultUserRoleOverrides(a []models.DefaultUserRoleOverrideable) []DefaultUserRoleOverride {
	res := []DefaultUserRoleOverride{}
	for i := range a {
		res = append(res, NewDefaultUserRoleOverride(a[i]))
	}
	return res
}

type DefaultUserRolePermissions struct {
	// Indicates whether the default user role can create applications.
	AllowedToCreateApps *bool `json:"allowedToCreateApps"`
	// Indicates whether the default user role can create security groups.
	AllowedToCreateSecurityGroups *bool `json:"allowedToCreateSecurityGroups"`
	// Indicates whether the default user role can read other users.
	AllowedToReadOtherUsers *bool `json:"allowedToReadOtherUsers"`
}

func NewDefaultUserRolePermissions(a models.DefaultUserRolePermissionsable) *DefaultUserRolePermissions {
	if a == nil {
		return nil
	}
	return &DefaultUserRolePermissions{
		AllowedToCreateApps:           a.GetAllowedToCreateApps(),
		AllowedToCreateSecurityGroups: a.GetAllowedToCreateSecurityGroups(),
		AllowedToReadOtherUsers:       a.GetAllowedToReadOtherUsers(),
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
	DefaultUserRoleOverrides []DefaultUserRoleOverride `json:"defaultUserRoleOverrides"`
	//
	DefaultUserRolePermissions *DefaultUserRolePermissions `json:"defaultUserRolePermissions"`
	// List of features enabled for private preview on the tenant.
	EnabledPreviewFeatures []string `json:"enabledPreviewFeatures"`
	// Represents role templateId for the role that should be granted to guest user. Currently following roles are supported:  User (a0b1b346-4d3e-4e8b-98f8-753987be4970), Guest User (10dae51f-b6af-4016-8d66-8c2a99b929b3), and Restricted Guest User (2af84b1e-32c8-42b7-82bc-daa82404023b).
	GuestUserRoleId *string `json:"guestUserRoleId"`
	// Indicates if user consent to apps is allowed, and if it is, which app consent policy (permissionGrantPolicy) governs the permission for users to grant consent. Values should be in the format managePermissionGrantsForSelf.{id}, where {id} is the id of a built-in or custom app consent policy. An empty list indicates user consent to apps is disabled.
	PermissionGrantPolicyIdsAssignedToDefaultUserRole []string `json:"permissionGrantPolicyIdsAssignedToDefaultUserRole"`
}

func NewAuthorizationPolicys(policies []models.AuthorizationPolicyable) []*AuthorizationPolicy {
	res := []*AuthorizationPolicy{}
	for i := range policies {
		res = append(res, NewAuthorizationPolicy(policies[i]))
	}
	return res
}

func NewAuthorizationPolicy(p models.AuthorizationPolicyable) *AuthorizationPolicy {
	if p == nil {
		return nil
	}
	return &AuthorizationPolicy{
		AllowedToSignUpEmailBasedSubscriptions:            p.GetAllowedToSignUpEmailBasedSubscriptions(),
		AllowedToUseSSPR:                                  p.GetAllowedToUseSSPR(),
		AllowEmailVerifiedUsersToJoinOrganization:         p.GetAllowEmailVerifiedUsersToJoinOrganization(),
		AllowInvitesFrom:                                  NewAllowInvitesFrom(p.GetAllowInvitesFrom()),
		BlockMsolPowerShell:                               p.GetBlockMsolPowerShell(),
		DefaultUserRoleOverrides:                          NewDefaultUserRoleOverrides(p.GetDefaultUserRoleOverrides()),
		DefaultUserRolePermissions:                        NewDefaultUserRolePermissions(p.GetDefaultUserRolePermissions()),
		EnabledPreviewFeatures:                            p.GetEnabledPreviewFeatures(),
		GuestUserRoleId:                                   p.GetGuestUserRoleId(),
		PermissionGrantPolicyIdsAssignedToDefaultUserRole: p.GetPermissionGrantPolicyIdsAssignedToDefaultUserRole(),
	}
}

type AverageComparativeScore struct {
	// Average score within specified basis.
	AverageScore *float64 `json:"averageScore"`
	// Scope type. The possible values are: AllTenants, TotalSeats, IndustryTypes.
	Basis *string `json:"basis"`
}

func NewAverageComparativeScore(p models.AverageComparativeScoreable) *AverageComparativeScore {
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

func NewControlScore(p models.ControlScoreable) *ControlScore {
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

func NewSecurityVendorInformation(p models.SecurityVendorInformationable) *SecurityVendorInformation {
	if p == nil {
		return nil
	}
	return &SecurityVendorInformation{
		Provider:        p.GetProvider(),
		ProviderVersion: p.GetProviderVersion(),
		SubProvider:     p.GetSubProvider(),
		Vendor:          p.GetVendor(),
	}
}

type IdentitySecurityDefaultsEnforcementPolicy struct {
	PolicyBase
	// If set to true, Azure Active Directory security defaults is enabled for the tenant.
	IsEnabled *bool `json:"isEnabled"`
}

func NewIdentitySecurityDefaultsEnforcementPolicy(p models.IdentitySecurityDefaultsEnforcementPolicyable) *IdentitySecurityDefaultsEnforcementPolicy {
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

func NewContactMergeSuggestions(p models.ContactMergeSuggestionsable) *ContactMergeSuggestions {
	if p == nil {
		return nil
	}

	return &ContactMergeSuggestions{
		Entity: Entity{
			Id: p.GetId(),
		},
		IsEnabled: p.GetIsEnabled(),
	}
}

type UserInsightsSettings struct {
	Entity
	// true if user's itemInsights and meeting hours insights are enabled; false if user's itemInsights and meeting hours insights are disabled. Default is true. Optional.
	IsEnabled *bool `json:"isEnabled"`
}

func NewUserInsightsSettings(p models.UserInsightsSettingsable) *UserInsightsSettings {
	if p == nil {
		return nil
	}

	return &UserInsightsSettings{
		Entity: Entity{
			Id: p.GetId(),
		},
		IsEnabled: p.GetIsEnabled(),
	}
}

type LocaleInfo struct {
	// A name representing the user's locale in natural language, for example, 'English (United States)'.
	DisplayName *string `json:"displayName"`
	// A locale representation for the user, which includes the user's preferred language and country/region. For example, 'en-us'. The language component follows 2-letter codes as defined in ISO 639-1, and the country component follows 2-letter codes as defined in ISO 3166-1 alpha-2.
	Locale *string `json:"locale"`
}

func NewLocalInfo(p models.LocaleInfoable) *LocaleInfo {
	if p == nil {
		return nil
	}
	return &LocaleInfo{
		DisplayName: p.GetDisplayName(),
		Locale:      p.GetLocale(),
	}
}

func NewLocalInfoList(policies []models.LocaleInfoable) []LocaleInfo {
	res := []LocaleInfo{}
	for i := range policies {
		res = append(res, *NewLocalInfo(policies[i]))
	}
	return res
}

type RegionalFormatOverrides struct {
	// The calendar to use, e.g., Gregorian Calendar.Returned by default.
	Calendar *string `json:"calendar"`
	// The first day of the week to use, e.g., Sunday.Returned by default.
	FirstDayOfWeek *string `json:"firstDayOfWeek"`
	// The long date time format to be used for displaying dates.Returned by default.
	LongDateFormat *string `json:"longDateFormat"`
	// The long time format to be used for displaying time.Returned by default.
	LongTimeFormat *string `json:"longTimeFormat"`
	// The short date time format to be used for displaying dates.Returned by default.
	ShortDateFormat *string `json:"shortDateFormat"`
	// The short time format to be used for displaying time.Returned by default.
	ShortTimeFormat *string `json:"shortTimeFormat"`
	// The timezone to be used for displaying time.Returned by default.
	TimeZone *string `json:"timeZone"`
}

func NewRegionalFormatOverrides(p models.RegionalFormatOverridesable) *RegionalFormatOverrides {
	if p == nil {
		return nil
	}
	return &RegionalFormatOverrides{
		Calendar:        p.GetCalendar(),
		FirstDayOfWeek:  p.GetFirstDayOfWeek(),
		LongDateFormat:  p.GetLongDateFormat(),
		LongTimeFormat:  p.GetLongTimeFormat(),
		ShortDateFormat: p.GetShortDateFormat(),
		ShortTimeFormat: p.GetShortTimeFormat(),
		TimeZone:        p.GetTimeZone(),
	}
}

type TranslationBehavior int

type TranslationLanguageOverride struct {
	// The language to apply the override.Returned by default. Not nullable.
	LanguageTag *string `json:"languageTag"`
	// The translation override behavior for the language, if any.Returned by default. Not nullable.
	TranslationBehavior *TranslationBehavior `json:"translationBehavior"`
}

func NewTranslationLanguageOverride(p models.TranslationLanguageOverrideable) TranslationLanguageOverride {
	var tb *TranslationBehavior
	if p.GetTranslationBehavior() != nil {
		v := TranslationBehavior(*p.GetTranslationBehavior())
		tb = &v
	}
	return TranslationLanguageOverride{
		LanguageTag:         p.GetLanguageTag(),
		TranslationBehavior: tb,
	}
}

func NewTranslationLanguageOverrideList(entries []models.TranslationLanguageOverrideable) []TranslationLanguageOverride {
	res := []TranslationLanguageOverride{}
	for i := range entries {
		res = append(res, NewTranslationLanguageOverride(entries[i]))
	}
	return res
}

type TranslationPreferences struct {
	// Translation override behavior for languages, if any.Returned by default.
	LanguageOverrides []TranslationLanguageOverride `json:"languageOverrides"`
	// The user's preferred translation behavior.Returned by default. Not nullable.
	TranslationBehavior *TranslationBehavior `json:"translationBehavior"`
	// The list of languages the user does not need translated. This is computed from the authoringLanguages collection in regionalAndLanguageSettings, and the languageOverrides collection in translationPreferences. The list specifies neutral culture values that include the language code without any country or region association. For example, it would specify 'fr' for the neutral French culture, but not 'fr-FR' for the French culture in France. Returned by default. Read only.
	UntranslatedLanguages []string `json:"untranslatedLanguages"`
}

func NewTranslationPreferences(p models.TranslationPreferencesable) *TranslationPreferences {
	if p == nil {
		return nil
	}

	var tb *TranslationBehavior
	if p.GetTranslationBehavior() != nil {
		v := TranslationBehavior(*p.GetTranslationBehavior())
		tb = &v
	}

	return &TranslationPreferences{
		LanguageOverrides:     NewTranslationLanguageOverrideList(p.GetLanguageOverrides()),
		TranslationBehavior:   tb,
		UntranslatedLanguages: p.GetUntranslatedLanguages(),
	}
}

type RegionalAndLanguageSettings struct {
	Entity
	// Prioritized list of languages the user reads and authors in.Returned by default. Not nullable.
	AuthoringLanguages []LocaleInfo `json:"authoringLanguages"`
	// The  user's preferred user interface language (menus, buttons, ribbons, warning messages) for Microsoft web applications.Returned by default. Not nullable.
	DefaultDisplayLanguage *LocaleInfo `json:"defaultDisplayLanguage"`
	// The locale that drives the default date, time, and calendar formatting.Returned by default.
	DefaultRegionalFormat *LocaleInfo `json:"defaultRegionalFormat"`
	// The language a user expected to use as input for text to speech scenarios.Returned by default.
	DefaultSpeechInputLanguage *LocaleInfo `json:"defaultSpeechInputLanguage"`
	// The language a user expects to have documents, emails, and messages translated into.Returned by default.
	DefaultTranslationLanguage *LocaleInfo `json:"defaultTranslationLanguage"`
	// Allows a user to override their defaultRegionalFormat with field specific formats.Returned by default.
	RegionalFormatOverrides *RegionalFormatOverrides `json:"regionalFormatOverrides"`
	// The user's preferred settings when consuming translated documents, emails, messages, and websites.Returned by default. Not nullable.
	TranslationPreferences *TranslationPreferences `json:"translationPreferences"`
}

func NewRegionalAndLanguageSettings(p models.RegionalAndLanguageSettingsable) *RegionalAndLanguageSettings {
	if p == nil {
		return nil
	}

	return &RegionalAndLanguageSettings{
		Entity: Entity{
			Id: p.GetId(),
		},
		AuthoringLanguages:         NewLocalInfoList(p.GetAuthoringLanguages()),
		DefaultDisplayLanguage:     NewLocalInfo(p.GetDefaultDisplayLanguage()),
		DefaultRegionalFormat:      NewLocalInfo(p.GetDefaultRegionalFormat()),
		DefaultSpeechInputLanguage: NewLocalInfo(p.GetDefaultSpeechInputLanguage()),
		DefaultTranslationLanguage: NewLocalInfo(p.GetDefaultTranslationLanguage()),
		RegionalFormatOverrides:    NewRegionalFormatOverrides(p.GetRegionalFormatOverrides()),
		TranslationPreferences:     NewTranslationPreferences(p.GetTranslationPreferences()),
	}
}

type Identity struct {
	// The identity's display name. Note that this may not always be available or up to date. For example, if a user changes their display name, the API may show the new value in a future response, but the items associated with the user won't show up as having changed when using delta.
	DisplayName *string `json:"displayName"`
	// Unique identifier for the identity.
	Id *string `json:"id"`
}

func NewIdentity(p models.Identityable) *Identity {
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

func NewIdentitySet(p models.IdentitySetable) *IdentitySet {
	if p == nil {
		return nil
	}
	return &IdentitySet{
		Application: NewIdentity(p.GetApplication()),
		Device:      NewIdentity(p.GetDevice()),
		User:        NewIdentity(p.GetUser()),
	}
}

type ChangeTrackedEntity struct {
	Entity
	//
	CreatedBy *IdentitySet `json:"createdBy"`
	// The Timestamp type represents date and time information using ISO 8601 format and is always in UTC time. For example, midnight UTC on Jan 1, 2014 is 2014-01-01T00:00:00Z
	CreatedDateTime *time.Time `json:"createdDateTime"`
	// Identity of the person who last modified the entity.
	LastModifiedBy *IdentitySet `json:"lastModifiedBy"`
	// The Timestamp type represents date and time information using ISO 8601 format and is always in UTC time. For example, midnight UTC on Jan 1, 2014 is 2014-01-01T00:00:00Z
	LastModifiedDateTime *time.Time `json:"lastModifiedDateTime"`
}

type TimeRange struct {
	// End time for the time range.
	EndTime *time.Time `json:"endTime"`
	// Start time for the time range.
	StartTime *time.Time `json:"startTime"`
}

const timeOnlyFormat = "15:04:05.000000000"

var timeOnlyParsingFormats = map[int]string{
	0: "15:04:05", // Go doesn't seem to support optional parameters in time.Parse, which is sad
	1: "15:04:05.0",
	2: "15:04:05.00",
	3: "15:04:05.000",
	4: "15:04:05.0000",
	5: "15:04:05.00000",
	6: "15:04:05.000000",
	7: "15:04:05.0000000",
	8: "15:04:05.00000000",
	9: timeOnlyFormat,
}

// ParseTimeOnly parses a string into a TimeOnly following the RFC3339 standard.
func ParseTimeOnly(s string) *time.Time {
	if len(strings.TrimSpace(s)) <= 0 {
		return nil
	}
	splat := strings.Split(s, ".")
	parsingFormat := timeOnlyParsingFormats[0]
	if len(splat) > 1 {
		dotSectionLen := len(splat[1])
		if dotSectionLen >= len(timeOnlyParsingFormats) {
			return nil
		}
		parsingFormat = timeOnlyParsingFormats[dotSectionLen]
	}
	timeValue, err := time.Parse(parsingFormat, s)
	if err != nil {
		return nil
	}
	return &timeValue
}

func NewTimeRange(p models.TimeRangeable) TimeRange {
	return TimeRange{
		EndTime:   ParseTimeOnly(p.GetEndTime().String()),
		StartTime: ParseTimeOnly(p.GetStartTime().String()),
	}
}

func NewTimeRangeList(entries []models.TimeRangeable) []TimeRange {
	res := []TimeRange{}
	for i := range entries {
		res = append(res, NewTimeRange(entries[i]))
	}
	return res
}

// TODO: update marshaling
type (
	DayOfWeek int
	WeekIndex int
)

func NewDayOfWeek(p models.DayOfWeek) DayOfWeek {
	return DayOfWeek(p)
}

func NewDayOfWeekList(entries []models.DayOfWeek) []DayOfWeek {
	res := []DayOfWeek{}
	for i := range entries {
		res = append(res, NewDayOfWeek(entries[i]))
	}
	return res
}

type RecurrencePatternType int

type RecurrencePattern struct {
	// The day of the month on which the event occurs. Required if type is absoluteMonthly or absoluteYearly.
	DayOfMonth *int32 `json:"dayOfMonth"`
	// A collection of the days of the week on which the event occurs. The possible values are: sunday, monday, tuesday, wednesday, thursday, friday, saturday. If type is relativeMonthly or relativeYearly, and daysOfWeek specifies more than one day, the event falls on the first day that satisfies the pattern.  Required if type is weekly, relativeMonthly, or relativeYearly.
	DaysOfWeek []DayOfWeek `json:"daysOfWeek"`
	// The first day of the week. The possible values are: sunday, monday, tuesday, wednesday, thursday, friday, saturday. Default is sunday. Required if type is weekly.
	FirstDayOfWeek *DayOfWeek `json:"firstDayOfWeek"`
	// Specifies on which instance of the allowed days specified in daysOfWeek the event occurs, counted from the first instance in the month. The possible values are: first, second, third, fourth, last. Default is first. Optional and used if type is relativeMonthly or relativeYearly.
	Index *WeekIndex `json:"index"`
	// The number of units between occurrences, where units can be in days, weeks, months, or years, depending on the type. Required.
	Interval *int32 `json:"interval"`
	// The month in which the event occurs.  This is a number from 1 to 12.
	Month *int32 `json:"month"`
	// The recurrence pattern type: daily, weekly, absoluteMonthly, relativeMonthly, absoluteYearly, relativeYearly. Required. For more information, see values of type property.
	Type *RecurrencePatternType `json:"type"`
}

func NewRecurrencePattern(p models.RecurrencePatternable) *RecurrencePattern {
	if p == nil {
		return nil
	}

	var idx *WeekIndex
	if p.GetIndex() != nil {
		v := WeekIndex(int(*p.GetIndex()))
		idx = &v
	}

	var t *RecurrencePatternType
	if p.GetType() != nil {
		v := RecurrencePatternType(int(*p.GetType()))
		t = &v
	}

	var firstDayOfWeek *DayOfWeek
	if p.GetFirstDayOfWeek() != nil {
		v := NewDayOfWeek(*p.GetFirstDayOfWeek())
		firstDayOfWeek = &v
	}

	return &RecurrencePattern{
		DayOfMonth:     p.GetDayOfMonth(),
		DaysOfWeek:     NewDayOfWeekList(p.GetDaysOfWeek()),
		FirstDayOfWeek: firstDayOfWeek,
		Index:          idx,
		Interval:       p.GetInterval(),
		Month:          p.GetMonth(),
		Type:           t,
	}
}

type RecurrenceRangeType int

type RecurrenceRange struct {
	// The date to stop applying the recurrence pattern. Depending on the recurrence pattern of the event, the last occurrence of the meeting may not be this date. Required if type is endDate.
	EndDate *time.Time `json:"endDate"`
	// The number of times to repeat the event. Required and must be positive if type is numbered.
	NumberOfOccurrences *int32 `json:"numberOfOccurrences"`
	// Time zone for the startDate and endDate properties. Optional. If not specified, the time zone of the event is used.
	RecurrenceTimeZone *string `json:"recurrenceTimeZone"`
	// The date to start applying the recurrence pattern. The first occurrence of the meeting may be this date or later, depending on the recurrence pattern of the event. Must be the same value as the start property of the recurring event. Required.
	StartDate *time.Time `json:"startDate"`
	// The recurrence range. The possible values are: endDate, noEnd, numbered. Required.
	Type *RecurrenceRangeType `json:"type"`
}

const dateOnlyFormat = "2006-01-02"

// ParseDateOnly parses a string into a DateOnly following the RFC3339 standard.
func ParseDateOnly(s string) *time.Time {
	if len(strings.TrimSpace(s)) <= 0 {
		return nil
	}
	timeValue, err := time.Parse(dateOnlyFormat, s)
	if err != nil {
		return nil
	}
	return &timeValue
}

func NewRecurrenceRange(p models.RecurrenceRangeable) *RecurrenceRange {
	if p == nil {
		return nil
	}

	var t *RecurrenceRangeType
	if p.GetType() != nil {
		v := RecurrenceRangeType(*p.GetType())
		t = &v
	}

	var endDate *time.Time
	if p.GetEndDate() != nil {
		endDate = ParseDateOnly(p.GetEndDate().String())
	}

	var startDate *time.Time
	if p.GetStartDate() != nil {
		startDate = ParseDateOnly(p.GetStartDate().String())
	}

	return &RecurrenceRange{
		EndDate:             endDate,
		NumberOfOccurrences: p.GetNumberOfOccurrences(),
		RecurrenceTimeZone:  p.GetRecurrenceTimeZone(),
		StartDate:           startDate,
		Type:                t,
	}
}

type PatternedRecurrence struct {
	// The frequency of an event.  For access reviews: Do not specify this property for a one-time access review.  Only interval, dayOfMonth, and type (weekly, absoluteMonthly) properties of recurrencePattern are supported.
	Pattern *RecurrencePattern `json:"pattern"`
	// The duration of an event.
	Range *RecurrenceRange `json:"range"`
}

func NewPatternedRecurrence(p models.PatternedRecurrenceable) *PatternedRecurrence {
	if p == nil {
		return nil
	}

	return &PatternedRecurrence{
		Pattern: NewRecurrencePattern(p.GetPattern()),
		Range:   NewRecurrenceRange(p.GetRange()),
	}
}

type ShiftAvailability struct {
	// Specifies the pattern for recurrence
	Recurrence *PatternedRecurrence `json:"recurrence"`
	// The time slot(s) preferred by the user.
	TimeSlots []TimeRange `json:"timeSlots"`
	// Specifies the time zone for the indicated time.
	TimeZone *string `json:"timeZone"`
}

func NewShiftAvailability(p models.ShiftAvailabilityable) ShiftAvailability {
	return ShiftAvailability{
		Recurrence: NewPatternedRecurrence(p.GetRecurrence()),
		TimeSlots:  NewTimeRangeList(p.GetTimeSlots()),
		TimeZone:   p.GetTimeZone(),
	}
}

func NewShiftAvailabilityList(entries []models.ShiftAvailabilityable) []ShiftAvailability {
	res := []ShiftAvailability{}
	for i := range entries {
		res = append(res, NewShiftAvailability(entries[i]))
	}
	return res
}

type ShiftPreferences struct {
	ChangeTrackedEntity
	// Availability of the user to be scheduled for work and its recurrence pattern.
	Availability []ShiftAvailability `json:"availability"`
}

func NewShiftPreferences(p models.ShiftPreferencesable) *ShiftPreferences {
	return &ShiftPreferences{
		ChangeTrackedEntity: ChangeTrackedEntity{
			Entity: Entity{
				Id: p.GetId(),
			},
			CreatedBy:            NewIdentitySet(p.GetCreatedBy()),
			LastModifiedBy:       NewIdentitySet(p.GetLastModifiedBy()),
			CreatedDateTime:      p.GetCreatedDateTime(),
			LastModifiedDateTime: p.GetLastModifiedDateTime(),
		},
		Availability: NewShiftAvailabilityList(p.GetAvailability()),
	}
}

type UserSettings struct {
	Entity
	//
	ContactMergeSuggestions *ContactMergeSuggestions `json:"contactMergeSuggestions"`
	// Reflects the Office Delve organization level setting. When set to true, the organization doesn't have access to Office Delve. This setting is read-only and can only be changed by administrators in the SharePoint admin center.
	ContributionToContentDiscoveryAsOrganizationDisabled *bool `json:"contributionToContentDiscoveryAsOrganizationDisabled"`
	// When set to true, documents in the user's Office Delve are disabled. Users can control this setting in Office Delve.
	ContributionToContentDiscoveryDisabled *bool `json:"contributionToContentDiscoveryDisabled"`
	// The user's settings for the visibility of meeting hour insights, and insights derived between a user and other items in Microsoft 365, such as documents or sites. Get userInsightsSettings through this navigation property.
	ItemInsights *UserInsightsSettings `json:"itemInsights"`
	// The user's preferences for languages, regional locale and date/time formatting.
	RegionalAndLanguageSettings *RegionalAndLanguageSettings `json:"regionalAndLanguageSettings"`
	// The shift preferences for the user.
	ShiftPreferences *ShiftPreferences `json:"shiftPreferences"`
}

func NewUserSettings(p models.UserSettingsable) *UserSettings {
	if p == nil {
		return nil
	}
	return &UserSettings{
		Entity: Entity{
			Id: p.GetId(),
		},
		ContactMergeSuggestions:                              NewContactMergeSuggestions(p.GetContactMergeSuggestions()),
		ContributionToContentDiscoveryAsOrganizationDisabled: p.GetContributionToContentDiscoveryAsOrganizationDisabled(),
		ContributionToContentDiscoveryDisabled:               p.GetContributionToContentDiscoveryDisabled(),
		ItemInsights:                                         NewUserInsightsSettings(p.GetItemInsights()),
		RegionalAndLanguageSettings:                          NewRegionalAndLanguageSettings(p.GetRegionalAndLanguageSettings()),
		ShiftPreferences:                                     NewShiftPreferences(p.GetShiftPreferences()),
	}
}

type (
	DeviceAndAppManagementAssignmentSource     int
	DeviceAndAppManagementAssignmentFilterType int
)

type DeviceAndAppManagementAssignmentTarget struct {
	// The Id of the filter for the target assignment.
	DeviceAndAppManagementAssignmentFilterId *string `json:"deviceAndAppManagementAssignmentFilterId"`
	// The type of filter of the target assignment i.e. Exclude or Include. Possible values are: none, include, exclude.
	DeviceAndAppManagementAssignmentFilterType *DeviceAndAppManagementAssignmentFilterType `json:"deviceAndAppManagementAssignmentFilterType"`
}

func NewDeviceAndAppManagementAssignmentTarget(p models.DeviceAndAppManagementAssignmentTargetable) *DeviceAndAppManagementAssignmentTarget {
	var filterType *DeviceAndAppManagementAssignmentFilterType
	if p.GetDeviceAndAppManagementAssignmentFilterType() != nil {
		t := DeviceAndAppManagementAssignmentFilterType(*p.GetDeviceAndAppManagementAssignmentFilterType())
		filterType = &t
	}
	return &DeviceAndAppManagementAssignmentTarget{
		DeviceAndAppManagementAssignmentFilterId:   p.GetDeviceAndAppManagementAssignmentFilterId(),
		DeviceAndAppManagementAssignmentFilterType: filterType,
	}
}

type DeviceCompliancePolicyAssignment struct {
	Entity
	// The assignment source for the device compliance policy, direct or parcel/policySet. Possible values are: direct, policySets.
	Source *DeviceAndAppManagementAssignmentSource `json:"source"`
	// The identifier of the source of the assignment.
	SourceId *string `json:"vendor"`
	// Target for the compliance policy assignment.
	Target *DeviceAndAppManagementAssignmentTarget `json:"target"`
}

func NewDeviceCompliancePolicyAssignment(p models.DeviceCompliancePolicyAssignmentable) *DeviceCompliancePolicyAssignment {
	if p == nil {
		return nil
	}
	var source *DeviceAndAppManagementAssignmentSource
	if p.GetSource() != nil {
		s := DeviceAndAppManagementAssignmentSource(*p.GetSource())
		source = &s
	}

	return &DeviceCompliancePolicyAssignment{
		Entity: Entity{
			Id: p.GetId(),
		},
		Source:   source,
		SourceId: p.GetSourceId(),
		Target:   NewDeviceAndAppManagementAssignmentTarget(p.GetTarget()),
	}
}

func NewDeviceCompliancePolicyAssignments(entries []models.DeviceCompliancePolicyAssignmentable) []*DeviceCompliancePolicyAssignment {
	res := []*DeviceCompliancePolicyAssignment{}
	for i := range entries {
		res = append(res, NewDeviceCompliancePolicyAssignment(entries[i]))
	}
	return res
}

type PermissionType int

type PermissionGrantConditionSet struct {
	Entity
	// Set to true to only match on client applications that are Microsoft 365 certified. Set to false to match on any other client app. Default is false.
	CertifiedClientApplicationsOnly *bool `json:"certifiedClientApplicationsOnly"`
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

func NewPermissionGrantConditionSet(p models.PermissionGrantConditionSetable) PermissionGrantConditionSet {
	t := PermissionType(*p.GetPermissionType())

	return PermissionGrantConditionSet{
		Entity: Entity{
			Id: p.GetId(),
		},
		CertifiedClientApplicationsOnly:             p.GetCertifiedClientApplicationsOnly(),
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

func NewPermissionGrantConditionSets(set []models.PermissionGrantConditionSetable) []PermissionGrantConditionSet {
	res := []PermissionGrantConditionSet{}
	for i := range set {
		res = append(res, NewPermissionGrantConditionSet(set[i]))
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

func NewPermissionGrantPolicy(p models.PermissionGrantPolicyable) *PermissionGrantPolicy {
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
		Excludes: NewPermissionGrantConditionSets(p.GetExcludes()),
		Includes: NewPermissionGrantConditionSets(p.GetIncludes()),
	}
}

func NewPermissionGrantPolicys(policies []models.PermissionGrantPolicyable) []*PermissionGrantPolicy {
	res := []*PermissionGrantPolicy{}
	for i := range policies {
		res = append(res, NewPermissionGrantPolicy(policies[i]))
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

func NewAccessReviewReviewerScope(p models.AccessReviewReviewerScopeable) AccessReviewReviewerScope {
	return AccessReviewReviewerScope{
		Query:     p.GetQuery(),
		QueryRoot: p.GetQueryRoot(),
		QueryType: p.GetQueryType(),
	}
}

func NewAccessReviewReviewerScopes(policies []models.AccessReviewReviewerScopeable) []AccessReviewReviewerScope {
	res := []AccessReviewReviewerScope{}
	for i := range policies {
		res = append(res, NewAccessReviewReviewerScope(policies[i]))
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

func NewAdminConsentRequestPolicy(p models.AdminConsentRequestPolicyable) *AdminConsentRequestPolicy {
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
		Reviewers:             NewAccessReviewReviewerScopes(p.GetReviewers()),
		Version:               p.GetVersion(),
	}
}
