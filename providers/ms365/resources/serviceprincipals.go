// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/microsoftgraph/msgraph-sdk-go/serviceprincipals"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers/ms365/connection"
)

func (m *mqlMicrosoftServiceprincipal) id() (string, error) {
	return m.Id.Data, nil
}

func (a *mqlMicrosoft) serviceprincipals() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := graphClient(conn)
	if err != nil {
		return nil, err
	}
	// TODO: we need to use Top, there are more than 100 SPs.
	ctx := context.Background()
	resp, err := graphClient.ServicePrincipals().Get(ctx, &serviceprincipals.ServicePrincipalsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}

	res := []interface{}{}
	sps := resp.GetValue()
	for _, sp := range sps {
		mqlResource, err := CreateResource(a.MqlRuntime, "microsoft.serviceprincipal",
			map[string]*llx.RawData{
				"id": llx.StringData(convert.ToString(sp.GetId())),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}
