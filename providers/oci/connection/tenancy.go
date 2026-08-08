// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"errors"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identity"
)

// The tenancy-shaped reads every lister depends on before it can fan out: which
// tenancy this is, which compartments it contains, and which regions it is
// subscribed to.

func (c *OciConnection) TenantID() string {
	return c.tenancyOcid
}

func (c *OciConnection) Tenant(ctx context.Context) (*identity.Tenancy, error) {
	oClient, err := c.IdentityClient()
	if err != nil {
		return nil, err
	}

	resp, err := oClient.GetTenancy(ctx, identity.GetTenancyRequest{
		TenancyId: &c.tenancyOcid,
	})
	if err != nil {
		return nil, err
	}
	return &resp.Tenancy, nil
}

// GetCompartments returns the tenancy root plus every active compartment
// beneath it.
//
// The result is memoized for the lifetime of the connection. A dozen listers
// fan out over the compartment tree, and each one asking for it separately
// meant walking the same paginated ListCompartments a dozen times per scan.
//
// A failure is not cached: an Identity call that fails on a throttle should be
// retried by the next lister rather than turning one bad moment into a scan
// with no compartments at all.
func (c *OciConnection) GetCompartments(ctx context.Context) ([]identity.Compartment, error) {
	c.compartmentLock.Lock()
	defer c.compartmentLock.Unlock()
	if c.compartmentsDone {
		return c.compartmentList, nil
	}

	oClient, err := c.IdentityClient()
	if err != nil {
		return nil, err
	}

	compartments := make([]identity.Compartment, 0)

	req := identity.GetCompartmentRequest{
		CompartmentId: &c.tenancyOcid,
	}

	resp, err := oClient.GetCompartment(ctx, req)
	if err != nil {
		return nil, err
	}
	compartments = append(compartments, resp.Compartment)

	var page *string
	for {
		request := identity.ListCompartmentsRequest{
			CompartmentId:          common.String(c.tenancyOcid),
			CompartmentIdInSubtree: common.Bool(true),
			LifecycleState:         identity.CompartmentLifecycleStateActive,
			Page:                   page,
		}

		response, err := oClient.ListCompartments(ctx, request)
		if err != nil {
			return nil, errors.Join(errors.New("failed to list compartments in tenancy: "+c.tenancyOcid), err)
		}

		for i := range response.Items {
			compartments = append(compartments, response.Items[i])
		}

		page = response.OpcNextPage
		if response.OpcNextPage == nil {
			break
		}
	}

	c.compartmentList = compartments
	c.compartmentsDone = true
	return compartments, nil
}

func (c *OciConnection) GetRegions(ctx context.Context) ([]identity.RegionSubscription, error) {
	oClient, err := c.IdentityClient()
	if err != nil {
		return nil, err
	}

	request := identity.ListRegionSubscriptionsRequest{
		TenancyId: common.String(c.tenancyOcid),
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
