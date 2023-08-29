// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	subscriptions "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers/azure/connection"
	"go.mondoo.com/cnquery/types"
)

// TODO: we need to make this NOT go through init, this is heavy on the API
// every request calls the  API
func initAzureSubscription(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.AzureConnection)

	subscriptionsC, err := subscriptions.NewClient(conn.Token(), &arm.ClientOptions{})
	if err != nil {
		return nil, nil, err
	}
	ctx := context.Background()
	resp, err := subscriptionsC.Get(ctx, conn.SubId(), &subscriptions.ClientGetOptions{})
	if err != nil {
		return nil, nil, err
	}

	managedByTenants := []interface{}{}
	for _, t := range resp.ManagedByTenants {
		if t != nil {
			managedByTenants = append(managedByTenants, *t.TenantID)
		}
	}
	subPolicies, err := convert.JsonToDict(resp.SubscriptionPolicies)
	if err != nil {
		return nil, nil, err
	}
	args["id"] = llx.StringData(convert.ToString(resp.ID))
	args["name"] = llx.StringData(convert.ToString(resp.DisplayName))
	args["tenantId"] = llx.StringData(convert.ToString(resp.TenantID))
	args["tags"] = llx.MapData(convert.PtrMapStrToInterface(resp.Tags), types.String)
	args["state"] = llx.StringData(convert.ToString((*string)(resp.State)))
	args["subscriptionId"] = llx.StringData(convert.ToString(resp.SubscriptionID))
	args["authorizationSource"] = llx.StringData(convert.ToString((*string)(resp.AuthorizationSource)))
	args["managedByTenants"] = llx.ArrayData(managedByTenants, types.String)
	args["subscriptionsPolicies"] = llx.DictData(subPolicies)

	return args, nil, nil
}

func (a *mqlAzureSubscription) id() (string, error) {
	return a.Id.Data, nil
}
