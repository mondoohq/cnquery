// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
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

	arcClusterPolicyExtensionDefinitionId        string = "/providers/Microsoft.Authorization/policyDefinitions/0adc5395-9169-4b9b-8687-af838d69410a"
	kubernetesClusterPolicyExtensionDefinitionId string = "/providers/Microsoft.Authorization/policyDefinitions/0adc5395-9169-4b9b-8687-af838d69410a"
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

// getSimpleDictPricing fetches pricing data for a Defender component and returns it as a dict
// with a single "enabled" boolean field. Used by the deprecated defenderForX() dict methods.
func (a *mqlAzureSubscriptionCloudDefenderService) getSimpleDictPricing(azurePricingName string) (any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
	if err != nil {
		return nil, err
	}

	pricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), azurePricingName, &security.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	type simplePricing struct {
		Enabled bool `json:"enabled"`
	}

	resp := simplePricing{}
	if pricing.Properties != nil && pricing.Properties.PricingTier != nil {
		resp.Enabled = *pricing.Properties.PricingTier == security.PricingTierStandard
	}

	return convert.JsonToDict(resp)
}

// getSimpleDefenderPricing fetches pricing data for a Defender component and creates a typed resource.
func (a *mqlAzureSubscriptionCloudDefenderService) getSimpleDefenderPricing(azurePricingName, mqlResourceName string) (plugin.Resource, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
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

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForServers() (any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
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
	list, err := getPolicyAssignments(ctx, armConn)
	if err != nil {
		return nil, err
	}
	serverVASetings, err := getServerVulnAssessmentSettings(ctx, armConn)
	if err != nil {
		return nil, err
	}

	type defenderForServers struct {
		Enabled                         bool   `json:"enabled"`
		VulnerabilityManagementToolName string `json:"vulnerabilityManagementToolName"`
	}

	resp := defenderForServers{}
	if vmPricing.Properties.PricingTier != nil {
		resp.Enabled = *vmPricing.Properties.PricingTier == security.PricingTierStandard
	}

	for _, it := range list.PolicyAssignments {
		if it.Properties.PolicyDefinitionID == vaQualysPolicyDefinitionId {
			resp.Enabled = true
			resp.VulnerabilityManagementToolName = "Microsoft Defender for Cloud integrated Qualys scanner"
		}
	}
	for _, sett := range serverVASetings.Settings {
		if sett.Properties.SelectedProvider == "MdeTvm" && sett.Name == "AzureServersSetting" {
			resp.Enabled = true
			resp.VulnerabilityManagementToolName = "Microsoft Defender vulnerability management"
		}
	}
	return convert.JsonToDict(resp)
}

func (a *mqlAzureSubscriptionCloudDefenderService) forServers() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForServers, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
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
	list, err := getPolicyAssignments(ctx, armConn)
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
		if it.Properties.PolicyDefinitionID == vaQualysPolicyDefinitionId {
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

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForAppServices() (any, error) {
	return a.getSimpleDictPricing("AppServices")
}

func (a *mqlAzureSubscriptionCloudDefenderService) forAppServices() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForAppServices, error) {
	resource, err := a.getSimpleDefenderPricing("AppServices", ResourceAzureSubscriptionCloudDefenderServiceDefenderForAppServices)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForAppServices), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForSqlServersOnMachines() (any, error) {
	return a.getSimpleDictPricing("SqlServerVirtualMachines")
}

func (a *mqlAzureSubscriptionCloudDefenderService) forSqlServersOnMachines() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForSqlServersOnMachines, error) {
	resource, err := a.getSimpleDefenderPricing("SqlServerVirtualMachines", ResourceAzureSubscriptionCloudDefenderServiceDefenderForSqlServersOnMachines)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForSqlServersOnMachines), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForSqlDatabases() (any, error) {
	return a.getSimpleDictPricing("SqlServers")
}

