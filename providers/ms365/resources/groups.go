// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/microsoftgraph/msgraph-sdk-go/groups"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers/ms365/connection"
)

func (m *mqlMicrosoftGroup) id() (string, error) {
	return m.Id.Data, nil
}

func (a *mqlMicrosoftGroup) members() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *mqlMicrosoft) groups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.Groups().Get(ctx, &groups.GroupsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}

	res := []interface{}{}
	grps := resp.GetValue()
	for _, grp := range grps {
		graphGrp, err := CreateResource(a.MqlRuntime, "microsoft.group",
			map[string]*llx.RawData{
				"id":              llx.StringDataPtr(grp.GetId()),
				"displayName":     llx.StringDataPtr(grp.GetDisplayName()),
				"mail":            llx.StringDataPtr(grp.GetMail()),
				"mailEnabled":     llx.BoolDataPtr(grp.GetMailEnabled()),
				"mailNickname":    llx.StringDataPtr(grp.GetMailNickname()),
				"securityEnabled": llx.BoolDataPtr(grp.GetSecurityEnabled()),
				"visibility":      llx.StringDataPtr(grp.GetVisibility()),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, graphGrp)
	}

	return res, nil
}
