// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"

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

	var enablementTime *time.Time
	if props.EnablementTime != nil {
		enablementTime = props.EnablementTime
	}
	args["enablementTime"] = llx.TimeDataPtr(enablementTime)

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

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForServers() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForServers, error) {
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

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForAppServices() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForAppServices, error) {
	resource, err := a.getSimpleDefenderPricing("AppServices", ResourceAzureSubscriptionCloudDefenderServiceDefenderForAppServices)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForAppServices), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForSqlServersOnMachines() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForSqlServersOnMachines, error) {
	resource, err := a.getSimpleDefenderPricing("SqlServerVirtualMachines", ResourceAzureSubscriptionCloudDefenderServiceDefenderForSqlServersOnMachines)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForSqlServersOnMachines), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForSqlDatabases() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForSqlDatabases, error) {
	resource, err := a.getSimpleDefenderPricing("SqlServers", ResourceAzureSubscriptionCloudDefenderServiceDefenderForSqlDatabases)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForSqlDatabases), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForOpenSourceDatabases() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForOpenSourceDatabases, error) {
	resource, err := a.getSimpleDefenderPricing("OpenSourceRelationalDatabases", ResourceAzureSubscriptionCloudDefenderServiceDefenderForOpenSourceDatabases)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForOpenSourceDatabases), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForCosmosDb() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForCosmosDb, error) {
	resource, err := a.getSimpleDefenderPricing("CosmosDbs", ResourceAzureSubscriptionCloudDefenderServiceDefenderForCosmosDb)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForCosmosDb), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForStorageAccounts() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForStorageAccounts, error) {
	resource, err := a.getSimpleDefenderPricing("StorageAccounts", ResourceAzureSubscriptionCloudDefenderServiceDefenderForStorageAccounts)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForStorageAccounts), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForKeyVaults() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForKeyVaults, error) {
	resource, err := a.getSimpleDefenderPricing("KeyVaults", ResourceAzureSubscriptionCloudDefenderServiceDefenderForKeyVaults)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionCloudDefenderServiceDefenderForKeyVaults), nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForResourceManager() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForResourceManager, error) {
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
	autoProvision := *setting.Properties.AutoProvision
	return autoProvision == security.AutoProvisionOn, nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForContainers() (*mqlAzureSubscriptionCloudDefenderServiceDefenderForContainers, error) {
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

		if sourceType == armsecurity.SourceTypeAlert {
			// back-fill alert notifications for backwards compatibility
			//
			// This field has two properties, `minimalSeverity` and `state`, that the new type is only missing state.
			//
			// https://github.com/Azure/azure-sdk-for-go/blob/sdk/resourcemanager/security/armsecurity/v0.13.0/sdk/resourcemanager/security/armsecurity/models.go#L2404
			//
			state := "On"
			sourceDict["state"] = state
			args["alertNotifications"] = llx.DictData(sourceDict)
		}
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
			"__id":                   llx.StringData(parentIdPrefix + "/extensions/" + name),
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

	return buildExtensionResources(a.MqlRuntime, containersPricing.Properties.Extensions,
		ResourceAzureSubscriptionCloudDefenderServiceDefenderForContainersExtension,
		ResourceAzureSubscriptionCloudDefenderServiceDefenderForContainers+"/"+subId)
}

func (a *mqlAzureSubscriptionCloudDefenderServiceDefenderForContainersExtension) id() (string, error) {
	return a.__id, nil
}
