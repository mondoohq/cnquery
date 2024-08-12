// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"github.com/microsoftgraph/msgraph-sdk-go/groups"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func (m *mqlMicrosoftGroup) id() (string, error) {
	return m.Id.Data, nil
}

func (a *mqlMicrosoftGroup) members() ([]interface{}, error) {
	msResource, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "microsoft", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	mqlMicrosoftResource := msResource.(*mqlMicrosoft)

	groupId := a.Id.Data
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}
	top := int32(200)

	queryParams := &groups.ItemMembersRequestBuilderGetQueryParameters{
		Top: &top,
	}
	ctx := context.Background()
	resp, err := graphClient.Groups().ByGroupId(groupId).Members().Get(ctx, &groups.ItemMembersRequestBuilderGetRequestConfiguration{
		QueryParameters: queryParams,
	})
	if err != nil {
		return nil, transformError(err)
	}

	res := []interface{}{}
	for _, member := range resp.GetValue() {
		memberId := member.GetId()
		if memberId == nil {
			continue
		}

		if member.GetOdataType() != nil && *member.GetOdataType() != "#microsoft.graph.user" {
			continue
		}

		// if the user is already indexed, we can reuse it
		userResource, ok := mqlMicrsoftResource.userById(*memberId)
		if ok {
			res = append(res, userResource)
			continue
		}

		newUserResource, err := a.MqlRuntime.NewResource(a.MqlRuntime, "microsoft.user", map[string]*llx.RawData{
			"id": llx.StringDataPtr(memberId),
		})
		if err != nil {
			return nil, err
		}
		mqlMicrsoftResource.index(newUserResource.(*mqlMicrosoftUser))
		res = append(res, newUserResource)
	}
	return res, nil
}

func (a *mqlMicrosoft) groups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	top := int32(200)
	queryParams := &groups.GroupsRequestBuilderGetQueryParameters{
		Top: &top,
	}
	ctx := context.Background()
	resp, err := graphClient.Groups().Get(ctx, &groups.GroupsRequestBuilderGetRequestConfiguration{
		QueryParameters: queryParams,
	})
	if err != nil {
		return nil, transformError(err)
	}
	grps, err := iterate[*models.Group](ctx, resp, graphClient.GetAdapter(), groups.CreateDeltaGetResponseFromDiscriminatorValue)
	if err != nil {
		return nil, transformError(err)
	}
	res := []interface{}{}
	for _, grp := range grps {
		graphGrp, err := CreateResource(a.MqlRuntime, "microsoft.group",
			map[string]*llx.RawData{
				"id":                            llx.StringDataPtr(grp.GetId()),
				"displayName":                   llx.StringDataPtr(grp.GetDisplayName()),
				"mail":                          llx.StringDataPtr(grp.GetMail()),
				"mailEnabled":                   llx.BoolDataPtr(grp.GetMailEnabled()),
				"mailNickname":                  llx.StringDataPtr(grp.GetMailNickname()),
				"securityEnabled":               llx.BoolDataPtr(grp.GetSecurityEnabled()),
				"visibility":                    llx.StringDataPtr(grp.GetVisibility()),
				"groupTypes":                    llx.ArrayData(llx.TArr2Raw(grp.GetGroupTypes()), types.String),
				"membershipRule":                llx.StringDataPtr(grp.GetMembershipRule()),
				"membershipRuleProcessingState": llx.StringDataPtr(grp.GetMembershipRuleProcessingState()),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, graphGrp)
	}

	return res, nil
}
