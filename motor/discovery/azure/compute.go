package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	compute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	network "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	azure_transport "go.mondoo.com/cnquery/motor/providers/azure"
)

type Compute struct {
	Subscription string
	AzureClient  *AzureClient
}

func NewCompute(client *AzureClient, subscriptionID string) *Compute {
	return &Compute{
		Subscription: subscriptionID,
		AzureClient:  client,
	}
}

func (c *Compute) VirtualMachinesClient() (*compute.VirtualMachinesClient, error) {
	return compute.NewVirtualMachinesClient(c.Subscription, c.AzureClient.Token, &arm.ClientOptions{})
}

func (c *Compute) InterfacesClient() (*network.InterfacesClient, error) {
	return network.NewInterfacesClient(c.Subscription, c.AzureClient.Token, &arm.ClientOptions{})
}

func (c *Compute) PublicIPAddressesClient() (*network.PublicIPAddressesClient, error) {
	return network.NewPublicIPAddressesClient(c.Subscription, c.AzureClient.Token, &arm.ClientOptions{})
}

// getPublicIp reads the public ip by using its resource identifier
// "/subscriptions/20192456-09dd-4782-8046-8cdfede4026a/resourceGroups/Demo/providers/Microsoft.Network/networkInterfaces/test35"
func (c *Compute) getPublicIp(ctx context.Context, resourceID string) ([]network.PublicIPAddress, error) {
	resource, err := azure_transport.ParseResourceID(resourceID)
	if err != nil {
		return nil, errors.Wrap(err, "invalid network resource")
	}

	name, err := resource.Component("networkInterfaces")
	if err != nil {
		return nil, errors.Wrap(err, "invalid network resource")
	}

	nicClient, err := c.InterfacesClient()
	if err != nil {
		return nil, errors.Wrap(err, "cannot initialize interfaces client")
	}
	nic, err := nicClient.Get(ctx, resource.ResourceGroup, name, &network.InterfacesClientGetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "could not find network resource")
	}
	if nic.Interface.Properties.IPConfigurations == nil {
		return nil, errors.New("invalid network information for resource " + resourceID)
	}

	// we got the network interface, now lets extract the public ip
	ipConfigs := nic.Interface.Properties.IPConfigurations
	detectedPublicIps := []network.PublicIPAddress{}

	for i := range ipConfigs {
		publicIP := ipConfigs[i].Properties.PublicIPAddress

		if publicIP != nil && publicIP.ID != nil {

			publicIPID := *publicIP.ID

			publicIpResource, err := azure_transport.ParseResourceID(publicIPID)
			if err != nil {
				return nil, errors.New("invalid network information for resource " + publicIPID)
			}

			ipAddrName, err := publicIpResource.Component("publicIPAddresses")
			if err != nil {
				return nil, errors.New("invalid network information for resource " + publicIPID)
			}

			ipClient, err := c.PublicIPAddressesClient()
			if err != nil {
				return nil, errors.Wrap(err, "cannot initialize ip address client")
			}
			ipResp, err := ipClient.Get(ctx, resource.ResourceGroup, ipAddrName, &network.PublicIPAddressesClientGetOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "invalid network information for resource "+publicIPID)
			}
			detectedPublicIps = append(detectedPublicIps, ipResp.PublicIPAddress)
		}
	}
	return detectedPublicIps, nil
}

func (c *Compute) ListInstances(ctx context.Context) ([]*asset.Asset, error) {
	assetList := []*asset.Asset{}

	// fetch all instances in resource group
	vmClient, err := c.VirtualMachinesClient()
	if err != nil {
		return nil, errors.Wrap(err, "cannot initialize virtual machines client")
	}
	res := vmClient.NewListAllPager(&compute.VirtualMachinesClientListAllOptions{})
	if err != nil {
		return nil, err
	}
	for res.More() {
		page, err := res.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, instance := range page.Value {
			connections := []*providers.Config{}

			interfaces := instance.Properties.NetworkProfile.NetworkInterfaces
			for ni := range interfaces {
				entry := interfaces[ni]
				log.Debug().Str("entry", *entry.ID).Msg("found network interface")

				publicIpAddrs, err := c.getPublicIp(ctx, *entry.ID)
				if err != nil {
					return nil, err
				}

				for _, ip := range publicIpAddrs {
					if ip.Properties.IPAddress != nil {
						connections = append(connections, &providers.Config{
							Backend: providers.ProviderType_SSH,
							Host:    *ip.Properties.IPAddress,
							// we do not add credentials here since those may not match the expected state
							// *instance.OsProfile.AdminUsername
						})
					}
				}
			}

			// TODO: derive platform information from azure instance
			asset := &asset.Asset{
				PlatformIds: []string{MondooAzureInstanceID(*instance.ID)},
				Name:        *instance.Name,
				Platform: &platform.Platform{
					Kind:    providers.Kind_KIND_VIRTUAL_MACHINE,
					Runtime: providers.RUNTIME_AZ_COMPUTE,
				},
				Connections: connections,
				// NOTE: this is really not working in azure, see https://github.com/Azure/azure-sdk-for-python/issues/573
				// we''ll update this later when each individual machine is scanned
				State:  asset.State_STATE_UNKNOWN,
				Labels: make(map[string]string),
			}

			// gather details about the instances
			res, err := azure_transport.ParseResourceID(*instance.ID)
			if err != nil {
				return nil, err
			}

			instanceView := compute.InstanceViewTypesInstanceView
			details, err := vmClient.Get(ctx, res.ResourceGroup, *instance.Name, &compute.VirtualMachinesClientGetOptions{Expand: &instanceView})
			if err != nil {
				return nil, err
			}

			for key := range details.Tags {
				value := ""
				if instance.Tags[key] != nil {
					value = *instance.Tags[key]
				}
				asset.Labels[key] = value
			}

			// fetch azure specific metadata
			asset.Labels["azure.mondoo.com/subscription"] = res.SubscriptionID
			asset.Labels["azure.mondoo.com/resourcegroup"] = res.ResourceGroup
			asset.Labels["azure.mondoo.com/computername"] = *instance.Properties.OSProfile.ComputerName
			asset.Labels["mondoo.com/region"] = *instance.Location
			asset.Labels["mondoo.com/instance"] = *instance.Properties.VMID

			assetList = append(assetList, asset)
		}
	}

	return assetList, nil
}

func MondooAzureInstanceID(instanceID string) string {
	return "//platformid.api.mondoo.app/runtime/azure" + instanceID
}
