// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/microsoftgraph/msgraph-sdk-go/directory"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/organization"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/logger"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers/ms365/connection"
	"go.mondoo.com/cnquery/v12/types"
)

func (m *mqlMicrosoftTenant) id() (string, error) {
	return m.Id.Data, nil
}

// Deprecated: use `microsoft.tenant` instead
func (a *mqlMicrosoft) organizations() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.Organization().Get(ctx, &organization.OrganizationRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}

	res := []any{}
	orgs := resp.GetValue()
	for i := range orgs {
		org := orgs[i]
		mqlResource, err := newMicrosoftTenant(a.MqlRuntime, org)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

var tenantFields = []string{
	"id",
	"assignedPlans",
	"createdDateTime",
	"displayName",
	"verifiedDomains",
	"onPremisesSyncEnabled",
	"tenantType",
	"provisionedPlans",
	"privacyProfile",
	"technicalNotificationMails",
	"preferredLanguage",
}

var tenantBrandingReport = `
$UserName = '%s'

# Query realm discovery endpoint
$response = Invoke-RestMethod -UseBasicParsing -Uri ("https://login.microsoftonline.com/common/userrealm/" + $UserName + "?api-version=2.0")

# Build object with only the fields we care about
$tenantBranding = New-Object PSObject
Add-Member -InputObject $tenantBranding -MemberType NoteProperty -Name Login -Value $response.Login
Add-Member -InputObject $tenantBranding -MemberType NoteProperty -Name DomainName -Value $response.DomainName
Add-Member -InputObject $tenantBranding -MemberType NoteProperty -Name FederationBrandName -Value $response.FederationBrandName

if ($response.TenantBrandingInfo -ne $null -and $response.TenantBrandingInfo.Count -gt 0) {
    $brandingInfo = $response.TenantBrandingInfo[0]
    Add-Member -InputObject $tenantBranding -MemberType NoteProperty -Name KeepMeSignedInDisabled -Value $brandingInfo.KeepMeSignedInDisabled
    Add-Member -InputObject $tenantBranding -MemberType NoteProperty -Name BannerLogo -Value $brandingInfo.BannerLogo
    Add-Member -InputObject $tenantBranding -MemberType NoteProperty -Name BackgroundColor -Value $brandingInfo.BackgroundColor
} else {
    Add-Member -InputObject $tenantBranding -MemberType NoteProperty -Name KeepMeSignedInDisabled -Value $null
}

ConvertTo-Json -Depth 4 $tenantBranding
`

func initMicrosoftTenant(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.Organization().ByOrganizationId(conn.TenantId()).Get(ctx, &organization.OrganizationItemRequestBuilderGetRequestConfiguration{
		QueryParameters: &organization.OrganizationItemRequestBuilderGetQueryParameters{
			Select: tenantFields,
		},
	})
	if err != nil {
		return nil, nil, transformError(err)
	}

	tenant, err := newMicrosoftTenant(runtime, resp)
	if err != nil {
		return nil, nil, err
	}
	return nil, tenant, nil
}

func newMicrosoftTenant(runtime *plugin.Runtime, org models.Organizationable) (*mqlMicrosoftTenant, error) {
	assignedPlans, err := convert.JsonToDictSlice(newAssignedPlans(org.GetAssignedPlans()))
	if err != nil {
		return nil, err
	}
	verifiedDomains, err := convert.JsonToDictSlice(newVerifiedDomains(org.GetVerifiedDomains()))
	if err != nil {
		return nil, err
	}

	provisionedPlans, err := convert.JsonToDictSlice(newProvisionedPlans(org.GetProvisionedPlans()))
	if err != nil {
		return nil, err
	}

	privacyProfileDict := map[string]any{}
	if org.GetPrivacyProfile() != nil {
		privacyProfileDict, err = convert.JsonToDict(newPrivacyProfile(org.GetPrivacyProfile()))
		if err != nil {
			return nil, err
		}
	}

	mqlResource, err := CreateResource(runtime, "microsoft.tenant",
		map[string]*llx.RawData{
			"id":                         llx.StringDataPtr(org.GetId()),
			"assignedPlans":              llx.DictData(assignedPlans),
			"createdDateTime":            llx.TimeDataPtr(org.GetCreatedDateTime()), // deprecated
			"name":                       llx.StringDataPtr(org.GetDisplayName()),
			"verifiedDomains":            llx.DictData(verifiedDomains),
			"onPremisesSyncEnabled":      llx.BoolDataPtr(org.GetOnPremisesSyncEnabled()),
			"createdAt":                  llx.TimeDataPtr(org.GetCreatedDateTime()),
			"type":                       llx.StringDataPtr(org.GetTenantType()),
			"provisionedPlans":           llx.DictData(provisionedPlans),
			"technicalNotificationMails": llx.ArrayData(convert.SliceAnyToInterface(org.GetTechnicalNotificationMails()), types.String),
			"preferredLanguage":          llx.StringDataPtr(org.GetPreferredLanguage()),
			"privacyProfile":             llx.DictData(privacyProfileDict),
		})
	if err != nil {
		return nil, err
	}
	return mqlResource.(*mqlMicrosoftTenant), nil
}

// https://learn.microsoft.com/en-us/entra/identity/users/licensing-service-plan-reference
func (a *mqlMicrosoftTenant) subscriptions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}
	resp, err := graphClient.Directory().Subscriptions().Get(context.Background(), &directory.SubscriptionsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}

	res := []any{}
	for _, sub := range resp.GetValue() {
		res = append(res, newCompanySubscription(sub))
	}

	return convert.JsonToDictSlice(res)
}

