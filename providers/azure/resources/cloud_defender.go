// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/security/armsecurity"
	security "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/security/armsecurity"
	"github.com/rs/zerolog/log"
)

const (
	vaQualysPolicyDefinitionId string = "/providers/Microsoft.Authorization/policyDefinitions/13ce0167-8ca6-4048-8e6b-f996402e3c1b"
	// There are two policy per component: one for ARC clusters and one for k8s clusters
	arcClusterDefenderExtensionDefinitionId        string = "/providers/Microsoft.Authorization/policyDefinitions/708b60a6-d253-4fe0-9114-4be4c00f012c"
	kubernetesClusterDefenderExtensionDefinitionId string = "/providers/Microsoft.Authorization/policyDefinitions/64def556-fbad-4622-930e-72d1d5589bf5"

	// 0adc5395-... is the Arc-enabled Kubernetes "Azure Policy extension" definition;
	// a8eff44f-... is the AKS "Azure Policy Add-on" definition. They were previously
	// both set to the Arc GUID, so azurePolicyForKubernetes never reflected the AKS
	// add-on assignment.
	arcClusterPolicyExtensionDefinitionId        string = "/providers/Microsoft.Authorization/policyDefinitions/0adc5395-9169-4b9b-8687-af838d69410a"
	kubernetesClusterPolicyExtensionDefinitionId string = "/providers/Microsoft.Authorization/policyDefinitions/a8eff44f-8c92-45c3-a3fb-9880802d67a7"
)

func (a *mqlAzureSubscriptionCloudDefenderService) id() (string, error) {
	return "azure.subscription.cloudDefender/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionCloudDefenderService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceSecurityContact) id() (string, error) {
	return a.Id.Data, nil
}

// commonPricingArgs extracts common pricing fields from an Azure PricingProperties response
// into a map suitable for CreateResource.
func commonPricingArgs(props *security.PricingProperties, mqlResourceName, subId string) map[string]*llx.RawData {
	args := map[string]*llx.RawData{
		"__id":           llx.StringData(mqlResourceName + "/" + subId),
		"subscriptionId": llx.StringData(subId),
	}

	if props == nil {
		props = &security.PricingProperties{}
	}

	enabled := false
	pricingTier := ""
	if props.PricingTier != nil {
		pricingTier = string(*props.PricingTier)
		enabled = *props.PricingTier == security.PricingTierStandard
	}
	args["enabled"] = llx.BoolData(enabled)
	args["pricingTier"] = llx.StringData(pricingTier)

	subPlan := ""
	if props.SubPlan != nil {
		subPlan = *props.SubPlan
	}
	args["subPlan"] = llx.StringData(subPlan)

	enforce := false
	if props.Enforce != nil {
		enforce = *props.Enforce == security.EnforceTrue
	}
	args["enforce"] = llx.BoolData(enforce)

	deprecated := false
	if props.Deprecated != nil {
		deprecated = *props.Deprecated
	}
	args["deprecated"] = llx.BoolData(deprecated)

	freeTrialRemainingTime := ""
	if props.FreeTrialRemainingTime != nil {
		freeTrialRemainingTime = *props.FreeTrialRemainingTime
	}
	args["freeTrialRemainingTime"] = llx.StringData(freeTrialRemainingTime)

	args["enablementTime"] = llx.TimeDataPtr(props.EnablementTime)

	inherited := false
	if props.Inherited != nil {
		inherited = *props.Inherited == security.InheritedTrue
	}
	args["inherited"] = llx.BoolData(inherited)

	inheritedFrom := ""
	if props.InheritedFrom != nil {
		inheritedFrom = *props.InheritedFrom
	}
	args["inheritedFrom"] = llx.StringData(inheritedFrom)

	replacedBy := []any{}
	for _, s := range props.ReplacedBy {
		if s != nil {
			replacedBy = append(replacedBy, *s)
		}
	}
	args["replacedBy"] = llx.ArrayData(replacedBy, types.String)

	resourcesCoverageStatus := ""
	if props.ResourcesCoverageStatus != nil {
		resourcesCoverageStatus = string(*props.ResourcesCoverageStatus)
	}
	args["resourcesCoverageStatus"] = llx.StringData(resourcesCoverageStatus)

	return args
}

// getSimpleDefenderPricing fetches pricing data for a Defender component and creates a typed resource.
func (a *mqlAzureSubscriptionCloudDefenderService) getSimpleDefenderPricing(azurePricingName, mqlResourceName string) (plugin.Resource, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), azurePricingName, &security.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	args := commonPricingArgs(pricing.Properties, mqlResourceName, subId)
	return CreateResource(a.MqlRuntime, mqlResourceName, args)
}

type mqlAzureSubscriptionCloudDefenderServiceInternal struct {
	policyAssignmentsOnce sync.Once
	policyAssignments     PolicyAssignments
	policyAssignmentsErr  error
}

