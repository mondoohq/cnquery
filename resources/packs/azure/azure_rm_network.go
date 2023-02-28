package azure

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	network "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"go.mondoo.com/cnquery/motor/providers/microsoft/azure"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzureSubscriptionNetworkService) init(args *resources.Args) (*resources.Args, AzureSubscriptionNetworkService, error) {
	if len(*args) > 0 {
		return args, nil, nil
	}

	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	(*args)["subscriptionId"] = at.SubscriptionID()

	return args, nil, nil
}

func (a *mqlAzureSubscriptionNetworkService) id() (string, error) {
	subId, err := a.SubscriptionId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/subscriptions/%s/networkService", subId), nil
}

func (a *mqlAzureSubscriptionNetworkService) GetInterfaces() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := network.NewInterfacesClient(at.SubscriptionID(), token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(&network.InterfacesClientListAllOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, iface := range page.Value {
			if iface != nil {

				mqlAzure, err := azureIfaceToMql(a.MotorRuntime, *iface)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlAzure)
			}
		}
	}
	return res, nil
}

func azureIfaceToMql(runtime *resources.Runtime, iface network.Interface) (resources.ResourceType, error) {
	properties, err := core.JsonToDict(iface.Properties)
	if err != nil {
		return nil, err
	}

	return runtime.CreateResource("azure.subscription.networkService.interface",
		"id", core.ToString(iface.ID),
		"name", core.ToString(iface.Name),
		"location", core.ToString(iface.Location),
		"tags", azureTagsToInterface(iface.Tags),
		"type", core.ToString(iface.Type),
		"etag", core.ToString(iface.Etag),
		"properties", properties,
	)
}

func (a *mqlAzureSubscriptionNetworkService) GetSecurityGroups() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := network.NewSecurityGroupsClient(at.SubscriptionID(), token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(&network.SecurityGroupsClientListAllOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, secGroup := range page.Value {
			if secGroup != nil {
				mqlAzure, err := azureSecGroupToMql(a.MotorRuntime, *secGroup)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlAzure)
			}
		}
	}

	return res, nil
}

// see https://github.com/Azure/azure-sdk-for-go/issues/8224
type AzureSecurityGroupPropertiesFormat network.SecurityGroupPropertiesFormat

func azureSecGroupToMql(runtime *resources.Runtime, secGroup network.SecurityGroup) (resources.ResourceType, error) {
	var properties map[string]interface{}
	ifaces := []interface{}{}
	securityRules := []interface{}{}
	defaultSecurityRules := []interface{}{}
	var err error
	if secGroup.Properties != nil {
		// avoid using the azure sdk SecurityGroupPropertiesFormat MarshalJSON
		var j AzureSecurityGroupPropertiesFormat
		j = AzureSecurityGroupPropertiesFormat(*secGroup.Properties)

		properties, err = core.JsonToDict(j)
		if err != nil {
			return nil, err
		}

		if secGroup.Properties.NetworkInterfaces != nil {
			list := secGroup.Properties.NetworkInterfaces
			for _, iface := range list {
				if iface != nil {
					mqlAzure, err := azureIfaceToMql(runtime, *iface)
					if err != nil {
						return nil, err
					}
					ifaces = append(ifaces, mqlAzure)
				}
			}
		}

		if secGroup.Properties.SecurityRules != nil {
			list := secGroup.Properties.SecurityRules
			for _, secRule := range list {
				if secRule != nil {
					mqlAzure, err := azureSecurityRuleToMql(runtime, *secRule)
					if err != nil {
						return nil, err
					}
					securityRules = append(securityRules, mqlAzure)
				}
			}
		}

		if secGroup.Properties.DefaultSecurityRules != nil {
			list := secGroup.Properties.DefaultSecurityRules
			for _, secRule := range list {
				if secRule != nil {
					mqlAzure, err := azureSecurityRuleToMql(runtime, *secRule)
					if err != nil {
						return nil, err
					}

					defaultSecurityRules = append(defaultSecurityRules, mqlAzure)
				}
			}
		}
	}

	return runtime.CreateResource("azure.subscription.networkService.securityGroup",
		"id", core.ToString(secGroup.ID),
		"name", core.ToString(secGroup.Name),
		"location", core.ToString(secGroup.Location),
		"tags", azureTagsToInterface(secGroup.Tags),
		"type", core.ToString(secGroup.Type),
		"etag", core.ToString(secGroup.Etag),
		"properties", properties,
		"interfaces", ifaces,
		"securityRules", securityRules,
		"defaultSecurityRules", defaultSecurityRules,
	)
}

