// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	"github.com/microsoftgraph/msgraph-sdk-go/groups"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/ms365/connection"
	"go.mondoo.com/cnquery/v12/types"
)

func (a *mqlMicrosoftGroup) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlMicrosoftGroup) members() ([]any, error) {
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
	resp, err := graphClient.Groups().
		ByGroupId(groupId).
		Members().
		Get(ctx, &groups.ItemMembersRequestBuilderGetRequestConfiguration{
			QueryParameters: queryParams,
		})
	if err != nil {
		return nil, transformError(err)
	}

	res := []any{}
	for _, member := range resp.GetValue() {
		memberId := member.GetId()
		if memberId == nil {
			continue
		}

		if member.GetOdataType() != nil && *member.GetOdataType() != "#microsoft.graph.user" {
			continue
		}

		// if the user is already indexed, we can reuse it
		userResource, ok := mqlMicrosoftResource.userById(*memberId)
		if ok {
			res = append(res, userResource)
			continue
		}

		newUserResource, err := a.MqlRuntime.
			NewResource(a.MqlRuntime, "microsoft.user", map[string]*llx.RawData{
				"id": llx.StringDataPtr(memberId),
			})
		if err != nil {
			return nil, err
		}
		mqlMicrosoftResource.indexUser(newUserResource.(*mqlMicrosoftUser))
		res = append(res, newUserResource)
	}
	return res, nil
}

func (a *mqlMicrosoft) groups() (*mqlMicrosoftGroups, error) {
	mqlResource, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "microsoft.groups", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	return mqlResource.(*mqlMicrosoftGroups), err
}

func initMicrosoftGroups(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	args["__id"] = newListResourceIdFromArguments("microsoft.groups", args)
	resource, err := runtime.CreateResource(runtime, "microsoft.groups", args)
	if err != nil {
		return args, nil, err
	}

	return args, resource.(*mqlMicrosoftGroups), nil
}

func (a *mqlMicrosoftGroups) list() ([]any, error) {
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
	res := []any{}
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

func (a *mqlMicrosoftGroup) owners() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	groupId := a.Id.Data
	ctx := context.Background()

	ownersResp, err := graphClient.Groups().
		ByGroupId(groupId).
		Owners().
		Get(ctx, &groups.ItemOwnersRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}

	var owners []any
	for _, owner := range ownersResp.GetValue() {
		if owner.GetId() == nil {
			continue
		}

		var displayName *string

		if owner.GetOdataType() != nil {
			switch *owner.GetOdataType() {
			case "#microsoft.graph.user":
				if user, ok := owner.(models.Userable); ok {
					displayName = user.GetDisplayName()
				}
			case "#microsoft.graph.servicePrincipal":
				if sp, ok := owner.(models.ServicePrincipalable); ok {
					displayName = sp.GetDisplayName()
				}
			}
		}

		ownerResource, err := CreateResource(a.MqlRuntime, "microsoft.group.owner",
			map[string]*llx.RawData{
				"__id":        llx.StringDataPtr(owner.GetId()),
				"id":          llx.StringDataPtr(owner.GetId()),
				"displayName": llx.StringDataPtr(displayName),
				"ownerType":   llx.StringDataPtr(owner.GetOdataType()),
			})
		if err != nil {
			return nil, err
		}
		owners = append(owners, ownerResource)
	}

	return owners, nil
}

func (a *mqlMicrosoftGroupOwner) user() (*mqlMicrosoftUser, error) {
	ownerType := a.OwnerType.Data
	if ownerType != "user" {
		return nil, nil
	}

	userId := a.Id.Data
	userResource, err := a.MqlRuntime.NewResource(a.MqlRuntime, "microsoft.user",
		map[string]*llx.RawData{
			"id": llx.StringData(userId),
		})
	if err != nil {
		return nil, err
	}
	return userResource.(*mqlMicrosoftUser), nil
}

func (a *mqlMicrosoftGroupOwner) servicePrincipal() (*mqlMicrosoftServiceprincipal, error) {
	ownerType := a.OwnerType.Data
	if ownerType != "servicePrincipal" {
		return nil, nil
	}

	spId := a.Id.Data
	spResource, err := a.MqlRuntime.NewResource(a.MqlRuntime, "microsoft.serviceprincipal",
		map[string]*llx.RawData{
			"id": llx.StringData(spId),
		})
	if err != nil {
		return nil, err
	}
	return spResource.(*mqlMicrosoftServiceprincipal), nil
}

func (a *mqlMicrosoftGroups) length() (int64, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return 0, err
	}

	opts := &groups.CountRequestBuilderGetRequestConfiguration{Headers: abstractions.NewRequestHeaders()}
	opts.Headers.Add("ConsistencyLevel", "eventual")
	length, err := graphClient.Groups().Count().Get(context.Background(), opts)
	if err != nil {
		return 0, err
	}
	if length == nil {
		// This should never happen, but we better check
		return 0, errors.New("unable to count groups, counter parameter API returned nil")
	}

	return int64(*length), nil
}
