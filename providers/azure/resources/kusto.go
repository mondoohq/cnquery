// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/kusto/armkusto"
	"github.com/rs/zerolog/log"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func initAzureSubscriptionKustoService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	args["subscriptionId"] = llx.StringData(conn.SubId())
	return args, nil, nil
}

func (a *mqlAzureSubscriptionKustoService) id() (string, error) {
	return "azure.subscription.kustoService/" + a.SubscriptionId.Data, nil
}

func (a *mqlAzureSubscriptionKustoServiceCluster) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionKustoService) clusters() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	ctx := context.Background()
	client, err := armkusto.NewClustersClient(a.SubscriptionId.Data, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list Data Explorer clusters due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, cluster := range page.Value {
			if cluster == nil {
				continue
			}
			mqlCluster, err := kustoClusterToMql(a.MqlRuntime, cluster)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlCluster)
		}
	}
	return res, nil
}

func kustoClusterToMql(runtime *plugin.Runtime, cluster *armkusto.Cluster) (*mqlAzureSubscriptionKustoServiceCluster, error) {
	sku, err := convert.JsonToDict(cluster.SKU)
	if err != nil {
		return nil, err
	}
	identity, err := convert.JsonToDict(cluster.Identity)
	if err != nil {
		return nil, err
	}

	var uri, dataIngestionURI, state, provisioningState string
	var publicNetworkAccess, publicIPType, restrictOutbound string
	var cmkKeyName, cmkKeyVaultURI, cmkKeyVersion string
	var enableDiskEncryption, enableDoubleEncryption, enableStreamingIngest, enablePurge bool
	allowedIPRangeList := []any{}
	allowedFqdnList := []any{}
	trustedExternalTenants := []any{}

	if p := cluster.Properties; p != nil {
		uri = convert.ToValue(p.URI)
		dataIngestionURI = convert.ToValue(p.DataIngestionURI)
		state = string(convert.ToValue(p.State))
		provisioningState = string(convert.ToValue(p.ProvisioningState))
		publicNetworkAccess = string(convert.ToValue(p.PublicNetworkAccess))
		publicIPType = string(convert.ToValue(p.PublicIPType))
		restrictOutbound = string(convert.ToValue(p.RestrictOutboundNetworkAccess))
		enableDiskEncryption = convert.ToValue(p.EnableDiskEncryption)
		enableDoubleEncryption = convert.ToValue(p.EnableDoubleEncryption)
		enableStreamingIngest = convert.ToValue(p.EnableStreamingIngest)
		enablePurge = convert.ToValue(p.EnablePurge)
		for _, ip := range p.AllowedIPRangeList {
			if ip != nil {
				allowedIPRangeList = append(allowedIPRangeList, *ip)
			}
		}
		for _, fqdn := range p.AllowedFqdnList {
			if fqdn != nil {
				allowedFqdnList = append(allowedFqdnList, *fqdn)
			}
		}
		for _, t := range p.TrustedExternalTenants {
			if t != nil && t.Value != nil {
				trustedExternalTenants = append(trustedExternalTenants, *t.Value)
			}
		}
		if kv := p.KeyVaultProperties; kv != nil {
			cmkKeyName = convert.ToValue(kv.KeyName)
			cmkKeyVaultURI = convert.ToValue(kv.KeyVaultURI)
			cmkKeyVersion = convert.ToValue(kv.KeyVersion)
		}
	}

	res, err := CreateResource(runtime, "azure.subscription.kustoService.cluster",
		map[string]*llx.RawData{
			"id":                            llx.StringDataPtr(cluster.ID),
			"name":                          llx.StringDataPtr(cluster.Name),
			"location":                      llx.StringDataPtr(cluster.Location),
			"tags":                          llx.MapData(convert.PtrMapStrToInterface(cluster.Tags), types.String),
			"sku":                           llx.DictData(sku),
			"identity":                      llx.DictData(identity),
			"uri":                           llx.StringData(uri),
			"dataIngestionUri":              llx.StringData(dataIngestionURI),
			"state":                         llx.StringData(state),
			"provisioningState":             llx.StringData(provisioningState),
			"publicNetworkAccess":           llx.StringData(publicNetworkAccess),
			"publicIpType":                  llx.StringData(publicIPType),
			"restrictOutboundNetworkAccess": llx.StringData(restrictOutbound),
			"enableDiskEncryption":          llx.BoolData(enableDiskEncryption),
			"enableDoubleEncryption":        llx.BoolData(enableDoubleEncryption),
			"enableStreamingIngest":         llx.BoolData(enableStreamingIngest),
			"enablePurge":                   llx.BoolData(enablePurge),
			"allowedIpRangeList":            llx.ArrayData(allowedIPRangeList, types.String),
			"allowedFqdnList":               llx.ArrayData(allowedFqdnList, types.String),
			"trustedExternalTenants":        llx.ArrayData(trustedExternalTenants, types.String),
			"cmkKeyName":                    llx.StringData(cmkKeyName),
			"cmkKeyVaultUri":                llx.StringData(cmkKeyVaultURI),
			"cmkKeyVersion":                 llx.StringData(cmkKeyVersion),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionKustoServiceCluster), nil
}