func azureSecurityRuleToMql(runtime *resources.Runtime, secRule network.SecurityRule) (resources.ResourceType, error) {
	properties, err := core.JsonToDict(secRule.Properties)
	if err != nil {
		return nil, err
	}

	destinationPortRange := []interface{}{}

	if secRule.Properties != nil && secRule.Properties.DestinationPortRange != nil {
		dPortRange := parseAzureSecurityRulePortRange(*secRule.Properties.DestinationPortRange)
		for i := range dPortRange {
			destinationPortRange = append(destinationPortRange, map[string]interface{}{
				"fromPort": dPortRange[i].FromPort,
				"toPort":   dPortRange[i].ToPort,
			})
		}
	}

	return runtime.CreateResource("azure.subscription.networkService.securityrule",
		"id", core.ToString(secRule.ID),
		"name", core.ToString(secRule.Name),
		"etag", core.ToString(secRule.Etag),
		"properties", properties,
		"destinationPortRange", destinationPortRange,
	)
}

type AzureSecurityRulePortRange struct {
	FromPort string
	ToPort   string
}

func parseAzureSecurityRulePortRange(portRange string) []AzureSecurityRulePortRange {
	res := []AzureSecurityRulePortRange{}
	entries := strings.Split(portRange, ",")
	for i := range entries {
		entry := strings.TrimSpace(entries[i])
		if strings.Contains(entry, "-") {
			entryRange := strings.Split(entry, "-")
			res = append(res, AzureSecurityRulePortRange{FromPort: entryRange[0], ToPort: entryRange[1]})
		} else {
			res = append(res, AzureSecurityRulePortRange{FromPort: entry, ToPort: entry})
		}
	}
	return res
}

func (a *mqlAzureSubscriptionNetworkServiceInterface) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureSubscriptionNetworkServiceInterface) GetVm() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *mqlAzureSubscriptionNetworkServiceSecurityGroup) init(args *resources.Args) (*resources.Args, AzureSubscriptionNetworkServiceSecurityGroup, error) {
	if len(*args) > 1 {
		return args, nil, nil
	}

	if len(*args) == 0 {
		if ids := getAssetIdentifier(a.MqlResource().MotorRuntime); ids != nil {
			(*args)["id"] = ids.id
		}
	}

	if (*args)["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure network security group")
	}

	obj, err := a.MotorRuntime.CreateResource("azure.subscription.networkService")
	if err != nil {
		return nil, nil, err
	}
	networkSvc := obj.(*mqlAzureSubscriptionNetworkService)

	rawResources, err := networkSvc.SecurityGroups()
	if err != nil {
		return nil, nil, err
	}

	id := (*args)["id"].(string)
	for i := range rawResources {
		instance := rawResources[i].(AzureSubscriptionNetworkServiceSecurityGroup)
		instanceId, err := instance.Id()
		if err != nil {
			return nil, nil, errors.New("azure network security group does not exist")
		}
		if instanceId == id {
			return args, instance, nil
		}
	}
	return nil, nil, errors.New("azure network security group does not exist")
}

