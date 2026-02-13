// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/databricks/armdatabricks"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func initAzureSubscriptionDatabricksService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionDatabricksService) id() (string, error) {
	return "azure.subscription.databricksService/" + a.SubscriptionId.Data, nil
}

func (a *mqlAzureSubscriptionDatabricksServiceWorkspace) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionDatabricksService) workspaces() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, errors.New("invalid connection provided. it is not an Azure connection")
	}

	ctx := context.Background()
	client, err := armdatabricks.NewWorkspacesClient(a.SubscriptionId.Data, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListBySubscriptionPager(&armdatabricks.WorkspacesClientListBySubscriptionOptions{})
	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, entry := range page.Value {
			if entry == nil {
				continue
			}

			resource, err := databricksWorkspaceToMql(a.MqlRuntime, entry)
			if err != nil {
				return nil, err
			}
			res = append(res, resource)
		}
	}

	return res, nil
}

func databricksWorkspaceToMql(runtime *plugin.Runtime, workspace *armdatabricks.Workspace) (*mqlAzureSubscriptionDatabricksServiceWorkspace, error) {
	propertiesData := llx.NilData
	if workspace.Properties != nil {
		if dict, err := convert.JsonToDict(workspace.Properties); err != nil {
			return nil, err
		} else if dict != nil {
			propertiesData = llx.DictData(dict)
		}
	}

	skuData := llx.NilData
	if workspace.SKU != nil {
		if dict, err := convert.JsonToDict(workspace.SKU); err != nil {
			return nil, err
		} else if dict != nil {
			skuData = llx.DictData(dict)
		}
	}

	res, err := CreateResource(runtime, ResourceAzureSubscriptionDatabricksServiceWorkspace, map[string]*llx.RawData{
		"id":         llx.StringDataPtr(workspace.ID),
		"name":       llx.StringDataPtr(workspace.Name),
		"location":   llx.StringDataPtr(workspace.Location),
		"tags":       llx.MapData(convert.PtrMapStrToInterface(workspace.Tags), types.String),
		"type":       llx.StringDataPtr(workspace.Type),
		"properties": propertiesData,
		"sku":        skuData,
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlAzureSubscriptionDatabricksServiceWorkspace), nil
}