func (a *mqlMicrosoft) tenantDomainName() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return "", err
	}
	ctx := context.Background()
	resp, err := graphClient.Organization().Get(ctx, &organization.OrganizationRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return "", transformError(err)
	}
	tenantDomainName := ""

	for _, org := range resp.GetValue() {
		for _, d := range org.GetVerifiedDomains() {
			if *d.GetIsInitial() {
				tenantDomainName = *d.GetName()
			}
		}
	}

	return tenantDomainName, nil
}

func (a *mqlMicrosoftTenant) settings() (*mqlMicrosoftTenantSettings, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.BetaGraphClient()
	if err != nil {
		return nil, err
	}

	appsAndServicesConfig, err := graphClient.Admin().AppsAndServices().Get(context.Background(), nil)
	if err != nil {
		return nil, transformError(err)
	}

	settingsId := fmt.Sprintf("%s-settings", a.Id.Data)

	if appsAndServicesConfig == nil || appsAndServicesConfig.GetSettings() == nil {
		mqlSettings, err := CreateResource(a.MqlRuntime, "microsoft.tenantSettings",
			map[string]*llx.RawData{
				"__id":                         llx.StringData(settingsId),
				"isAppAndServicesTrialEnabled": llx.BoolData(false),
				"isOfficeStoreEnabled":         llx.BoolData(false),
			})
		if err != nil {
			return nil, err
		}
		return mqlSettings.(*mqlMicrosoftTenantSettings), nil
	}

	mqlSettings, err := CreateResource(a.MqlRuntime, "microsoft.tenantSettings",
		map[string]*llx.RawData{
			"__id":                         llx.StringData(settingsId),
			"isAppAndServicesTrialEnabled": llx.BoolDataPtr(appsAndServicesConfig.GetSettings().GetIsAppAndServicesTrialEnabled()),
			"isOfficeStoreEnabled":         llx.BoolDataPtr(appsAndServicesConfig.GetSettings().GetIsOfficeStoreEnabled()),
		})
	if err != nil {
		return nil, err
	}

	return mqlSettings.(*mqlMicrosoftTenantSettings), nil
}

