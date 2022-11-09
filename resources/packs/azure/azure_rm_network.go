package azure

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	network "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzurermNetwork) id() (string, error) {
	return "azurerm.network", nil
}

func (a *mqlAzurermNetwork) GetInterfaces() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
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

	return runtime.CreateResource("azurerm.network.interface",
		"id", core.ToString(iface.ID),
		"name", core.ToString(iface.Name),
		"location", core.ToString(iface.Location),
		"tags", azureTagsToInterface(iface.Tags),
		"type", core.ToString(iface.Type),
		"etag", core.ToString(iface.Etag),
		"properties", properties,
	)
}

func (a *mqlAzurermNetwork) GetSecurityGroups() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
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

	return runtime.CreateResource("azurerm.network.securitygroup",
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

	return runtime.CreateResource("azurerm.network.securityrule",
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

func (a *mqlAzurermNetworkInterface) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermNetworkInterface) GetVm() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *mqlAzurermNetworkSecuritygroup) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermNetworkSecurityrule) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermNetwork) GetWatchers() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
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

			mqlAzure, err := a.MotorRuntime.CreateResource("azurerm.network.watcher",
				"id", core.ToString(watcher.ID),
				"name", core.ToString(watcher.Name),
				"location", core.ToString(watcher.Location),
				"tags", azureTagsToInterface(watcher.Tags),
				"type", core.ToString(watcher.Type),
				"etag", core.ToString(watcher.Etag),
				"properties", properties,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzurermNetworkWatcher) id() (string, error) {
	return a.Id()
}
