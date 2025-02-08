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
	vmPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "VirtualMachines", &armsecurity.PricingsClientGetOptions{})
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
		resp.Enabled = *vmPricing.Properties.PricingTier == armsecurity.PricingTierStandard
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

	appServicePricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "AppServices", &armsecurity.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	type defenderForAppServices struct {
		Enabled bool `json:"enabled"`
	}

	resp := defenderForAppServices{}
	if appServicePricing.Properties.PricingTier != nil {
		// Check if the pricing tier is set to 'Standard' which indicates that Defender for App Services is enabled
		resp.Enabled = *appServicePricing.Properties.PricingTier == armsecurity.PricingTierStandard
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

	sqlServerVmPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "SqlServerVirtualMachines", &armsecurity.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	type defenderForSqlServersOnMachines struct {
		Enabled bool `json:"enabled"`
	}

	resp := defenderForSqlServersOnMachines{}
	if sqlServerVmPricing.Properties.PricingTier != nil {
		// Check if the pricing tier is set to 'Standard' which indicates that Defender for SQL Servers on Machines is enabled
		resp.Enabled = *sqlServerVmPricing.Properties.PricingTier == armsecurity.PricingTierStandard
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

	sqlDbPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "SqlServers", &armsecurity.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	type defenderForSqlDatabases struct {
		Enabled bool `json:"enabled"`
	}

	resp := defenderForSqlDatabases{}
	if sqlDbPricing.Properties.PricingTier != nil {
		// Check if the pricing tier is set to 'Standard' which indicates that Defender for SQL Databases is enabled
		resp.Enabled = *sqlDbPricing.Properties.PricingTier == armsecurity.PricingTierStandard
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

	openSourceDbPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "OpenSourceRelationalDatabases", &armsecurity.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	type defenderForOpenSourceDatabases struct {
		Enabled bool `json:"enabled"`
	}

	resp := defenderForOpenSourceDatabases{}
	if openSourceDbPricing.Properties.PricingTier != nil {
		// Check if the pricing tier is set to 'Standard' which indicates that Defender for Open-source Relational Databases is enabled
		resp.Enabled = *openSourceDbPricing.Properties.PricingTier == armsecurity.PricingTierStandard
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

	cosmosDbPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "CosmosDbs", &armsecurity.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	type defenderForCosmosDb struct {
		Enabled bool `json:"enabled"`
	}

	resp := defenderForCosmosDb{}
	if cosmosDbPricing.Properties.PricingTier != nil {
		// Check if the pricing tier is set to 'Standard' which indicates that Defender for Cosmos DB is enabled
		resp.Enabled = *cosmosDbPricing.Properties.PricingTier == armsecurity.PricingTierStandard
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

	storageAccountsPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "StorageAccounts", &armsecurity.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	type defenderForStorageAccounts struct {
		Enabled bool `json:"enabled"`
	}

	resp := defenderForStorageAccounts{}
	if storageAccountsPricing.Properties.PricingTier != nil {
		// Check if the pricing tier is set to 'Standard' which indicates that Defender for Storage Accounts is enabled
		resp.Enabled = *storageAccountsPricing.Properties.PricingTier == armsecurity.PricingTierStandard
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

	keyVaultsPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "KeyVaults", &armsecurity.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	type defenderForKeyVaults struct {
		Enabled bool `json:"enabled"`
	}

	resp := defenderForKeyVaults{}
	if keyVaultsPricing.Properties.PricingTier != nil {
		// Check if the pricing tier is set to 'Standard' which indicates that Defender for Key Vaults is enabled
		resp.Enabled = *keyVaultsPricing.Properties.PricingTier == armsecurity.PricingTierStandard
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

	resourceManagerPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "Arm", &armsecurity.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	type defenderForResourceManager struct {
		Enabled bool `json:"enabled"`
	}

	resp := defenderForResourceManager{}
	if resourceManagerPricing.Properties.PricingTier != nil {
		// Check if the pricing tier is set to 'Standard' which indicates that Defender for Resource Manager is enabled
		resp.Enabled = *resourceManagerPricing.Properties.PricingTier == armsecurity.PricingTierStandard
	}

	return convert.JsonToDict(resp)
}

func (a *mqlAzureSubscriptionCloudDefenderService) monitoringAgentAutoProvision() (bool, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := armsecurity.NewAutoProvisioningSettingsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return false, err
	}

	setting, err := client.Get(ctx, "default", &armsecurity.AutoProvisioningSettingsClientGetOptions{})
	if err != nil {
		return false, err
	}
	autoProvision := *setting.Properties.AutoProvision
	return autoProvision == armsecurity.AutoProvisionOn, nil
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

	containersPricing, err := clientFactory.NewPricingsClient().Get(ctx, fmt.Sprintf("subscriptions/%s", subId), "Containers", &armsecurity.PricingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	enabled := false
	if containersPricing.Properties.PricingTier != nil {
		enabled = *containersPricing.Properties.PricingTier == armsecurity.PricingTierStandard
	}
	extensions := []extension{}
	for _, ext := range containersPricing.Properties.Extensions {
		if ext.IsEnabled == nil || ext.Name == nil {
			continue
		}
		e := false
		if *ext.IsEnabled == armsecurity.IsEnabledTrue {
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

func (a *mqlAzureSubscriptionCloudDefenderService) securityContacts() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := a.SubscriptionId.Data
	armConn, err := getArmSecurityConnection(ctx, conn, subId)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	list, err := getSecurityContacts(ctx, armConn)
	if err != nil {
		// https: //github.com/mondoohq/cnquery/issues/4997
		log.Warn().Err(err).Msg("fail gracefully")
		return res, nil
	}
	for _, contact := range list {
		alertNotifications, err := convert.JsonToDict(contact.Properties.AlertNotifications)
		if err != nil {
			log.Debug().Err(err).Msg("unable to convert armsecurity.Contact.Properties.AlertNotifications to dict")
		}
		notificationsByRole, err := convert.JsonToDict(contact.Properties.NotificationsByRole)
		if err != nil {
			log.Debug().Err(err).Msg("unable to convert armsecurity.Contact.Properties.NotificationsByRole to dict")
		}
		mails := ""
		if contact.Properties.Emails != nil {
			mails = *contact.Properties.Emails
		}
		mailsArr := strings.Split(mails, ";")
		mqlSecurityContact, err := CreateResource(a.MqlRuntime, "azure.subscription.cloudDefenderService.securityContact",
			map[string]*llx.RawData{
				"id":                  llx.StringDataPtr(contact.ID),
				"name":                llx.StringDataPtr(contact.Name),
				"emails":              llx.ArrayData(convert.SliceAnyToInterface(mailsArr), types.String),
				"notificationsByRole": llx.DictData(notificationsByRole),
				"alertNotifications":  llx.DictData(alertNotifications),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSecurityContact)
	}
	return res, nil
}
