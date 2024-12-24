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

func (a *mqlAzureSubscriptionPolicy) assignments() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := a.SubscriptionId.Data

	armConn, err := getArmSecurityConnection(ctx, conn, subId)
	if err != nil {
		return nil, err
	}

	pas, err := getPolicyAssignments(ctx, armConn)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for _, assignment := range pas.PolicyAssignments {
		assignmentData := map[string]*llx.RawData{
			"__id":            llx.StringData(fmt.Sprintf("azure.subscription.policy/%s/%s", assignment.Properties.Scope, assignment.Properties.DisplayName)),
			"id":              llx.StringData(assignment.Properties.PolicyDefinitionID),
			"name":            llx.StringData(assignment.Properties.DisplayName),
			"scope":           llx.StringData(assignment.Properties.Scope),
			"description":     llx.StringData(assignment.Properties.Description),
			"enforcementMode": llx.StringData(assignment.Properties.EnforcementMode),
			"parameters":      llx.StringData(assignment.Properties.Parameters),
		}

		mqlAssignment, err := CreateResource(a.MqlRuntime, "azure.subscription.policy.assignment", assignmentData)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAssignment)
	}
	return res, nil
}
