// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/microsoftgraph/msgraph-sdk-go/organization"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
)

type mqlMicrosoftInternal struct {
	// index users by id
	idxUsersById map[string]*mqlMicrosoftUser
	// index users by principal name
	idxUsersByPrincipalName map[string]*mqlMicrosoftUser
}

// initIndex ensures the user indexes are initialized,
// can be called multiple times without side effects
func (a *mqlMicrosoft) initIndex() {
	if a.idxUsersById == nil {
		a.idxUsersById = make(map[string]*mqlMicrosoftUser)
	}
	if a.idxUsersByPrincipalName == nil {
		a.idxUsersByPrincipalName = make(map[string]*mqlMicrosoftUser)
	}
}

// index adds a user to the internal indexes
func (a *mqlMicrosoft) index(user *mqlMicrosoftUser) {
	a.initIndex()
	a.idxUsersById[user.Id.Data] = user
	a.idxUsersByPrincipalName[user.UserPrincipalName.Data] = user
}

// userById returns a user by id if it exists in the index
func (a *mqlMicrosoft) userById(id string) (*mqlMicrosoftUser, bool) {
	if a.idxUsersById == nil {
		return nil, false
	}

	res, ok := a.idxUsersById[id]
	return res, ok
}

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
