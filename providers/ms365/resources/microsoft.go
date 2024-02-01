// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/microsoftgraph/msgraph-sdk-go/organization"
	"go.mondoo.com/cnquery/v10/providers/ms365/connection"
)

func (a *mqlMicrosoft) tenantDomainName() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return "", err
	}
	ctx := context.Background()
	resp, err := graphClient.Organization().Get(ctx, &organization.OrganizationRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return "", transformError(err)
	}
	tenantDomainName := ""
	for _, org := range resp.GetValue() {
		for _, d := range org.GetVerifiedDomains() {
			if *d.GetIsInitial() {
				tenantDomainName = *d.GetName()
			}
		}
	}

	return tenantDomainName, nil
}