func (a *mqlAzureSubscriptionCloudDefenderService) forSqlDatabases() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForSqlDatabases, error) {
	resource, err := a.getSimpleDefenderPricing("SqlServers", ResourceAzureSubscriptionCloudDefenderServiceDefenderForSqlDatabases)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForSqlDatabases), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForOpenSourceDatabases() (any, error) {
	return a.getSimpleDictPricing("OpenSourceRelationalDatabases")
}

func (a *mqlAzureSubscriptionCloudDefenderService) forOpenSourceDatabases() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForOpenSourceDatabases, error) {
	resource, err := a.getSimpleDefenderPricing("OpenSourceRelationalDatabases", ResourceAzureSubscriptionCloudDefenderServiceDefenderForOpenSourceDatabases)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForOpenSourceDatabases), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForCosmosDb() (any, error) {
	return a.getSimpleDictPricing("CosmosDbs")
}

func (a *mqlAzureSubscriptionCloudDefenderService) forCosmosDb() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForCosmosDb, error) {
	resource, err := a.getSimpleDefenderPricing("CosmosDbs", ResourceAzureSubscriptionCloudDefenderServiceDefenderForCosmosDb)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForCosmosDb), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForStorageAccounts() (any, error) {
	return a.getSimpleDictPricing("StorageAccounts")
}

func (a *mqlAzureSubscriptionCloudDefenderService) forStorageAccounts() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForStorageAccounts, error) {
	resource, err := a.getSimpleDefenderPricing("StorageAccounts", ResourceAzureSubscriptionCloudDefenderServiceDefenderForStorageAccounts)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForStorageAccounts), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForKeyVaults() (any, error) {
	return a.getSimpleDictPricing("KeyVaults")
}

func (a *mqlAzureSubscriptionCloudDefenderService) forKeyVaults() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForKeyVaults, error) {
	resource, err := a.getSimpleDefenderPricing("KeyVaults", ResourceAzureSubscriptionCloudDefenderServiceDefenderForKeyVaults)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForKeyVaults), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForResourceManager() (any, error) {
	return a.getSimpleDictPricing("Arm")
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

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
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

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
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
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := a.SubscriptionId.Data

	armConn, err := getArmSecurityConnection(ctx, conn, subId)
	if err != nil {
		return nil, err
	}

	pas, err := getPolicyAssignments(ctx, armConn)
	if err != nil {
		return nil, err
	}

	type extension struct {
		Name      string `json:"name"`
		IsEnabled bool   `json:"isEnabled"`
	}

	type defenderForContainers struct {
		DefenderDaemonSet        bool        `json:"defenderDaemonSet"`
		AzurePolicyForKubernetes bool        `json:"azurePolicyForKubernetes"`
		Enabled                  bool        `json:"enabled"`
		Extensions               []extension `json:"extensions"`
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
	clientFactory, err := armsecurity.NewClientFactory(subId, armConn.token, nil)
	if err != nil {
		return nil, err
	}

	containersPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "Containers", &security.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	enabled := false
	if containersPricing.Properties.PricingTier != nil {
		enabled = *containersPricing.Properties.PricingTier == security.PricingTierStandard
	}
	extensions := []extension{}
	for _, ext := range containersPricing.Properties.Extensions {
		if ext.IsEnabled == nil || ext.Name == nil {
			continue
		}
		e := false
		if *ext.IsEnabled == security.IsEnabledTrue {
			e = true
		}
		extensions = append(extensions, extension{Name: *ext.Name, IsEnabled: e})
	}

	def := defenderForContainers{
		DefenderDaemonSet:        arcDefender && kubernetesDefender,
		AzurePolicyForKubernetes: arcPolicyExt && kubernetesPolicyExt,
		Enabled:                  enabled,
		Extensions:               extensions,
	}

	return convert.JsonToDict(def)
}

func (a *mqlAzureSubscriptionCloudDefenderService) forContainers() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForContainers, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := a.SubscriptionId.Data

	armConn, err := getArmSecurityConnection(ctx, conn, subId)
	if err != nil {
		return nil, err
	}

	pas, err := getPolicyAssignments(ctx, armConn)
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
	clientFactory, err := armsecurity.NewClientFactory(subId, armConn.token, nil)
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
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForContainers), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) settingsMCAS() (*mqlAzureSubscriptionCloudDefenderServiceSettings, error) {
	return a.getSecuritySettingsFor(security.SettingNameAutoGeneratedMCAS)
}

