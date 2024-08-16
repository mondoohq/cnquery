// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/azure/connection"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

func initAzureSubscriptionPolicy(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionPolicyAssignment) id() (string, error) {
	// Ensure that all the parts of the ID are available
	if a.Scope.Data == "" || a.Name.Data == "" {
		return "", errors.New("missing required fields to generate id")
	}

	return fmt.Sprintf("azure.subscription.policy/%s/%s", a.Scope.Data, a.Name.Data), nil
}

func (a *mqlAzureSubscriptionPolicy) policyAssignments() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	rawToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.core.windows.net//.default"},
	})
	if err != nil {
		return nil, err
	}

	ep := cloud.AzurePublic.Services[cloud.ResourceManager].Endpoint
	pas, err := getPolicyAssignments(ctx, subId, ep, rawToken.Token)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for _, assignment := range pas.PolicyAssignments {
		assignmentData := map[string]*llx.RawData{
			"name":  llx.StringData(assignment.Properties.DisplayName),
			"scope": llx.StringData(assignment.Properties.Scope),
			"type":  llx.StringData(assignment.Type),
		}

		mqlAssignment, err := CreateResource(a.MqlRuntime, "azure.subscription.policy.assignment", assignmentData)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAssignment)
	}
	return res, nil
}
