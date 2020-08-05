package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	azure_transport "go.mondoo.io/mondoo/motor/transports/azure"
)

// calls az to get azure token
func getAuthorizer() (autorest.Authorizer, error) {
	// create an authorizer from env vars or Azure Managed Service Idenity
	// authorizer, err := auth.NewAuthorizerFromEnvironment()
	return auth.NewAuthorizerFromCLI()
}

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

// call `az account list` -> account.json -> gather default subscription
// az://subscriptions/20192456-09dd-4782-8046-8cdfede4026a/resourceGroups/Demo"
func NewCompute(azureResource string) (*Compute, error) {

	resource, err := azure_transport.ParseResourceID(azureResource)
	if err != nil {
		return nil, errors.Wrap(err, "invalid azure resource e.g. use /subscriptions/1234/resourceGroups/Name")
	}

	a, err := getAuthorizer()
	if err != nil {
		return nil, errors.Wrap(err, "could not detect az authentication")
	}

	ac := &AzureClient{
		Subscription: resource.SubscriptionID,
		Authorizer:   a,
	}

	return &Compute{
		Subscription:  resource.SubscriptionID,
		ResourceGroup: resource.ResourceGroup,
		AzureClient:   ac,
	}, nil
}

type Compute struct {
	Subscription  string
	ResourceGroup string
	AzureClient   *AzureClient
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
	res, err := vmClient.List(ctx, c.ResourceGroup)
	if err != nil {
		return nil, err
	}
	values := res.Values()
	for i := range values {
		instance := values[i]
		// data, _ := json.Marshal(instance)
		// fmt.Println(string(data))

		connections := []*transports.TransportConfig{}

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
					connections = append(connections, &transports.TransportConfig{
						Backend: transports.TransportBackend_CONNECTION_SSH,
						User:    *instance.OsProfile.AdminUsername,
						Host:    ip,
					})
				}
			}
		}

		asset := &asset.Asset{
			// ReferenceIDs: []string{MondooGcpInstanceID(project, zone, instance)},
			Name: *instance.Name,
			Platform: &platform.Platform{
				Kind:    transports.Kind_KIND_VIRTUAL_MACHINE,
				Runtime: transports.RUNTIME_AZ_COMPUTE,
			},
			Connections: connections,
			// State:       mapInstanceState(instance.Status),
			Labels: make(map[string]string),
		}

		// gather details about the instances
		details, err := vmClient.Get(ctx, c.ResourceGroup, *instance.Name, compute.InstanceView)
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
		asset.Labels["azure.mondoo.app/subscription"] = "project"
		asset.Labels["azure.mondoo.app/resourcegroup"] = "project"
		asset.Labels["mondoo.app/region"] = *instance.Location
		asset.Labels["mondoo.app/instance"] = *instance.VMID

		assetList = append(assetList, asset)
	}

	return assetList, nil
}
