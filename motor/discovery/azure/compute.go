package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/providers"
	azure_transport "go.mondoo.io/mondoo/motor/providers/azure"
)

type AzureClient struct {
	Subscription string
	Authorizer   autorest.Authorizer
}

func (c *AzureClient) VirtualMachinesClient() compute.VirtualMachinesClient {
	vmClient := compute.NewVirtualMachinesClient(c.Subscription)
	vmClient.Authorizer = c.Authorizer
	return vmClient
}

func (c *AzureClient) InterfacesClient() network.InterfacesClient {
	nicClient := network.NewInterfacesClient(c.Subscription)
	nicClient.Authorizer = c.Authorizer
	return nicClient
}

func (c *AzureClient) PublicIPAddressesClient() network.PublicIPAddressesClient {
	publicIPclient := network.NewPublicIPAddressesClient(c.Subscription)
	publicIPclient.Authorizer = c.Authorizer
	return publicIPclient
}

func NewCompute(subscriptionID string) (*Compute, error) {
	a, err := azure_transport.GetAuthorizer()
	if err != nil {
		return nil, errors.Wrap(err, "could not detect az authentication")
	}

	ac := &AzureClient{
		Subscription: subscriptionID,
		Authorizer:   a,
	}

	return &Compute{
		Subscription: subscriptionID,
		AzureClient:  ac,
	}, nil
}

type Compute struct {
	Subscription string
	AzureClient  *AzureClient
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

	nicClient := c.AzureClient.InterfacesClient()
	nic, err := nicClient.Get(ctx, resource.ResourceGroup, name, "")
	if err != nil {
		return nil, errors.Wrap(err, "could not find network resource")
	}

	if nic.IPConfigurations == nil {
		return nil, errors.New("invalid network information for resource " + resourceID)
	}

	// we got the network interface, not lets extract the public ip
	ipConfigs := *nic.IPConfigurations
	detectedPublicIps := []network.PublicIPAddress{}

	for i := range ipConfigs {
		publicIP := ipConfigs[i].PublicIPAddress

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

			ipClient := c.AzureClient.PublicIPAddressesClient()
			ipResp, err := ipClient.Get(ctx, resource.ResourceGroup, ipAddrName, "")
			if err != nil {
				return nil, errors.Wrap(err, "invalid network information for resource "+publicIPID)
			}
			detectedPublicIps = append(detectedPublicIps, ipResp)
		}
	}
	return detectedPublicIps, nil
}

func (c *Compute) ListInstances(ctx context.Context) ([]*asset.Asset, error) {
	assetList := []*asset.Asset{}

	// fetch all instances in resource group
	vmClient := c.AzureClient.VirtualMachinesClient()
	res, err := vmClient.ListAll(ctx, "", "")
	if err != nil {
		return nil, err
	}
	values := res.Values()
	for i := range values {
		instance := values[i]

		connections := []*providers.TransportConfig{}

		interfaces := *instance.NetworkProfile.NetworkInterfaces
		for ni := range interfaces {
			entry := interfaces[ni]
			log.Debug().Str("entry", *entry.ID).Msg("found network interface")

			publicIpAddrs, err := c.getPublicIp(ctx, *entry.ID)
			if err != nil {
				return nil, err
			}

			for pi := range publicIpAddrs {
				ipResp := publicIpAddrs[pi]

				if ipResp.IPAddress != nil {
					ip := *ipResp.IPAddress
					connections = append(connections, &providers.TransportConfig{
						Backend: providers.ProviderType_SSH,
						Host:    ip,
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

		details, err := vmClient.Get(ctx, res.ResourceGroup, *instance.Name, compute.InstanceViewTypesInstanceView)
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
		asset.Labels["azure.mondoo.com/computername"] = *instance.OsProfile.ComputerName
		asset.Labels["mondoo.com/region"] = *instance.Location
		asset.Labels["mondoo.com/instance"] = *instance.VMID

		assetList = append(assetList, asset)
	}

	return assetList, nil
}

func MondooAzureInstanceID(instanceID string) string {
	return "//platformid.api.mondoo.app/runtime/azure" + instanceID
}
