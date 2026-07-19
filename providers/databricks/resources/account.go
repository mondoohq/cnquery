// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strconv"

	"github.com/databricks/databricks-sdk-go/service/provisioning"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

type mqlDatabricksWorkspaceInternal struct {
	cacheNetworkId               string
	cachePrivateAccessSettingsId string
}

func (r *mqlDatabricks) workspaces() ([]any, error) {
	acc, err := accountClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	workspaces, err := acc.Workspaces.List(context.Background())
	if err != nil {
		return nil, err
	}

	list := []any{}
	for i := range workspaces {
		mqlWorkspace, err := newMqlDatabricksWorkspace(r.MqlRuntime, workspaces[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlWorkspace)
	}
	return list, nil
}

func newMqlDatabricksWorkspace(runtime *plugin.Runtime, ws provisioning.Workspace) (*mqlDatabricksWorkspace, error) {
	res, err := CreateResource(runtime, "databricks.workspace", map[string]*llx.RawData{
		"__id":                                llx.StringData("databricks.workspace/" + strconv.FormatInt(ws.WorkspaceId, 10)),
		"workspaceId":                         llx.IntData(ws.WorkspaceId),
		"name":                                llx.StringData(ws.WorkspaceName),
		"deploymentName":                      llx.StringData(ws.DeploymentName),
		"status":                              llx.StringData(string(ws.WorkspaceStatus)),
		"statusMessage":                       llx.StringData(ws.WorkspaceStatusMessage),
		"pricingTier":                         llx.StringData(string(ws.PricingTier)),
		"cloud":                               llx.StringData(ws.Cloud),
		"awsRegion":                           llx.StringData(ws.AwsRegion),
		"location":                            llx.StringData(ws.Location),
		"managedServicesCustomerManagedKeyId": llx.StringData(ws.ManagedServicesCustomerManagedKeyId),
		"storageCustomerManagedKeyId":         llx.StringData(ws.StorageCustomerManagedKeyId),
		"customTags":                          llx.MapData(strMap(ws.CustomTags), types.String),
		"creationTime":                        llx.TimeDataPtr(epochMsTime(ws.CreationTime)),
	})
	if err != nil {
		return nil, err
	}
	mqlWorkspace := res.(*mqlDatabricksWorkspace)
	mqlWorkspace.cacheNetworkId = ws.NetworkId
	mqlWorkspace.cachePrivateAccessSettingsId = ws.PrivateAccessSettingsId
	return mqlWorkspace, nil
}

func (r *mqlDatabricksWorkspace) network() (*mqlDatabricksNetwork, error) {
	if r.cacheNetworkId == "" {
		r.Network.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	acc, err := accountClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	net, err := acc.Networks.Get(context.Background(), provisioning.GetNetworkRequest{NetworkId: r.cacheNetworkId})
	if err != nil {
		return nil, err
	}
	if net == nil {
		r.Network.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return newMqlDatabricksNetwork(r.MqlRuntime, *net)
}

func (r *mqlDatabricksWorkspace) privateAccessSettings() (*mqlDatabricksPrivateAccessSetting, error) {
	if r.cachePrivateAccessSettingsId == "" {
		r.PrivateAccessSettings.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	acc, err := accountClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	pas, err := acc.PrivateAccess.Get(context.Background(), provisioning.GetPrivateAccesRequest{PrivateAccessSettingsId: r.cachePrivateAccessSettingsId})
	if err != nil {
		return nil, err
	}
	if pas == nil {
		r.PrivateAccessSettings.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return newMqlDatabricksPrivateAccessSetting(r.MqlRuntime, *pas)
}

func (r *mqlDatabricks) metastores() ([]any, error) {
	acc, err := accountClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	metastores, err := acc.Metastores.ListAll(context.Background())
	if err != nil {
		return nil, err
	}

	list := []any{}
	for i := range metastores {
		m := metastores[i]
		res, err := CreateResource(r.MqlRuntime, "databricks.metastore", map[string]*llx.RawData{
			"__id":              llx.StringData("databricks.metastore/" + m.MetastoreId),
			"id":                llx.StringData(m.MetastoreId),
			"name":              llx.StringData(m.Name),
			"globalMetastoreId": llx.StringData(m.GlobalMetastoreId),
			"owner":             llx.StringData(m.Owner),
			"cloud":             llx.StringData(m.Cloud),
			"region":            llx.StringData(m.Region),
			"storageRoot":       llx.StringData(m.StorageRoot),
			"deltaSharingScope": llx.StringData(string(m.DeltaSharingScope)),
			"deltaSharingRecipientTokenLifetimeInSeconds": llx.IntData(m.DeltaSharingRecipientTokenLifetimeInSeconds),
			"externalAccessEnabled":                       llx.BoolData(m.ExternalAccessEnabled),
			"privilegeModelVersion":                       llx.StringData(m.PrivilegeModelVersion),
			"createdAt":                                   llx.TimeDataPtr(epochMsTime(m.CreatedAt)),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}

func (r *mqlDatabricks) networks() ([]any, error) {
	acc, err := accountClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	networks, err := acc.Networks.List(context.Background())
	if err != nil {
		return nil, err
	}

	list := []any{}
	for i := range networks {
		mqlNetwork, err := newMqlDatabricksNetwork(r.MqlRuntime, networks[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlNetwork)
	}
	return list, nil
}

func newMqlDatabricksNetwork(runtime *plugin.Runtime, net provisioning.Network) (*mqlDatabricksNetwork, error) {
	res, err := CreateResource(runtime, "databricks.network", map[string]*llx.RawData{
		"__id":             llx.StringData("databricks.network/" + net.NetworkId),
		"id":               llx.StringData(net.NetworkId),
		"networkName":      llx.StringData(net.NetworkName),
		"vpcId":            llx.StringData(net.VpcId),
		"subnetIds":        llx.ArrayData(strSlice(net.SubnetIds), types.String),
		"securityGroupIds": llx.ArrayData(strSlice(net.SecurityGroupIds), types.String),
		"vpcStatus":        llx.StringData(string(net.VpcStatus)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlDatabricksNetwork), nil
}

func (r *mqlDatabricks) privateAccessSettings() ([]any, error) {
	acc, err := accountClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	settings, err := acc.PrivateAccess.List(context.Background())
	if err != nil {
		return nil, err
	}

	list := []any{}
	for i := range settings {
		mqlPas, err := newMqlDatabricksPrivateAccessSetting(r.MqlRuntime, settings[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlPas)
	}
	return list, nil
}

func newMqlDatabricksPrivateAccessSetting(runtime *plugin.Runtime, pas provisioning.PrivateAccessSettings) (*mqlDatabricksPrivateAccessSetting, error) {
	res, err := CreateResource(runtime, "databricks.privateAccessSetting", map[string]*llx.RawData{
		"__id":                  llx.StringData("databricks.privateAccessSetting/" + pas.PrivateAccessSettingsId),
		"id":                    llx.StringData(pas.PrivateAccessSettingsId),
		"name":                  llx.StringData(pas.PrivateAccessSettingsName),
		"region":                llx.StringData(pas.Region),
		"publicAccessEnabled":   llx.BoolData(pas.PublicAccessEnabled),
		"privateAccessLevel":    llx.StringData(string(pas.PrivateAccessLevel)),
		"allowedVpcEndpointIds": llx.ArrayData(strSlice(pas.AllowedVpcEndpointIds), types.String),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlDatabricksPrivateAccessSetting), nil
}