// cachedPolicyAssignments fetches the subscription's policy assignments once and
// caches the result. Both the forServers and forContainers Defender plans read
// the same policy-assignments list to derive their enabled/extension state, so
// without this a scan that touches both plans issues the list call twice.
func (a *mqlAzureSubscriptionCloudDefenderService) cachedPolicyAssignments(ctx context.Context) (PolicyAssignments, error) {
	a.policyAssignmentsOnce.Do(func() {
		conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
		armConn, err := getArmSecurityConnection(ctx, conn, a.SubscriptionId.Data)
		if err != nil {
			a.policyAssignmentsErr = err
			return
		}
		a.policyAssignments, a.policyAssignmentsErr = getPolicyAssignments(ctx, armConn)
	})
	return a.policyAssignments, a.policyAssignmentsErr
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForServers() (any, error) {
	typed := a.GetForServers()
	if typed.Error != nil {
		return nil, typed.Error
	}
	return map[string]any{
		"enabled":                         typed.Data.GetEnabled().Data,
		"vulnerabilityManagementToolName": typed.Data.GetVulnerabilityManagementToolName().Data,
	}, nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) forServers() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForServers, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	clientFactory, err := armsecurity.NewClientFactory(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	vmPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "VirtualMachines", &security.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	armConn, err := getArmSecurityConnection(ctx, conn, subId)
	if err != nil {
		return nil, err
	}
	list, err := a.cachedPolicyAssignments(ctx)
	if err != nil {
		return nil, err
	}
	serverVASetings, err := getServerVulnAssessmentSettings(ctx, armConn)
	if err != nil {
		return nil, err
	}

	args := commonPricingArgs(vmPricing.Properties, ResourceAzureSubscriptionCloudDefenderServiceDefenderForServers, subId)

	// Override enabled based on policy assignments and vulnerability assessment settings
	vulnToolName := ""
	for _, it := range list.PolicyAssignments {
		// Scope the assignment to this subscription; the unfiltered list also
		// returns management-group-inherited assignments, which would otherwise
		// report enabled even when VA is not configured on this subscription
		// (mirrors the scope check in the Defender-for-Containers detection).
		if it.Properties.PolicyDefinitionID == vaQualysPolicyDefinitionId &&
			it.Properties.Scope == fmt.Sprintf("/subscriptions/%s", subId) {
			args["enabled"] = llx.BoolData(true)
			vulnToolName = "Microsoft Defender for Cloud integrated Qualys scanner"
		}
	}
	for _, sett := range serverVASetings.Settings {
		if sett.Properties.SelectedProvider == "MdeTvm" && sett.Name == "AzureServersSetting" {
			args["enabled"] = llx.BoolData(true)
			vulnToolName = "Microsoft Defender vulnerability management"
		}
	}
	args["vulnerabilityManagementToolName"] = llx.StringData(vulnToolName)

	resource, err := CreateResource(a.MqlRuntime,
		ResourceAzureSubscriptionCloudDefenderServiceDefenderForServers,
		args,
	)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForServers), nil
}

// simpleDefenderDict serializes a typed Defender pricing resource's `enabled`
// flag into the shape that the deprecated defenderForX() dict methods used to
// return — `{"enabled": <bool>}`. Letting the dict methods delegate here keeps
// them in sync with the typed equivalent and reuses the PricingsClient.Get
// already issued by the typed resource.
func simpleDefenderDict(enabled *plugin.TValue[bool]) (any, error) {
	if enabled.Error != nil {
		return nil, enabled.Error
	}
	return map[string]any{"enabled": enabled.Data}, nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForAppServices() (any, error) {
	typed := a.GetForAppServices()
	if typed.Error != nil {
		return nil, typed.Error
	}
	return simpleDefenderDict(typed.Data.GetEnabled())
}

func (a *mqlAzureSubscriptionCloudDefenderService) forAppServices() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForAppServices, error) {
	resource, err := a.getSimpleDefenderPricing("AppServices", ResourceAzureSubscriptionCloudDefenderServiceDefenderForAppServices)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForAppServices), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForSqlServersOnMachines() (any, error) {
	typed := a.GetForSqlServersOnMachines()
	if typed.Error != nil {
		return nil, typed.Error
	}
	return simpleDefenderDict(typed.Data.GetEnabled())
}

func (a *mqlAzureSubscriptionCloudDefenderService) forSqlServersOnMachines() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForSqlServersOnMachines, error) {
	resource, err := a.getSimpleDefenderPricing("SqlServerVirtualMachines", ResourceAzureSubscriptionCloudDefenderServiceDefenderForSqlServersOnMachines)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForSqlServersOnMachines), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForSqlDatabases() (any, error) {
	typed := a.GetForSqlDatabases()
	if typed.Error != nil {
		return nil, typed.Error
	}
	return simpleDefenderDict(typed.Data.GetEnabled())
}

func (a *mqlAzureSubscriptionCloudDefenderService) forSqlDatabases() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForSqlDatabases, error) {
	resource, err := a.getSimpleDefenderPricing("SqlServers", ResourceAzureSubscriptionCloudDefenderServiceDefenderForSqlDatabases)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForSqlDatabases), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForOpenSourceDatabases() (any, error) {
	typed := a.GetForOpenSourceDatabases()
	if typed.Error != nil {
		return nil, typed.Error
	}
	return simpleDefenderDict(typed.Data.GetEnabled())
}

func (a *mqlAzureSubscriptionCloudDefenderService) forOpenSourceDatabases() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForOpenSourceDatabases, error) {
	resource, err := a.getSimpleDefenderPricing("OpenSourceRelationalDatabases", ResourceAzureSubscriptionCloudDefenderServiceDefenderForOpenSourceDatabases)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForOpenSourceDatabases), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForCosmosDb() (any, error) {
	typed := a.GetForCosmosDb()
	if typed.Error != nil {
		return nil, typed.Error
	}
	return simpleDefenderDict(typed.Data.GetEnabled())
}

func (a *mqlAzureSubscriptionCloudDefenderService) forCosmosDb() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForCosmosDb, error) {
	resource, err := a.getSimpleDefenderPricing("CosmosDbs", ResourceAzureSubscriptionCloudDefenderServiceDefenderForCosmosDb)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForCosmosDb), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForStorageAccounts() (any, error) {
	typed := a.GetForStorageAccounts()
	if typed.Error != nil {
		return nil, typed.Error
	}
	return simpleDefenderDict(typed.Data.GetEnabled())
}

func (a *mqlAzureSubscriptionCloudDefenderService) forStorageAccounts() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForStorageAccounts, error) {
	resource, err := a.getSimpleDefenderPricing("StorageAccounts", ResourceAzureSubscriptionCloudDefenderServiceDefenderForStorageAccounts)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForStorageAccounts), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForKeyVaults() (any, error) {
	typed := a.GetForKeyVaults()
	if typed.Error != nil {
		return nil, typed.Error
	}
	return simpleDefenderDict(typed.Data.GetEnabled())
}

func (a *mqlAzureSubscriptionCloudDefenderService) forKeyVaults() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForKeyVaults, error) {
	resource, err := a.getSimpleDefenderPricing("KeyVaults", ResourceAzureSubscriptionCloudDefenderServiceDefenderForKeyVaults)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForKeyVaults), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForResourceManager() (any, error) {
	typed := a.GetForResourceManager()
	if typed.Error != nil {
		return nil, typed.Error
	}
	return simpleDefenderDict(typed.Data.GetEnabled())
}

