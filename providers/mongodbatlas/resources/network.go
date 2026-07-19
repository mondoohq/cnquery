// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"net/http"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlMongodbatlas) ipAccessList() ([]any, error) {
	pid, err := projectID(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	client := atlasClient(r.MqlRuntime)
	ctx := context.Background()

	out := []any{}
	for page := 1; ; page++ {
		resp, _, err := client.ProjectIPAccessListApi.ListProjectIpAccessLists(ctx, pid).ItemsPerPage(pageSize).PageNum(page).Execute()
		if err != nil {
			return nil, err
		}
		results := resp.GetResults()
		for i := range results {
			e := results[i]
			value := e.GetCidrBlock()
			if value == "" {
				value = e.GetIpAddress()
			}
			if value == "" {
				value = e.GetAwsSecurityGroup()
			}
			res, err := CreateResource(r.MqlRuntime, "mongodbatlas.networkAccessEntry", map[string]*llx.RawData{
				"__id":             llx.StringData("mongodbatlas.networkAccessEntry/" + pid + "/" + value),
				"cidrBlock":        llx.StringData(e.GetCidrBlock()),
				"ipAddress":        llx.StringData(e.GetIpAddress()),
				"awsSecurityGroup": llx.StringData(e.GetAwsSecurityGroup()),
				"comment":          llx.StringData(e.GetComment()),
				"deleteAfterDate":  llx.TimeDataPtr(timePtr(e.GetDeleteAfterDate())),
			})
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
		if len(results) < pageSize {
			break
		}
	}
	return out, nil
}

func (r *mqlMongodbatlas) privateEndpoints() ([]any, error) {
	pid, err := projectID(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	client := atlasClient(r.MqlRuntime)
	ctx := context.Background()

	out := []any{}
	for _, provider := range []string{"AWS", "AZURE", "GCP"} {
		services, httpResp, err := client.PrivateEndpointServicesApi.ListPrivateEndpointServices(ctx, pid, provider).Execute()
		if err != nil {
			// A provider without any configured private endpoint service returns
			// 404; skip it, but surface auth, throttling, and other real errors.
			if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
				continue
			}
			return nil, err
		}
		for i := range services {
			svc := services[i]
			res, err := CreateResource(r.MqlRuntime, "mongodbatlas.privateEndpointService", map[string]*llx.RawData{
				"__id":               llx.StringData("mongodbatlas.privateEndpointService/" + pid + "/" + svc.GetId()),
				"id":                 llx.StringData(svc.GetId()),
				"cloudProvider":      llx.StringData(svc.GetCloudProvider()),
				"regionName":         llx.StringData(svc.GetRegionName()),
				"status":             llx.StringData(svc.GetStatus()),
				"errorMessage":       llx.StringData(svc.GetErrorMessage()),
				"interfaceEndpoints": llx.ArrayData(strSlice(svc.GetInterfaceEndpoints()), types.String),
			})
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
	}
	return out, nil
}

func (r *mqlMongodbatlas) networkPeerings() ([]any, error) {
	pid, err := projectID(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	client := atlasClient(r.MqlRuntime)
	ctx := context.Background()

	out := []any{}
	for page := 1; ; page++ {
		resp, _, err := client.NetworkPeeringApi.ListPeeringConnections(ctx, pid).ItemsPerPage(pageSize).PageNum(page).Execute()
		if err != nil {
			return nil, err
		}
		results := resp.GetResults()
		for i := range results {
			p := results[i]
			// AWS reports status in statusName; Azure and GCP use status.
			status := p.GetStatusName()
			if status == "" {
				status = p.GetStatus()
			}
			res, err := CreateResource(r.MqlRuntime, "mongodbatlas.networkPeering", map[string]*llx.RawData{
				"__id":                llx.StringData("mongodbatlas.networkPeering/" + pid + "/" + p.GetId()),
				"id":                  llx.StringData(p.GetId()),
				"providerName":        llx.StringData(p.GetProviderName()),
				"containerId":         llx.StringData(p.GetContainerId()),
				"status":              llx.StringData(status),
				"awsAccountId":        llx.StringData(p.GetAwsAccountId()),
				"vpcId":               llx.StringData(p.GetVpcId()),
				"vnetName":            llx.StringData(p.GetVnetName()),
				"networkName":         llx.StringData(p.GetNetworkName()),
				"routeTableCidrBlock": llx.StringData(p.GetRouteTableCidrBlock()),
			})
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
		if len(results) < pageSize {
			break
		}
	}
	return out, nil
}

func (r *mqlMongodbatlas) cloudProviderAccessRoles() ([]any, error) {
	pid, err := projectID(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	client := atlasClient(r.MqlRuntime)
	ctx := context.Background()

	roles, _, err := client.CloudProviderAccessApi.ListCloudProviderAccessRoles(ctx, pid).Execute()
	if err != nil {
		return nil, err
	}

	out := []any{}
	for _, role := range roles.GetAwsIamRoles() {
		res, err := CreateResource(r.MqlRuntime, "mongodbatlas.cloudProviderAccessRole", map[string]*llx.RawData{
			"__id":               llx.StringData("mongodbatlas.cloudProviderAccessRole/" + pid + "/aws/" + role.GetRoleId()),
			"id":                 llx.StringData(role.GetRoleId()),
			"providerName":       llx.StringData(role.GetProviderName()),
			"iamAssumedRoleArn":  llx.StringData(role.GetIamAssumedRoleArn()),
			"atlasAWSAccountArn": llx.StringData(role.GetAtlasAWSAccountArn()),
			"azureAtlasAppId":    llx.StringData(""),
			"azureTenantId":      llx.StringData(""),
			"gcpServiceAccount":  llx.StringData(""),
			"authorizedDate":     llx.TimeDataPtr(timePtr(role.GetAuthorizedDate())),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	for _, sp := range roles.GetAzureServicePrincipals() {
		res, err := CreateResource(r.MqlRuntime, "mongodbatlas.cloudProviderAccessRole", map[string]*llx.RawData{
			"__id":               llx.StringData("mongodbatlas.cloudProviderAccessRole/" + pid + "/azure/" + sp.GetId()),
			"id":                 llx.StringData(sp.GetId()),
			"providerName":       llx.StringData(sp.GetProviderName()),
			"iamAssumedRoleArn":  llx.StringData(""),
			"atlasAWSAccountArn": llx.StringData(""),
			"azureAtlasAppId":    llx.StringData(sp.GetAtlasAzureAppId()),
			"azureTenantId":      llx.StringData(sp.GetTenantId()),
			"gcpServiceAccount":  llx.StringData(""),
			"authorizedDate":     llx.TimeDataPtr(timePtr(sp.GetCreatedDate())),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	for _, sa := range roles.GetGcpServiceAccounts() {
		res, err := CreateResource(r.MqlRuntime, "mongodbatlas.cloudProviderAccessRole", map[string]*llx.RawData{
			"__id":               llx.StringData("mongodbatlas.cloudProviderAccessRole/" + pid + "/gcp/" + sa.GetRoleId()),
			"id":                 llx.StringData(sa.GetRoleId()),
			"providerName":       llx.StringData(sa.GetProviderName()),
			"iamAssumedRoleArn":  llx.StringData(""),
			"atlasAWSAccountArn": llx.StringData(""),
			"azureAtlasAppId":    llx.StringData(""),
			"azureTenantId":      llx.StringData(""),
			"gcpServiceAccount":  llx.StringData(sa.GetGcpServiceAccountForAtlas()),
			"authorizedDate":     llx.TimeDataPtr(timePtr(sa.GetCreatedDate())),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}
