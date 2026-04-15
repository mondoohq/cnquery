// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	iap "cloud.google.com/go/iap/apiv1"
	"cloud.google.com/go/iap/apiv1/iappb"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProject) iap() (*mqlGcpProjectIapService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	res, err := CreateResource(g.MqlRuntime, "gcp.project.iapService", map[string]*llx.RawData{
		"projectId": llx.StringData(g.Id.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectIapService), nil
}

func (g *mqlGcpProjectIapService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf("gcp.project/%s/iapService", g.ProjectId.Data), nil
}

func (g *mqlGcpProjectIapService) brands() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(iap.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := iap.NewIdentityAwareProxyOAuthClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// ListBrands is not paginated — returns all brands at once
	resp, err := client.ListBrands(ctx, &iappb.ListBrandsRequest{
		Parent: fmt.Sprintf("projects/%s", projectId),
	})
	if err != nil {
		return nil, err
	}

	var res []any
	for _, brand := range resp.Brands {
		mqlBrand, err := CreateResource(g.MqlRuntime, "gcp.project.iapService.brand", map[string]*llx.RawData{
			"projectId":        llx.StringData(projectId),
			"name":             llx.StringData(brand.Name),
			"applicationTitle": llx.StringData(brand.ApplicationTitle),
			"supportEmail":     llx.StringData(brand.SupportEmail),
			"orgInternalOnly":  llx.BoolData(brand.OrgInternalOnly),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBrand)
	}

	return res, nil
}

func (g *mqlGcpProjectIapServiceBrand) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	return fmt.Sprintf("gcp.project/%s/iapService.brand/%s", g.ProjectId.Data, g.Name.Data), nil
}

func (g *mqlGcpProjectIapService) tunnelDestGroups() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(iap.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := iap.NewIdentityAwareProxyAdminClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListTunnelDestGroups(ctx, &iappb.ListTunnelDestGroupsRequest{
		Parent: fmt.Sprintf("projects/%s/iap_tunnel/locations/-", projectId),
	})

	var res []any
	for {
		group, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		cidrs := make([]any, len(group.Cidrs))
		for i, c := range group.Cidrs {
			cidrs[i] = c
		}
		fqdns := make([]any, len(group.Fqdns))
		for i, f := range group.Fqdns {
			fqdns[i] = f
		}

		mqlGroup, err := CreateResource(g.MqlRuntime, "gcp.project.iapService.tunnelDestGroup", map[string]*llx.RawData{
			"projectId": llx.StringData(projectId),
			"name":      llx.StringData(group.Name),
			"cidrs":     llx.ArrayData(cidrs, types.String),
			"fqdns":     llx.ArrayData(fqdns, types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlGroup)
	}

	return res, nil
}

func (g *mqlGcpProjectIapServiceTunnelDestGroup) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	return fmt.Sprintf("gcp.project/%s/iapService.tunnelDestGroup/%s", g.ProjectId.Data, g.Name.Data), nil
}
