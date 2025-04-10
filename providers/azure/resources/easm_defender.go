// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/azure/connection"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/security/armsecurity"
)

func (a *mqlAzureSubscription) easmDefender() (*mqlAzureSubscriptionEasmDefenderService, error) {
	svc, err := NewResource(a.MqlRuntime, "azure.subscription.easmDefenderService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	return svc.(*mqlAzureSubscriptionEasmDefenderService), nil
}

func (a *mqlAzureSubscriptionEasmDefenderService) id() (string, error) {
	return "azure.subscription.cloudDefender/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionEasmDefenderService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionEasmDefenderServiceWorkspace) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionEasmDefenderService) workspaces() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	clientFactory, err := armsecurity.NewClientFactory(subId, token, nil)
	if err != nil {
		return nil, err
	}
	// @afiune there is no SDK implementation just yet, EASM is in preview mode
	// https://learn.microsoft.com/en-us/rest/api/defenderforeasm/?view=rest-defenderforeasm-controlplanepreview-2023-04-01-preview
	clientFactory.FetchWorkspacesFromEASM(ctx)

	return nil, nil
}