func (a *mqlAzureSubscriptionCloudDefenderService) forResourceManager() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForResourceManager, error) {
	resource, err := a.getSimpleDefenderPricing("Arm", ResourceAzureSubscriptionCloudDefenderServiceDefenderForResourceManager)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForResourceManager), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForApis() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForApis, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	apiPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "Api", &security.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	args := commonPricingArgs(apiPricing.Properties, ResourceAzureSubscriptionCloudDefenderServiceDefenderForApis, subId)

	resource, err := CreateResource(a.MqlRuntime,
		ResourceAzureSubscriptionCloudDefenderServiceDefenderForApis,
		args,
	)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForApis), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderCSPM() (*mqlAzureSubscriptionCloudDefenderServiceDefenderCSPM, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	cloudPosturePricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "CloudPosture", &security.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	args := commonPricingArgs(cloudPosturePricing.Properties, ResourceAzureSubscriptionCloudDefenderServiceDefenderCSPM, subId)

	resource, err := CreateResource(a.MqlRuntime,
		ResourceAzureSubscriptionCloudDefenderServiceDefenderCSPM,
		args,
	)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderCSPM), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) monitoringAgentAutoProvision() (bool, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := security.NewAutoProvisioningSettingsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return false, err
	}

	setting, err := client.Get(ctx, "default", &security.AutoProvisioningSettingsClientGetOptions{})
	if err != nil {
		return false, err
	}
	if setting.Properties == nil || setting.Properties.AutoProvision == nil {
		return false, nil
	}
	return *setting.Properties.AutoProvision == security.AutoProvisionOn, nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForContainers() (any, error) {
	typed := a.GetForContainers()
	if typed.Error != nil {
		return nil, typed.Error
	}

	rawExts, err := typed.Data.fetchRawExtensions()
	if err != nil {
		return nil, err
	}
	exts := make([]map[string]any, 0, len(rawExts))
	for _, ext := range rawExts {
		if ext.IsEnabled == nil || ext.Name == nil {
			continue
		}
		exts = append(exts, map[string]any{
			"name":      *ext.Name,
			"isEnabled": *ext.IsEnabled == security.IsEnabledTrue,
		})
	}

	return map[string]any{
		"defenderDaemonSet":        typed.Data.GetDefenderDaemonSet().Data,
		"azurePolicyForKubernetes": typed.Data.GetAzurePolicyForKubernetes().Data,
		"enabled":                  typed.Data.GetEnabled().Data,
		"extensions":               exts,
	}, nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) forContainers() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForContainers, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := a.SubscriptionId.Data

	armConn, err := getArmSecurityConnection(ctx, conn, subId)
	if err != nil {
		return nil, err
	}

	pas, err := a.cachedPolicyAssignments(ctx)
	if err != nil {
		return nil, err
	}

	kubernetesDefender := false
	arcDefender := false
	kubernetesPolicyExt := false
	arcPolicyExt := false
	for _, it := range pas.PolicyAssignments {
		if it.Properties.PolicyDefinitionID == arcClusterDefenderExtensionDefinitionId &&
			it.Properties.Scope == fmt.Sprintf("/subscriptions/%s", subId) {
			arcDefender = true
		}
		if it.Properties.PolicyDefinitionID == kubernetesClusterDefenderExtensionDefinitionId &&
			it.Properties.Scope == fmt.Sprintf("/subscriptions/%s", subId) {
			kubernetesDefender = true
		}
		if it.Properties.PolicyDefinitionID == arcClusterPolicyExtensionDefinitionId &&
			it.Properties.Scope == fmt.Sprintf("/subscriptions/%s", subId) {
			arcPolicyExt = true
		}
		if it.Properties.PolicyDefinitionID == kubernetesClusterPolicyExtensionDefinitionId &&
			it.Properties.Scope == fmt.Sprintf("/subscriptions/%s", subId) {
			kubernetesPolicyExt = true
		}
	}

	// Check if Defender for Containers is enabled by querying the pricing tier
	clientFactory, err := armsecurity.NewClientFactory(subId, armConn.token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	containersPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "Containers", &security.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	args := commonPricingArgs(containersPricing.Properties, ResourceAzureSubscriptionCloudDefenderServiceDefenderForContainers, subId)
	args["defenderDaemonSet"] = llx.BoolData(arcDefender && kubernetesDefender)
	args["azurePolicyForKubernetes"] = llx.BoolData(arcPolicyExt && kubernetesPolicyExt)

	resource, err := CreateResource(a.MqlRuntime,
		ResourceAzureSubscriptionCloudDefenderServiceDefenderForContainers,
		args,
	)
	if err != nil {
		return nil, err
	}
	mqlContainers := resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForContainers)
	var rawExtensions []*security.Extension
	if containersPricing.Properties != nil {
		rawExtensions = containersPricing.Properties.Extensions
	}
	mqlContainers.rawExtensionsOnce.Do(func() {
		mqlContainers.rawExtensions = rawExtensions
	})
	return mqlContainers, nil
}

type mqlAzureSubscriptionCloudDefenderServiceDefenderForContainersInternal struct {
	rawExtensionsOnce sync.Once
	rawExtensions     []*security.Extension
	rawExtensionsErr  error
}

// fetchRawExtensions returns the raw extension list from the Containers
// pricing endpoint. Cached with sync.Once so the typed extensions() accessor
// and the deprecated defenderForContainers() dict share a single fetch (the
// parent forContainers() primes the cache; standalone access falls back to
// the API).
func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderForContainers) fetchRawExtensions() ([]*security.Extension, error) {
	a.rawExtensionsOnce.Do(func() {
		conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
		subId := a.SubscriptionId.Data
		clientFactory, err := armsecurity.NewClientFactory(subId, conn.Token(), &arm.ClientOptions{
			ClientOptions: conn.ClientOptions(),
		})
		if err != nil {
			a.rawExtensionsErr = err
			return
		}
		resp, err := clientFactory.NewPricingsClient().Get(context.Background(),
			fmt.Sprintf("subscriptions/%s", subId), "Containers",
			&security.PricingsClientGetOptions{})
		if err != nil {
			a.rawExtensionsErr = err
			return
		}
		if resp.Properties == nil {
			return
		}
		a.rawExtensions = resp.Properties.Extensions
	})
	return a.rawExtensions, a.rawExtensionsErr
}

