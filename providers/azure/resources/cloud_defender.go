// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/azure/connection"
	"go.mondoo.com/cnquery/v11/types"

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

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForServers() (interface{}, error) {
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
		// According to the CIS implementation of checking if the defender for servers is on, we need to check if the pricing tier is standard
		// https://learn.microsoft.com/en-us/rest/api/defenderforcloud/pricings/list?view=rest-defenderforcloud-2024-01-01&tabs=HTTP#pricingtier
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

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForAppServices() (interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
	if err != nil {
		return nil, err
	}

	appServicePricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "AppServices", &security.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	type defenderForAppServices struct {
		Enabled bool `json:"enabled"`
	}

	resp := defenderForAppServices{}
	if appServicePricing.Properties.PricingTier != nil {
		// Check if the pricing tier is set to 'Standard' which indicates that Defender for App Services is enabled
		resp.Enabled = *appServicePricing.Properties.PricingTier == security.PricingTierStandard
	}

	return convert.JsonToDict(resp)
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForSqlServersOnMachines() (interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
	if err != nil {
		return nil, err
	}

	sqlServerVmPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "SqlServerVirtualMachines", &security.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	type defenderForSqlServersOnMachines struct {
		Enabled bool `json:"enabled"`
	}

	resp := defenderForSqlServersOnMachines{}
	if sqlServerVmPricing.Properties.PricingTier != nil {
		// Check if the pricing tier is set to 'Standard' which indicates that Defender for SQL Servers on Machines is enabled
		resp.Enabled = *sqlServerVmPricing.Properties.PricingTier == security.PricingTierStandard
	}

	return convert.JsonToDict(resp)
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForSqlDatabases() (interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
	if err != nil {
		return nil, err
	}

	sqlDbPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "SqlServers", &security.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	type defenderForSqlDatabases struct {
		Enabled bool `json:"enabled"`
	}

	resp := defenderForSqlDatabases{}
	if sqlDbPricing.Properties.PricingTier != nil {
		// Check if the pricing tier is set to 'Standard' which indicates that Defender for SQL Databases is enabled
		resp.Enabled = *sqlDbPricing.Properties.PricingTier == security.PricingTierStandard
	}

	return convert.JsonToDict(resp)
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForOpenSourceDatabases() (interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
	if err != nil {
		return nil, err
	}

	openSourceDbPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "OpenSourceRelationalDatabases", &security.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	type defenderForOpenSourceDatabases struct {
		Enabled bool `json:"enabled"`
	}

	resp := defenderForOpenSourceDatabases{}
	if openSourceDbPricing.Properties.PricingTier != nil {
		// Check if the pricing tier is set to 'Standard' which indicates that Defender for Open-source Relational Databases is enabled
		resp.Enabled = *openSourceDbPricing.Properties.PricingTier == security.PricingTierStandard
	}

	return convert.JsonToDict(resp)
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForCosmosDb() (interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
	if err != nil {
		return nil, err
	}

	cosmosDbPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "CosmosDbs", &security.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	type defenderForCosmosDb struct {
		Enabled bool `json:"enabled"`
	}

	resp := defenderForCosmosDb{}
	if cosmosDbPricing.Properties.PricingTier != nil {
		// Check if the pricing tier is set to 'Standard' which indicates that Defender for Cosmos DB is enabled
		resp.Enabled = *cosmosDbPricing.Properties.PricingTier == security.PricingTierStandard
	}

	return convert.JsonToDict(resp)
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForStorageAccounts() (interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
	if err != nil {
		return nil, err
	}

	storageAccountsPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "StorageAccounts", &security.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	type defenderForStorageAccounts struct {
		Enabled bool `json:"enabled"`
	}

	resp := defenderForStorageAccounts{}
	if storageAccountsPricing.Properties.PricingTier != nil {
		// Check if the pricing tier is set to 'Standard' which indicates that Defender for Storage Accounts is enabled
		resp.Enabled = *storageAccountsPricing.Properties.PricingTier == security.PricingTierStandard
	}

	return convert.JsonToDict(resp)
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForKeyVaults() (interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
	if err != nil {
		return nil, err
	}

	keyVaultsPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "KeyVaults", &security.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	type defenderForKeyVaults struct {
		Enabled bool `json:"enabled"`
	}

	resp := defenderForKeyVaults{}
	if keyVaultsPricing.Properties.PricingTier != nil {
		// Check if the pricing tier is set to 'Standard' which indicates that Defender for Key Vaults is enabled
		resp.Enabled = *keyVaultsPricing.Properties.PricingTier == security.PricingTierStandard
	}

	return convert.JsonToDict(resp)
}

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForResourceManager() (interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
	if err != nil {
		return nil, err
	}

	resourceManagerPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "Arm", &security.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	type defenderForResourceManager struct {
		Enabled bool `json:"enabled"`
	}

	resp := defenderForResourceManager{}
	if resourceManagerPricing.Properties.PricingTier != nil {
		// Check if the pricing tier is set to 'Standard' which indicates that Defender for Resource Manager is enabled
		resp.Enabled = *resourceManagerPricing.Properties.PricingTier == security.PricingTierStandard
	}

	return convert.JsonToDict(resp)
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

func (a *mqlAzureSubscriptionCloudDefenderService) defenderForContainers() (interface{}, error) {
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
	response, err := clientFactory.NewSettingsClient().Get(ctx, name, nil)
	if err != nil {
		return nil, err
	}
	settings := response.GetSetting()
	if settings == nil {
		return nil, fmt.Errorf("unable to get %s security settings, response was empty", name)
	}

	mqlMCAS, err := CreateResource(a.MqlRuntime,
		"azure.subscription.cloudDefenderService.settings",
		map[string]*llx.RawData{
			"id":   llx.StringDataPtr(settings.ID),
			"name": llx.StringDataPtr(settings.Name),
			"kind": llx.StringDataPtr((*string)(settings.Kind)),
			"type": llx.StringDataPtr(settings.Type),
		},
	)
	if err != nil {
		return nil, err
	}
	return mqlMCAS.(*mqlAzureSubscriptionCloudDefenderServiceSettings), nil
}

func (s *mqlAzureSubscriptionCloudDefenderServiceSettings) id() (string, error) {
	return s.Id.Data, nil
}

func (a *mqlAzureSubscriptionCloudDefenderService) securityContacts() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
	if err != nil {
		return nil, err
	}
	pager := clientFactory.NewContactsClient().NewListPager(nil)
	res := []interface{}{}
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

	sources := map[string]interface{}{}
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

	return args
}
