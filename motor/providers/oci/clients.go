package oci

import (
	"context"

	"errors"
	"github.com/oracle/oci-go-sdk/v65/audit"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
)

func (p *Provider) IdentityClient() (identity.IdentityClient, error) {
	return identity.NewIdentityClientWithConfigurationProvider(p.config)
}

func (p *Provider) TenantID() string {
	return p.tenancyOcid
}

func (p *Provider) Tenant(ctx context.Context) (*identity.Tenancy, error) {
	oClient, err := p.IdentityClient()
	if err != nil {
		return nil, err
	}

	resp, err := oClient.GetTenancy(ctx, identity.GetTenancyRequest{
		TenancyId: &p.tenancyOcid,
	})
	if err != nil {
		return nil, err
	}
	return &resp.Tenancy, nil
}

func (p *Provider) GetCompartments(ctx context.Context) ([]identity.Compartment, error) {
	oClient, err := p.IdentityClient()
	if err != nil {
		return nil, err
	}

	compartments := make([]identity.Compartment, 0)

	req := identity.GetCompartmentRequest{
		CompartmentId: &p.tenancyOcid,
	}

	resp, err := oClient.GetCompartment(ctx, req)
	if err != nil {
		return nil, err
	}
	compartments = append(compartments, resp.Compartment)

	var page *string
	for {
		request := identity.ListCompartmentsRequest{
			CompartmentId:          common.String(p.tenancyOcid),
			CompartmentIdInSubtree: common.Bool(true),
			LifecycleState:         identity.CompartmentLifecycleStateActive,
			Page:                   page,
		}

		response, err := oClient.ListCompartments(ctx, request)
		if err != nil {
			return nil, errors.Join(err, errors.New("failed to list compartments in tenancy: "+p.tenancyOcid))
		}

		for i := range response.Items {
			compartments = append(compartments, response.Items[i])
		}

		page = response.OpcNextPage
		if response.OpcNextPage == nil {
			break
		}
	}

	return compartments, nil
}

func (p *Provider) GetRegions(ctx context.Context) ([]identity.RegionSubscription, error) {
	oClient, err := p.IdentityClient()
	if err != nil {
		return nil, err
	}

	request := identity.ListRegionSubscriptionsRequest{
		TenancyId: common.String(p.tenancyOcid),
	}

	response, err := oClient.ListRegionSubscriptions(ctx, request)
	if err != nil {
		return nil, err
	}

	regions := make([]identity.RegionSubscription, 0)
	for _, region := range response.Items {
		if region.Status != identity.RegionSubscriptionStatusReady {
			continue
		}
		regions = append(regions, region)
	}

	return regions, nil
}

func (p *Provider) ComputeClient(region string) (*core.ComputeClient, error) {
	client, err := core.NewComputeClientWithConfigurationProvider(p.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (p *Provider) IdentityClientWithRegion(region string) (*identity.IdentityClient, error) {
	client, err := identity.NewIdentityClientWithConfigurationProvider(p.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (p *Provider) NetworkClient(region string) (*core.VirtualNetworkClient, error) {
	client, err := core.NewVirtualNetworkClientWithConfigurationProvider(p.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (p *Provider) AuditClient(region string) (*audit.AuditClient, error) {
	client, err := audit.NewAuditClientWithConfigurationProvider(p.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (p *Provider) ObjectStorageClient(region string) (*objectstorage.ObjectStorageClient, error) {
	client, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(p.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}