func (a *mqlAzureSubscriptionCloudDefenderService) settingsMCAS() (*mqlAzureSubscriptionCloudDefenderServiceSettings, error) {
	return a.getSecuritySettingsFor(security.SettingNameMCAS)
}

func (a *mqlAzureSubscriptionCloudDefenderService) settingsWDATP() (*mqlAzureSubscriptionCloudDefenderServiceSettings, error) {
	return a.getSecuritySettingsFor(security.SettingNameWDATP)
}

func (a *mqlAzureSubscriptionCloudDefenderService) settingsSentinel() (*mqlAzureSubscriptionCloudDefenderServiceSettings, error) {
	return a.getSecuritySettingsFor(security.SettingNameSentinel)
}

func (a *mqlAzureSubscriptionCloudDefenderService) getSecuritySettingsFor(name security.SettingName) (*mqlAzureSubscriptionCloudDefenderServiceSettings, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	clientFactory, err := armsecurity.NewClientFactory(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	settingResp, err := clientFactory.NewSettingsClient().Get(ctx, name, nil)
	if err != nil {
		return nil, err
	}

	baseSetting := settingResp.SettingClassification.GetSetting()
	if baseSetting == nil || baseSetting.Kind == nil {
		return nil, fmt.Errorf("retrieved setting or its kind is nil for '%s'", name)
	}

	switch *baseSetting.Kind {
	case armsecurity.SettingKindDataExportSettings:
		// Handles MCAS and Sentinel
		settings, ok := settingResp.SettingClassification.(*armsecurity.DataExportSettings)
		if !ok {
			return nil, fmt.Errorf("failed assertion to DataExportSettings for kind '%s', setting '%s'. Actual type: %T", *baseSetting.Kind, name, settingResp.SettingClassification)
		}
		properties, err := convert.JsonToDict(settings.Properties)
		if err != nil {
			return nil, err
		}
		resource, err := CreateResource(a.MqlRuntime,
			"azure.subscription.cloudDefenderService.settings",
			map[string]*llx.RawData{
				"id":         llx.StringDataPtr(settings.ID),
				"name":       llx.StringDataPtr(settings.Name),
				"kind":       llx.StringDataPtr((*string)(settings.Kind)),
				"type":       llx.StringDataPtr(settings.Type),
				"properties": llx.DictData(properties),
			},
		)
		if err != nil {
			return nil, err
		}
		return resource.(*mqlAzureSubscriptionCloudDefenderServiceSettings), nil
	case armsecurity.SettingKindAlertSyncSettings:
		// Handles WDATP
		settings, ok := settingResp.SettingClassification.(*armsecurity.AlertSyncSettings)
		if !ok {
			return nil, fmt.Errorf("failed assertion to AlertSyncSettings for kind '%s', setting '%s'. Actual type: %T", *baseSetting.Kind, name, settingResp.SettingClassification)
		}
		properties, err := convert.JsonToDict(settings.Properties)
		if err != nil {
			return nil, err
		}
		resource, err := CreateResource(a.MqlRuntime,
			"azure.subscription.cloudDefenderService.settings",
			map[string]*llx.RawData{
				"id":         llx.StringDataPtr(settings.ID),
				"name":       llx.StringDataPtr(settings.Name),
				"kind":       llx.StringDataPtr((*string)(settings.Kind)),
				"type":       llx.StringDataPtr(settings.Type),
				"properties": llx.DictData(properties),
			},
		)
		if err != nil {
			return nil, err
		}
		return resource.(*mqlAzureSubscriptionCloudDefenderServiceSettings), nil
	default:
		return nil, fmt.Errorf("unsupported settings '%s' of kind '%s'", name, *baseSetting.Kind)
	}
}

func (s *mqlAzureSubscriptionCloudDefenderServiceSettings) id() (string, error) {
	return s.Id.Data, nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) securityContacts() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	clientFactory, err := armsecurity.NewClientFactory(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := clientFactory.NewContactsClient().NewListPager(nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, contact := range page.ContactList.Value {
			args := argsFromContactProperties(contact.Properties)
			args["id"] = llx.StringDataPtr(contact.ID)
			args["name"] = llx.StringDataPtr(contact.Name)

			mqlSecurityContact, err := CreateResource(
				a.MqlRuntime,
				"azure.subscription.cloudDefenderService.securityContact",
				args,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlSecurityContact)
		}
	}
	return res, nil
}

func argsFromContactProperties(props *armsecurity.ContactProperties) map[string]*llx.RawData {
	args := map[string]*llx.RawData{}
	if props == nil {
		return args
	}

	sources := map[string]any{}
	for _, source := range props.NotificationsSources {
		notificationSource := source.GetNotificationsSource()
		if notificationSource == nil || notificationSource.SourceType == nil {
			continue
		}

		sourceDict, err := convert.JsonToDict(source)
		if err != nil {
			log.Debug().Err(err).Msg("unable to convert armsecurity.props.NotificationsSources to dict")
			continue
		}

		if notificationSource.SourceType == nil {
			continue
		}
		sourceType := *notificationSource.SourceType
		sources[string(sourceType)] = sourceDict
	}
	args["notificationSources"] = llx.DictData(sources)

	notificationsByRole, err := convert.JsonToDict(props.NotificationsByRole)
	if err != nil {
		log.Debug().Err(err).Msg("unable to convert armsecurity.Contact.Properties.NotificationsByRole to dict")
	}
	args["notificationsByRole"] = llx.DictData(notificationsByRole)

	// emails
	mails := ""
	if props.Emails != nil {
		mails = *props.Emails
	}
	mailsArr := strings.Split(mails, ";")
	args["emails"] = llx.ArrayData(convert.SliceAnyToInterface(mailsArr), types.String)

	args["isEnabled"] = llx.BoolDataPtr(props.IsEnabled)
	args["phone"] = llx.StringDataPtr(props.Phone)

	return args
}

// buildExtensionResources creates typed extension sub-resources from a list of Azure SDK extensions.
func buildExtensionResources(runtime *plugin.Runtime, extensions []*security.Extension, mqlResourceName, parentIdPrefix string) ([]any, error) {
	res := []any{}
	for _, ext := range extensions {
		if ext == nil {
			continue
		}

		name := ""
		if ext.Name != nil {
			name = *ext.Name
		}

		isEnabled := false
		if ext.IsEnabled != nil {
			isEnabled = *ext.IsEnabled == security.IsEnabledTrue
		}

		additionalProps, err := convert.JsonToDict(ext.AdditionalExtensionProperties)
		if err != nil {
			log.Debug().Err(err).Str("extension", name).Msg("unable to convert extension additional properties to dict")
			additionalProps = nil
		}

		opCode := ""
		opMessage := ""
		if ext.OperationStatus != nil {
			if ext.OperationStatus.Code != nil {
				opCode = string(*ext.OperationStatus.Code)
			}
			if ext.OperationStatus.Message != nil {
				opMessage = *ext.OperationStatus.Message
			}
		}

		extResource, err := CreateResource(runtime, mqlResourceName, map[string]*llx.RawData{
			"__id":                   llx.StringData(parentIdPrefix + "/extension/" + name),
			"name":                   llx.StringData(name),
			"isEnabled":              llx.BoolData(isEnabled),
			"additionalProperties":   llx.DictData(additionalProps),
			"operationStatusCode":    llx.StringData(opCode),
			"operationStatusMessage": llx.StringData(opMessage),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, extResource)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderForApis) id() (string, error) {
	return a.__id, nil
}

// extensions lazily fetches extension data for Defender CSPM. This re-fetches the CloudPosture
// pricing data (already retrieved by defenderCSPM) so that extensions are only loaded when accessed.
func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderCSPM) extensions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	cloudPosturePricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "CloudPosture", &security.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	if cloudPosturePricing.Properties == nil {
		return []any{}, nil
	}

	return buildExtensionResources(a.MqlRuntime, cloudPosturePricing.Properties.Extensions,
		ResourceAzureSubscriptionCloudDefenderServiceDefenderCSPMExtension,
		ResourceAzureSubscriptionCloudDefenderServiceDefenderCSPM+"/"+subId)
}

func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderCSPM) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderCSPMExtension) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderForServers) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderForAppServices) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderForSqlServersOnMachines) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderForSqlDatabases) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderForOpenSourceDatabases) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderForCosmosDb) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderForStorageAccounts) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderForKeyVaults) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderForResourceManager) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderForContainers) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderForContainers) extensions() ([]any, error) {
	exts, err := a.fetchRawExtensions()
	if err != nil {
		return nil, err
	}
	return buildExtensionResources(a.MqlRuntime, exts,
		ResourceAzureSubscriptionCloudDefenderServiceDefenderForContainersExtension,
		ResourceAzureSubscriptionCloudDefenderServiceDefenderForContainers+"/"+a.SubscriptionId.Data)
}

