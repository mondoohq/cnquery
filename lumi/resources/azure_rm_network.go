package resources

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"go.mondoo.io/mondoo/lumi"
)

func (a *lumiAzurermNetwork) id() (string, error) {
	return "azurerm.network", nil
}

func (a *lumiAzurermNetwork) GetInterfaces() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := network.NewInterfacesClient(at.SubscriptionID())
	client.Authorizer = authorizer

	ifaces, err := client.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range ifaces.Values() {
		iface := ifaces.Values()[i]

		lumiAzure, err := azureIfaceToLumi(a.MotorRuntime, iface)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzure)
	}

	return res, nil
}

func azureIfaceToLumi(runtime *lumi.Runtime, iface network.Interface) (lumi.ResourceType, error) {
	properties, err := jsonToDict(iface.InterfacePropertiesFormat)
	if err != nil {
		return nil, err
	}

	return runtime.CreateResource("azurerm.network.interface",
		"id", toString(iface.ID),
		"name", toString(iface.Name),
		"location", toString(iface.Location),
		"tags", azureTagsToInterface(iface.Tags),
		"type", toString(iface.Type),
		"etag", toString(iface.Etag),
		"properties", properties,
	)
}

func (a *lumiAzurermNetwork) GetSecurityGroups() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := network.NewSecurityGroupsClient(at.SubscriptionID())
	client.Authorizer = authorizer

	secGroups, err := client.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range secGroups.Values() {
		secGroup := secGroups.Values()[i]

		lumiAzure, err := azureSecGroupToLumi(a.MotorRuntime, secGroup)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzure)
	}

	return res, nil
}

// see https://github.com/Azure/azure-sdk-for-go/issues/8224
type AzureSecurityGroupPropertiesFormat network.SecurityGroupPropertiesFormat

func azureSecGroupToLumi(runtime *lumi.Runtime, secGroup network.SecurityGroup) (lumi.ResourceType, error) {
	var properties map[string]interface{}
	ifaces := []interface{}{}
	securityRules := []interface{}{}
	defaultSecurityRules := []interface{}{}
	var err error
	if secGroup.SecurityGroupPropertiesFormat != nil {
		// avoid using the azure sdk SecurityGroupPropertiesFormat MarshalJSON
		var j AzureSecurityGroupPropertiesFormat
		j = AzureSecurityGroupPropertiesFormat(*secGroup.SecurityGroupPropertiesFormat)

		properties, err = jsonToDict(j)
		if err != nil {
			return nil, err
		}

		if secGroup.SecurityGroupPropertiesFormat.NetworkInterfaces != nil {
			list := *secGroup.SecurityGroupPropertiesFormat.NetworkInterfaces
			for i := range list {
				iface := list[i]

				lumiAzure, err := azureIfaceToLumi(runtime, iface)
				if err != nil {
					return nil, err
				}
				ifaces = append(ifaces, lumiAzure)
			}
		}

		if secGroup.SecurityGroupPropertiesFormat.SecurityRules != nil {
			list := *secGroup.SecurityGroupPropertiesFormat.SecurityRules
			for i := range list {
				secRule := list[i]

				lumiAzure, err := azureSecurityRuleToLumi(runtime, secRule)
				if err != nil {
					return nil, err
				}
				securityRules = append(securityRules, lumiAzure)
			}
		}

		if secGroup.SecurityGroupPropertiesFormat.DefaultSecurityRules != nil {
			list := *secGroup.SecurityGroupPropertiesFormat.DefaultSecurityRules
			for i := range list {
				secRule := list[i]

				lumiAzure, err := azureSecurityRuleToLumi(runtime, secRule)
				if err != nil {
					return nil, err
				}
				defaultSecurityRules = append(defaultSecurityRules, lumiAzure)
			}
		}
	}

	return runtime.CreateResource("azurerm.network.securitygroup",
		"id", toString(secGroup.ID),
		"name", toString(secGroup.Name),
		"location", toString(secGroup.Location),
		"tags", azureTagsToInterface(secGroup.Tags),
		"type", toString(secGroup.Type),
		"etag", toString(secGroup.Etag),
		"properties", properties,
		"interfaces", ifaces,
		"securityRules", securityRules,
		"defaultSecurityRules", defaultSecurityRules,
	)
}

func azureSecurityRuleToLumi(runtime *lumi.Runtime, secRule network.SecurityRule) (lumi.ResourceType, error) {
	properties, err := jsonToDict(secRule.SecurityRulePropertiesFormat)
	if err != nil {
		return nil, err
	}

	destinationPortRange := []interface{}{}

	if secRule.SecurityRulePropertiesFormat != nil && secRule.SecurityRulePropertiesFormat.DestinationPortRange != nil {
		dPortRange := parseAzureSecurityRulePortRange(*secRule.SecurityRulePropertiesFormat.DestinationPortRange)
		for i := range dPortRange {
			destinationPortRange = append(destinationPortRange, map[string]interface{}{
				"fromPort": dPortRange[i].FromPort,
				"toPort":   dPortRange[i].ToPort,
			})
		}
	}

	return runtime.CreateResource("azurerm.network.securityrule",
		"id", toString(secRule.ID),
		"name", toString(secRule.Name),
		"etag", toString(secRule.Etag),
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

func (a *lumiAzurermNetworkInterface) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermNetworkInterface) GetVm() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *lumiAzurermNetworkSecuritygroup) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermNetworkSecurityrule) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermNetwork) GetWatchers() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := network.NewWatchersClient(at.SubscriptionID())
	client.Authorizer = authorizer

	watchers, err := client.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	if watchers.Value == nil {
		return res, nil
	}

	list := *watchers.Value
	for i := range list {
		watcher := list[i]

		properties, err := jsonToDict(watcher.WatcherPropertiesFormat)
		if err != nil {
			return nil, err
		}

		lumiAzure, err := a.MotorRuntime.CreateResource("azurerm.network.watcher",
			"id", toString(watcher.ID),
			"name", toString(watcher.Name),
			"location", toString(watcher.Location),
			"tags", azureTagsToInterface(watcher.Tags),
			"type", toString(watcher.Type),
			"etag", toString(watcher.Etag),
			"properties", properties,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzure)
	}

	return res, nil
}

func (a *lumiAzurermNetworkWatcher) id() (string, error) {
	return a.Id()
}