func (a *mqlAzureSubscriptionCloudDefenderService) settingsWDATP() (*mqlAzureSubscriptionCloudDefenderServiceSettings, error) {
	return a.getSecuritySettingsFor(security.SettingNameAutoGeneratedWDATP)
}

func (a *mqlAzureSubscriptionCloudDefenderService) settingsSentinel() (*mqlAzureSubscriptionCloudDefenderServiceSettings, error) {
	return a.getSecuritySettingsFor(security.SettingNameAutoGeneratedSentinel)
}

func (a *mqlAzureSubscriptionCloudDefenderService) getSecuritySettingsFor(name security.SettingNameAutoGenerated) (*mqlAzureSubscriptionCloudDefenderServiceSettings, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
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
	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
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

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
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

// extensions lazily fetches extension data for Defender for Containers. This re-fetches the Containers
// pricing data (already retrieved by defenderForContainers) so that extensions are only loaded when accessed.
func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderForContainers) extensions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
	if err != nil {
		return nil, err
	}

	containersPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "Containers", &security.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	if containersPricing.Properties == nil {
		return []any{}, nil
	}

	return buildExtensionResources(a.MqlRuntime, containersPricing.Properties.Extensions,
		ResourceAzureSubscriptionCloudDefenderServiceDefenderForContainersExtension,
		ResourceAzureSubscriptionCloudDefenderServiceDefenderForContainers+"/"+subId)
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

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
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

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
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

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
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

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
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

// assessmentSeverityByName returns a map of assessment name to its severity.
// Assessment list responses don't carry severity, so it is joined in from the
// metadata catalog: built-in definitions come from the global list, custom ones
// from the subscription-scoped list.
func assessmentSeverityByName(ctx context.Context, clientFactory *armsecurity.ClientFactory) map[string]string {
	severities := map[string]string{}
	metadataClient := clientFactory.NewAssessmentsMetadataClient()

	collect := func(items []*armsecurity.AssessmentMetadataResponse) {
		for _, item := range items {
			if item.Name == nil || item.Properties == nil || item.Properties.Severity == nil {
				continue
			}
			severities[*item.Name] = string(*item.Properties.Severity)
		}
	}

	builtinPager := metadataClient.NewListPager(nil)
	for builtinPager.More() {
		page, err := builtinPager.NextPage(ctx)
		if err != nil {
			log.Debug().Err(err).Msg("could not list built-in assessment metadata for severities")
			break
		}
		collect(page.Value)
	}

	subPager := metadataClient.NewListBySubscriptionPager(nil)
	for subPager.More() {
		page, err := subPager.NextPage(ctx)
		if err != nil {
			log.Debug().Err(err).Msg("could not list subscription assessment metadata for severities")
			break
		}
		collect(page.Value)
	}

	return severities
}

func (a *mqlAzureSubscriptionCloudDefenderService) assessments() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
	if err != nil {
		return nil, err
	}

	severities := assessmentSeverityByName(ctx, clientFactory)

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
			additionalData := map[string]any{}

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
				}
				for k, v := range props.AdditionalData {
					if v != nil {
						additionalData[k] = *v
					}
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
			severity := ""
			if item.Name != nil {
				severity = severities[*item.Name]
			}

			mqlResource, err := CreateResource(a.MqlRuntime,
				"azure.subscription.cloudDefenderService.assessment",
				map[string]*llx.RawData{
					"__id":              llx.StringDataPtr(item.ID),
					"id":                llx.StringDataPtr(item.ID),
					"name":              llx.StringDataPtr(item.Name),
					"displayName":       llx.StringData(displayName),
					"status":            llx.StringData(status),
					"statusCause":       llx.StringData(statusCause),
					"statusDescription": llx.StringData(statusDescription),
					"severity":          llx.StringData(severity),
					"resourceId":        llx.StringData(resourceId),
					"additionalData":    llx.DictData(additionalData),
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

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
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