func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderForContainersExtension) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceSecureScore) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceSecureScoreControl) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceRegulatoryComplianceStandard) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceRegulatoryComplianceControl) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) secureScores() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := clientFactory.NewSecureScoresClient().NewListPager(nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list secure scores due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, item := range page.Value {
			var displayName string
			var currentScore float64
			var maxScore int64
			var percentage float64
			var weight int64

			if item.Properties != nil {
				if item.Properties.DisplayName != nil {
					displayName = *item.Properties.DisplayName
				}
				if item.Properties.Score != nil {
					if item.Properties.Score.Current != nil {
						currentScore = *item.Properties.Score.Current
					}
					if item.Properties.Score.Max != nil {
						maxScore = int64(*item.Properties.Score.Max)
					}
					if item.Properties.Score.Percentage != nil {
						percentage = *item.Properties.Score.Percentage
					}
				}
				if item.Properties.Weight != nil {
					weight = *item.Properties.Weight
				}
			}

			mqlResource, err := CreateResource(a.MqlRuntime,
				ResourceAzureSubscriptionCloudDefenderServiceSecureScore,
				map[string]*llx.RawData{
					"__id":         llx.StringDataPtr(item.ID),
					"id":           llx.StringDataPtr(item.ID),
					"name":         llx.StringDataPtr(item.Name),
					"displayName":  llx.StringData(displayName),
					"currentScore": llx.FloatData(currentScore),
					"maxScore":     llx.IntData(maxScore),
					"percentage":   llx.FloatData(percentage),
					"weight":       llx.IntData(weight),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlResource)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) secureScoreControls() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := clientFactory.NewSecureScoreControlsClient().NewListPager(nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list secure score controls due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, item := range page.Value {
			var displayName string
			var description string
			var currentScore float64
			var maxScore int64
			var percentage float64
			var weight int64
			var healthyResourceCount int64
			var unhealthyResourceCount int64
			var notApplicableResourceCount int64

			if item.Properties != nil {
				if item.Properties.DisplayName != nil {
					displayName = *item.Properties.DisplayName
				}
				if item.Properties.Score != nil {
					if item.Properties.Score.Current != nil {
						currentScore = *item.Properties.Score.Current
					}
					if item.Properties.Score.Max != nil {
						maxScore = int64(*item.Properties.Score.Max)
					}
					if item.Properties.Score.Percentage != nil {
						percentage = *item.Properties.Score.Percentage
					}
				}
				if item.Properties.Weight != nil {
					weight = int64(*item.Properties.Weight)
				}
				if item.Properties.HealthyResourceCount != nil {
					healthyResourceCount = int64(*item.Properties.HealthyResourceCount)
				}
				if item.Properties.UnhealthyResourceCount != nil {
					unhealthyResourceCount = int64(*item.Properties.UnhealthyResourceCount)
				}
				if item.Properties.NotApplicableResourceCount != nil {
					notApplicableResourceCount = int64(*item.Properties.NotApplicableResourceCount)
				}
				if item.Properties.Definition != nil && item.Properties.Definition.Properties != nil && item.Properties.Definition.Properties.Description != nil {
					description = *item.Properties.Definition.Properties.Description
				}
			}

			mqlResource, err := CreateResource(a.MqlRuntime,
				ResourceAzureSubscriptionCloudDefenderServiceSecureScoreControl,
				map[string]*llx.RawData{
					"__id":                       llx.StringDataPtr(item.ID),
					"id":                         llx.StringDataPtr(item.ID),
					"name":                       llx.StringDataPtr(item.Name),
					"displayName":                llx.StringData(displayName),
					"description":                llx.StringData(description),
					"currentScore":               llx.FloatData(currentScore),
					"maxScore":                   llx.IntData(maxScore),
					"percentage":                 llx.FloatData(percentage),
					"weight":                     llx.IntData(weight),
					"healthyResourceCount":       llx.IntData(healthyResourceCount),
					"unhealthyResourceCount":     llx.IntData(unhealthyResourceCount),
					"notApplicableResourceCount": llx.IntData(notApplicableResourceCount),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlResource)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) regulatoryComplianceStandards() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := clientFactory.NewRegulatoryComplianceStandardsClient().NewListPager(nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list regulatory compliance standards due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, item := range page.Value {
			var state string
			var passedControls int64
			var failedControls int64
			var skippedControls int64

			if item.Properties != nil {
				if item.Properties.State != nil {
					state = string(*item.Properties.State)
				}
				if item.Properties.PassedControls != nil {
					passedControls = int64(*item.Properties.PassedControls)
				}
				if item.Properties.FailedControls != nil {
					failedControls = int64(*item.Properties.FailedControls)
				}
				if item.Properties.SkippedControls != nil {
					skippedControls = int64(*item.Properties.SkippedControls)
				}
			}

			mqlResource, err := CreateResource(a.MqlRuntime,
				ResourceAzureSubscriptionCloudDefenderServiceRegulatoryComplianceStandard,
				map[string]*llx.RawData{
					"__id":            llx.StringDataPtr(item.ID),
					"id":              llx.StringDataPtr(item.ID),
					"name":            llx.StringDataPtr(item.Name),
					"state":           llx.StringData(state),
					"passedControls":  llx.IntData(passedControls),
					"failedControls":  llx.IntData(failedControls),
					"skippedControls": llx.IntData(skippedControls),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlResource)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceRegulatoryComplianceStandard) controls() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()

	// Extract subscription ID from the standard's ID path
	// Format: /subscriptions/{subId}/providers/Microsoft.Security/regulatoryComplianceStandards/{standardName}
	standardId := a.Id.Data
	parts := strings.Split(standardId, "/")
	var subId string
	for i, p := range parts {
		if strings.EqualFold(p, "subscriptions") && i+1 < len(parts) {
			subId = parts[i+1]
			break
		}
	}
	if subId == "" {
		subId = conn.SubId()
	}

	standardName := a.Name.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := clientFactory.NewRegulatoryComplianceControlsClient().NewListPager(standardName, nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list regulatory compliance controls due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, item := range page.Value {
			var state string
			var description string
			var passedAssessments int64
			var failedAssessments int64
			var skippedAssessments int64

			if item.Properties != nil {
				if item.Properties.State != nil {
					state = string(*item.Properties.State)
				}
				if item.Properties.Description != nil {
					description = *item.Properties.Description
				}
				if item.Properties.PassedAssessments != nil {
					passedAssessments = int64(*item.Properties.PassedAssessments)
				}
				if item.Properties.FailedAssessments != nil {
					failedAssessments = int64(*item.Properties.FailedAssessments)
				}
				if item.Properties.SkippedAssessments != nil {
					skippedAssessments = int64(*item.Properties.SkippedAssessments)
				}
			}

			mqlResource, err := CreateResource(a.MqlRuntime,
				ResourceAzureSubscriptionCloudDefenderServiceRegulatoryComplianceControl,
				map[string]*llx.RawData{
					"__id":               llx.StringDataPtr(item.ID),
					"id":                 llx.StringDataPtr(item.ID),
					"name":               llx.StringDataPtr(item.Name),
					"description":        llx.StringData(description),
					"state":              llx.StringData(state),
					"passedAssessments":  llx.IntData(passedAssessments),
					"failedAssessments":  llx.IntData(failedAssessments),
					"skippedAssessments": llx.IntData(skippedAssessments),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlResource)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceAssessment) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceAlert) id() (string, error) {
	return a.__id, nil
}

// assessmentMetadata holds the catalog metadata for a single assessment
// definition, joined into the per-assessment list results by name.
type assessmentMetadata struct {
	severity               string
	assessmentType         string
	implementationEffort   string
	userImpact             string
	remediationDescription string
	description            string
	preview                bool
	categories             []any
	threats                []any
	tactics                []any
	techniques             []any
}

// enumSliceToInterface converts a slice of pointers to string-backed enum
// values into a slice of plain strings for llx.ArrayData.
func enumSliceToInterface[T ~string](in []*T) []any {
	out := make([]any, 0, len(in))
	for _, v := range in {
		if v != nil {
			out = append(out, string(*v))
		}
	}
	return out
}

// assessmentAttackPaths converts the Defender risk attack-path graph into a
// slice of {id, nodes, edges} dicts. Each node is {id, labels} and each edge
// is {id, sourceId, targetId}.
func assessmentAttackPaths(paths []*armsecurity.AssessmentPropertiesBaseRiskPathsItem) []any {
	out := make([]any, 0, len(paths))
	for _, p := range paths {
		if p == nil {
			continue
		}
		nodes := make([]any, 0, len(p.Nodes))
		for _, n := range p.Nodes {
			if n == nil {
				continue
			}
			nodes = append(nodes, map[string]any{
				"id":     convert.ToValue(n.ID),
				"labels": enumSliceToInterface(n.NodePropertiesLabel),
			})
		}
		edges := make([]any, 0, len(p.Edges))
		for _, e := range p.Edges {
			if e == nil {
				continue
			}
			edges = append(edges, map[string]any{
				"id":       convert.ToValue(e.ID),
				"sourceId": convert.ToValue(e.SourceID),
				"targetId": convert.ToValue(e.TargetID),
			})
		}
		out = append(out, map[string]any{
			"id":    convert.ToValue(p.ID),
			"nodes": nodes,
			"edges": edges,
		})
	}
	return out
}

// assessmentMetadataByName returns a map of assessment name to its catalog
// metadata. Assessment list responses don't carry this metadata, so it is
// joined in from the metadata catalog: built-in definitions come from the
// global list, custom ones from the subscription-scoped list.
func assessmentMetadataByName(ctx context.Context, clientFactory *armsecurity.ClientFactory) map[string]assessmentMetadata {
	metadata := map[string]assessmentMetadata{}
	metadataClient := clientFactory.NewAssessmentsMetadataClient()

	collect := func(items []*armsecurity.AssessmentMetadataResponse) {
		for _, item := range items {
			if item.Name == nil || item.Properties == nil {
				continue
			}
			p := item.Properties
			metadata[*item.Name] = assessmentMetadata{
				severity:               string(convert.ToValue(p.Severity)),
				assessmentType:         string(convert.ToValue(p.AssessmentType)),
				implementationEffort:   string(convert.ToValue(p.ImplementationEffort)),
				userImpact:             string(convert.ToValue(p.UserImpact)),
				remediationDescription: convert.ToValue(p.RemediationDescription),
				description:            convert.ToValue(p.Description),
				preview:                convert.ToValue(p.Preview),
				categories:             enumSliceToInterface(p.Categories),
				threats:                enumSliceToInterface(p.Threats),
				tactics:                enumSliceToInterface(p.Tactics),
				techniques:             enumSliceToInterface(p.Techniques),
			}
		}
	}

	builtinPager := metadataClient.NewListPager(nil)
	for builtinPager.More() {
		page, err := builtinPager.NextPage(ctx)
		if err != nil {
			log.Debug().Err(err).Msg("could not list built-in assessment metadata")
			break
		}
		collect(page.Value)
	}

	subPager := metadataClient.NewListBySubscriptionPager(nil)
	for subPager.More() {
		page, err := subPager.NextPage(ctx)
		if err != nil {
			log.Debug().Err(err).Msg("could not list subscription assessment metadata")
			break
		}
		collect(page.Value)
	}

	return metadata
}

func (a *mqlAzureSubscriptionCloudDefenderService) assessments() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	metaByName := assessmentMetadataByName(ctx, clientFactory)

	pager := clientFactory.NewAssessmentsClient().NewListPager("/subscriptions/"+subId, nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list security assessments due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, item := range page.Value {
			var displayName, status, statusCause, statusDescription string
			var firstEvaluationDate, statusChangeDate *time.Time
			additionalData := map[string]any{}

			var riskLevel string
			var riskIsContextual bool
			riskFactors := []any{}
			riskAttackPathRefs := []any{}
			riskAttackPaths := []any{}

			if props := item.Properties; props != nil {
				if props.DisplayName != nil {
					displayName = *props.DisplayName
				}
				if props.Status != nil {
					if props.Status.Code != nil {
						status = string(*props.Status.Code)
					}
					if props.Status.Cause != nil {
						statusCause = *props.Status.Cause
					}
					if props.Status.Description != nil {
						statusDescription = *props.Status.Description
					}
					firstEvaluationDate = props.Status.FirstEvaluationDate
					statusChangeDate = props.Status.StatusChangeDate
				}
				for k, v := range props.AdditionalData {
					if v != nil {
						additionalData[k] = *v
					}
				}
				if risk := props.Risk; risk != nil {
					riskLevel = string(convert.ToValue(risk.Level))
					riskIsContextual = convert.ToValue(risk.IsContextualRisk)
					riskFactors = enumSliceToInterface(risk.RiskFactors)
					riskAttackPathRefs = enumSliceToInterface(risk.AttackPathsReferences)
					riskAttackPaths = assessmentAttackPaths(risk.Paths)
				}
			}

			// The assessed resource is the segment of the assessment ID before the
			// Microsoft.Security/assessments path; for subscription-wide assessments
			// that is the subscription itself.
			id := ""
			if item.ID != nil {
				id = *item.ID
			}
			resourceId := strings.SplitN(id, "/providers/Microsoft.Security/assessments/", 2)[0]

			meta := assessmentMetadata{}
			if item.Name != nil {
				meta = metaByName[*item.Name]
			}

			mqlResource, err := CreateResource(a.MqlRuntime,
				"azure.subscription.cloudDefenderService.assessment",
				map[string]*llx.RawData{
					"__id":                     llx.StringDataPtr(item.ID),
					"id":                       llx.StringDataPtr(item.ID),
					"name":                     llx.StringDataPtr(item.Name),
					"displayName":              llx.StringData(displayName),
					"status":                   llx.StringData(status),
					"statusCause":              llx.StringData(statusCause),
					"statusDescription":        llx.StringData(statusDescription),
					"firstEvaluationDate":      llx.TimeDataPtr(firstEvaluationDate),
					"statusChangeDate":         llx.TimeDataPtr(statusChangeDate),
					"severity":                 llx.StringData(meta.severity),
					"resourceId":               llx.StringData(resourceId),
					"additionalData":           llx.DictData(additionalData),
					"riskLevel":                llx.StringData(riskLevel),
					"riskFactors":              llx.ArrayData(riskFactors, types.String),
					"riskAttackPathReferences": llx.ArrayData(riskAttackPathRefs, types.String),
					"riskAttackPaths":          llx.ArrayData(riskAttackPaths, types.Dict),
					"riskIsContextual":         llx.BoolData(riskIsContextual),
					"assessmentType":           llx.StringData(meta.assessmentType),
					"categories":               llx.ArrayData(meta.categories, types.String),
					"threats":                  llx.ArrayData(meta.threats, types.String),
					"tactics":                  llx.ArrayData(meta.tactics, types.String),
					"techniques":               llx.ArrayData(meta.techniques, types.String),
					"implementationEffort":     llx.StringData(meta.implementationEffort),
					"userImpact":               llx.StringData(meta.userImpact),
					"remediationDescription":   llx.StringData(meta.remediationDescription),
					"preview":                  llx.BoolData(meta.preview),
					"description":              llx.StringData(meta.description),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlResource)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) alerts() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := clientFactory.NewAlertsClient().NewListPager(nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list security alerts due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, item := range page.Value {
			var displayName, severity, status, description, alertType string
			var intent, vendorName, compromisedEntity string
			var startTime, endTime, timeGenerated *time.Time
			remediationSteps := []any{}
			resourceIdentifiers := []any{}

			if props := item.Properties; props != nil {
				if props.AlertDisplayName != nil {
					displayName = *props.AlertDisplayName
				}
				if props.Severity != nil {
					severity = string(*props.Severity)
				}
				if props.Status != nil {
					status = string(*props.Status)
				}
				if props.Description != nil {
					description = *props.Description
				}
				if props.AlertType != nil {
					alertType = *props.AlertType
				}
				if props.Intent != nil {
					intent = string(*props.Intent)
				}
				if props.VendorName != nil {
					vendorName = *props.VendorName
				}
				if props.CompromisedEntity != nil {
					compromisedEntity = *props.CompromisedEntity
				}
				for _, step := range props.RemediationSteps {
					if step != nil {
						remediationSteps = append(remediationSteps, *step)
					}
				}
				for _, ri := range props.ResourceIdentifiers {
					riDict, err := convert.JsonToDict(ri)
					if err != nil {
						log.Debug().Err(err).Msg("unable to convert alert resource identifier to dict")
						continue
					}
					resourceIdentifiers = append(resourceIdentifiers, riDict)
				}
				startTime = props.StartTimeUTC
				endTime = props.EndTimeUTC
				timeGenerated = props.TimeGeneratedUTC
			}

			mqlResource, err := CreateResource(a.MqlRuntime,
				"azure.subscription.cloudDefenderService.alert",
				map[string]*llx.RawData{
					"__id":                llx.StringDataPtr(item.ID),
					"id":                  llx.StringDataPtr(item.ID),
					"name":                llx.StringDataPtr(item.Name),
					"displayName":         llx.StringData(displayName),
					"severity":            llx.StringData(severity),
					"status":              llx.StringData(status),
					"description":         llx.StringData(description),
					"alertType":           llx.StringData(alertType),
					"intent":              llx.StringData(intent),
					"vendorName":          llx.StringData(vendorName),
					"compromisedEntity":   llx.StringData(compromisedEntity),
					"startTime":           llx.TimeDataPtr(startTime),
					"endTime":             llx.TimeDataPtr(endTime),
					"timeGenerated":       llx.TimeDataPtr(timeGenerated),
					"remediationSteps":    llx.ArrayData(remediationSteps, types.String),
					"resourceIdentifiers": llx.ArrayData(resourceIdentifiers, types.Dict),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlResource)
		}
	}
	return res, nil
}

// subAssessments resolves the detailed findings underlying a security
// assessment (per-CVE, per-misconfiguration). Sub-assessments are keyed by the
// assessed resource scope plus the assessment name, both of which the parent
// assessment already carries.
func (a *mqlAzureSubscriptionCloudDefenderServiceAssessment) subAssessments() ([]any, error) {
	scope := a.ResourceId.Data
	assessmentName := a.Name.Data
	if scope == "" || assessmentName == "" {
		return []any{}, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	clientFactory, err := armsecurity.NewClientFactory(conn.SubId(), conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	client := clientFactory.NewSubAssessmentsClient()
	pager := client.NewListPager(scope, assessmentName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, sub := range page.Value {
			if sub == nil {
				continue
			}
			var displayName, vulnerabilityId, status, severity, statusCause, statusDescription string
			var category, description, impact, remediation string
			var timeGenerated *time.Time
			resourceDetails := map[string]any{}
			additionalData := map[string]any{}
			if p := sub.Properties; p != nil {
				displayName = convert.ToValue(p.DisplayName)
				vulnerabilityId = convert.ToValue(p.ID)
				category = convert.ToValue(p.Category)
				description = convert.ToValue(p.Description)
				impact = convert.ToValue(p.Impact)
				remediation = convert.ToValue(p.Remediation)
				timeGenerated = p.TimeGenerated
				if s := p.Status; s != nil {
					if s.Code != nil {
						status = string(*s.Code)
					}
					if s.Severity != nil {
						severity = string(*s.Severity)
					}
					statusCause = convert.ToValue(s.Cause)
					statusDescription = convert.ToValue(s.Description)
				}
				if rd, err := convert.JsonToDict(p.ResourceDetails); err == nil {
					resourceDetails = rd
				}
				if ad, err := convert.JsonToDict(p.AdditionalData); err == nil {
					additionalData = ad
				}
			}
			mqlSub, err := CreateResource(a.MqlRuntime, "azure.subscription.cloudDefenderService.assessment.subAssessment",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(sub.ID),
					"name":              llx.StringDataPtr(sub.Name),
					"displayName":       llx.StringData(displayName),
					"vulnerabilityId":   llx.StringData(vulnerabilityId),
					"status":            llx.StringData(status),
					"severity":          llx.StringData(severity),
					"statusCause":       llx.StringData(statusCause),
					"statusDescription": llx.StringData(statusDescription),
					"category":          llx.StringData(category),
					"description":       llx.StringData(description),
					"impact":            llx.StringData(impact),
					"remediation":       llx.StringData(remediation),
					"timeGenerated":     llx.TimeDataPtr(timeGenerated),
					"resourceDetails":   llx.DictData(resourceDetails),
					"additionalData":    llx.DictData(additionalData),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlSub)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionCloudDefenderServiceAssessmentSubAssessment) id() (string, error) {
	return a.Id.Data, nil
}