func (a *mqlAzureSubscriptionNetworkServiceSecurityGroup) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureSubscriptionNetworkServiceSecurityrule) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureSubscriptionNetworkService) GetWatchers() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := network.NewWatchersClient(at.SubscriptionID(), token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}
	pager := client.NewListAllPager(&network.WatchersClientListAllOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, watcher := range page.Value {
			properties, err := core.JsonToDict(watcher.Properties)
			if err != nil {
				return nil, err
			}

			mqlAzure, err := a.MotorRuntime.CreateResource("azure.subscription.networkService.watcher",
				"id", core.ToString(watcher.ID),
				"name", core.ToString(watcher.Name),
				"location", core.ToString(watcher.Location),
				"tags", azureTagsToInterface(watcher.Tags),
				"type", core.ToString(watcher.Type),
				"etag", core.ToString(watcher.Etag),
				"properties", properties,
				"provisioningState", core.ToString((*string)(watcher.Properties.ProvisioningState)),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceWatcher) GetFlowLogs() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	id, err := a.id()
	if err != nil {
		return nil, err
	}
	watcherName, err := a.Name()
	if err != nil {
		return nil, err
	}

	resourceID, err := azure.ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	client, err := network.NewFlowLogsClient(at.SubscriptionID(), token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}
	pager := client.NewListPager(resourceID.ResourceGroup, watcherName, &network.FlowLogsClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		type mqlRetentionPolicy struct {
			Enabled       bool `json:"enabled"`
			RetentionDays int  `json:"retentionDays"`
		}
		type mqlFlowLogAnalytics struct {
			Enabled             bool   `json:"allowedApplications"`
			AnalyticsInterval   int    `json:"analyticsInterval"`
			WorkspaceId         string `json:"workspaceResourceId"`
			WorkspaceResourceId string `json:"workspaceId"`
			WorkspaceRegion     string `json:"workspaceRegion"`
		}
		for _, flowLog := range page.Value {
			retentionPolicy := mqlRetentionPolicy{
				Enabled:       core.ToBool(flowLog.Properties.RetentionPolicy.Enabled),
				RetentionDays: core.ToIntFrom32(flowLog.Properties.RetentionPolicy.Days),
			}
			retentionPolicyDict, err := core.JsonToDict(retentionPolicy)
			if err != nil {
				return nil, err
			}
			flowLogAnalytics := mqlFlowLogAnalytics{
				Enabled:             core.ToBool(flowLog.Properties.FlowAnalyticsConfiguration.NetworkWatcherFlowAnalyticsConfiguration.Enabled),
				AnalyticsInterval:   core.ToIntFrom32(flowLog.Properties.FlowAnalyticsConfiguration.NetworkWatcherFlowAnalyticsConfiguration.TrafficAnalyticsInterval),
				WorkspaceRegion:     core.ToString(flowLog.Properties.FlowAnalyticsConfiguration.NetworkWatcherFlowAnalyticsConfiguration.WorkspaceRegion),
				WorkspaceResourceId: core.ToString(flowLog.Properties.FlowAnalyticsConfiguration.NetworkWatcherFlowAnalyticsConfiguration.WorkspaceResourceID),
				WorkspaceId:         core.ToString(flowLog.Properties.FlowAnalyticsConfiguration.NetworkWatcherFlowAnalyticsConfiguration.WorkspaceID),
			}
			flowLogAnalyticsDict, err := core.JsonToDict(flowLogAnalytics)
			if err != nil {
				return nil, err
			}
			mqlFlowLog, err := a.MotorRuntime.CreateResource("azure.subscription.networkService.watcher.flowlog",
				"id", core.ToString(flowLog.ID),
				"name", core.ToString(flowLog.Name),
				"location", core.ToString(flowLog.Location),
				"tags", azureTagsToInterface(flowLog.Tags),
				"type", core.ToString(flowLog.Type),
				"etag", core.ToString(flowLog.Etag),
				"retentionPolicy", retentionPolicyDict,
				"format", core.ToString((*string)(flowLog.Properties.Format.Type)),
				"version", core.ToInt64From32(flowLog.Properties.Format.Version),
				"enabled", core.ToBool(flowLog.Properties.Enabled),
				"storageAccountId", core.ToString(flowLog.Properties.StorageID),
				"targetResourceId", core.ToString(flowLog.Properties.TargetResourceID),
				"targetResourceGuid", core.ToString(flowLog.Properties.TargetResourceGUID),
				"provisioningState", core.ToString((*string)(flowLog.Properties.ProvisioningState)),
				"analytics", flowLogAnalyticsDict,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlFlowLog)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceWatcher) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureSubscriptionNetworkServiceWatcherFlowlog) id() (string, error) {
	return a.Id()
}