// Least privileged permissions: OrgSettings-Forms.Read.All
func (a *mqlMicrosoftTenant) formsSettings() (*mqlMicrosoftTenantFormsSettings, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	beatGraphClient, err := conn.BetaGraphClient()
	if err != nil {
		return nil, err
	}

	formsSetting, err := beatGraphClient.Admin().Forms().Get(context.Background(), nil)
	if err != nil {
		return nil, transformError(err)
	}

	if formsSetting == nil {
		return nil, nil
	}

	settings := formsSetting.GetSettings()
	if settings == nil {
		return nil, nil
	}

	formsSettingId := fmt.Sprintf("%s-forms-settings", a.Id.Data)

	formSetting, err := CreateResource(a.MqlRuntime, "microsoft.tenantFormsSettings",
		map[string]*llx.RawData{
			"__id":                                llx.StringData(formsSettingId),
			"isExternalSendFormEnabled":           llx.BoolDataPtr(settings.GetIsExternalSendFormEnabled()),
			"isExternalShareCollaborationEnabled": llx.BoolDataPtr(settings.GetIsExternalShareCollaborationEnabled()),
			"isExternalShareResultEnabled":        llx.BoolDataPtr(settings.GetIsExternalShareResultEnabled()),
			"isExternalShareTemplateEnabled":      llx.BoolDataPtr(settings.GetIsExternalShareTemplateEnabled()),
			"isRecordIdentityByDefaultEnabled":    llx.BoolDataPtr(settings.GetIsRecordIdentityByDefaultEnabled()),
			"isBingImageSearchEnabled":            llx.BoolDataPtr(settings.GetIsBingImageSearchEnabled()),
			"isInOrgFormsPhishingScanEnabled":     llx.BoolDataPtr(settings.GetIsInOrgFormsPhishingScanEnabled()),
		})
	if err != nil {
		return nil, err
	}

	return formSetting.(*mqlMicrosoftTenantFormsSettings), nil
}

// Helper function to get branding info for a specific domain from PowerShell script
func (a *mqlMicrosoftTenant) getBrandingInfoForDomain(domainName string) (*BrandingInfo, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)

	userName := domainName

	fmtScript := fmt.Sprintf(tenantBrandingReport, userName)
	res, err := conn.CheckAndRunPowershellScript(fmtScript)
	if err != nil {
		return nil, err
	}

	var branding BrandingInfo
	if res.ExitStatus == 0 {
		data, err := io.ReadAll(res.Stdout)
		if err != nil {
			return nil, fmt.Errorf("failed to read script output: %w", err)
		}
		str := string(data)
		// Strip away any PowerShell output before the JSON
		idx := strings.IndexByte(str, '{')
		if idx == -1 {
			logger.DebugDumpJSON("tenant-branding-script-output", []byte(str))
			return nil, errors.New("no JSON output found in script response")
		}
		after := str[idx:]
		newData := []byte(after)

		err = json.Unmarshal(newData, &branding)
		if err != nil {
			return nil, fmt.Errorf("failed to parse tenant branding JSON: %w", err)
		}
	} else {
		data, err := io.ReadAll(res.Stderr)
		if err != nil {
			return nil, fmt.Errorf("failed to read script error: %w", err)
		}

		str := string(data)
		logger.DebugDumpJSON("tenant-branding-error", string(data))
		return nil, fmt.Errorf("failed to generate tenant branding report (exit code %d): %s", res.ExitStatus, str)
	}

	return &branding, nil
}

func (a *mqlMicrosoftTenant) branding() ([]any, error) {
	verifiedDomains := a.GetVerifiedDomains()
	if verifiedDomains.Error != nil {
		return nil, verifiedDomains.Error
	}

	if len(verifiedDomains.Data) == 0 {
		return []any{}, nil
	}

	var brandingInfos []any
	for _, domain := range verifiedDomains.Data {
		domainDict, ok := domain.(map[string]any)
		if !ok {
			continue
		}

		domainName, ok := domainDict["name"].(string)
		if !ok || domainName == "" {
			continue
		}

		branding, err := a.getBrandingInfoForDomain(domainName)
		if err != nil {
			logger.DebugDumpJSON("tenant-branding-domain-error", []byte(fmt.Sprintf("domain: %s, error: %v", domainName, err)))
			continue
		}

		brandingId := fmt.Sprintf("%s-branding-info-%s", a.Id.Data, branding.DomainName)

		mqlBranding, err := CreateResource(a.MqlRuntime, "microsoft.tenant.brandingInfo",
			map[string]*llx.RawData{
				"__id":                   llx.StringData(brandingId),
				"keepMeSignedInDisabled": llx.BoolData(branding.KeepMeSignedInDisabled),
				"backgroundColor":        llx.StringData(branding.BackgroundColor),
				"bannerLogo":             llx.StringData(branding.BannerLogo),
				"federationBrandName":    llx.StringData(branding.FederationBrandName),
				"domainName":             llx.StringData(branding.DomainName),
				"login":                  llx.StringData(branding.Login),
			})
		if err != nil {
			logger.DebugDumpJSON("tenant-branding-create-error", []byte(fmt.Sprintf("domain: %s, error: %v", domainName, err)))
			continue
		}

		brandingInfos = append(brandingInfos, mqlBranding)
	}

	return brandingInfos, nil
}
