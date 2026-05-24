// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	betamodels "github.com/microsoftgraph/msgraph-beta-sdk-go/models"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/reports"
	"go.mondoo.com/mql/v13/providers/ms365/connection"
)

var idxUsersById = &sync.RWMutex{}
var idxDevicesById = &sync.RWMutex{}

type mfaResp struct {
	// holds the error if that is what the request returned
	err    error
	mfaMap map[string]bool
}

type mqlMicrosoftInternal struct {
	permissionIndexer
	// index users by id
	idxUsersById map[string]*mqlMicrosoftUser
	// index devices by id
	idxDevicesById map[string]*mqlMicrosoftDevice
	// guards mfaResp; ensures the MFA registration fetch only runs once
	mfaOnce sync.Once
	// the response when asking for the user registration details
	mfaResp mfaResp
}

// loadMfaResp lazily fetches user MFA registration details from the beta
// Graph API and caches the result on a.mfaResp. Both microsoft.users.list
// (eager batch) and microsoft.user.mfaEnabled (per-user lookup) call this
// so the data is available regardless of which path was queried first.
func (a *mqlMicrosoft) loadMfaResp() *mfaResp {
	a.mfaOnce.Do(func() {
		conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
		betaClient, err := conn.BetaGraphClient()
		if err != nil {
			a.mfaResp = mfaResp{err: err}
			return
		}

		ctx := context.Background()
		top := int32(999)
		resp, err := betaClient.
			Reports().
			AuthenticationMethods().
			UserRegistrationDetails().
			Get(ctx, &reports.AuthenticationMethodsUserRegistrationDetailsRequestBuilderGetRequestConfiguration{
				QueryParameters: &reports.AuthenticationMethodsUserRegistrationDetailsRequestBuilderGetQueryParameters{
					Top: &top,
				},
			})
		// a failure here typically means the tenant lacks the required license;
		// store the error so mfaEnabled can surface it but don't fail callers.
		if err != nil {
			a.mfaResp = mfaResp{err: err}
			return
		}

		details, err := iterate[*betamodels.UserRegistrationDetails](ctx, resp, betaClient.GetAdapter(), betamodels.CreateUserRegistrationDetailsCollectionResponseFromDiscriminatorValue)
		if err != nil {
			a.mfaResp = mfaResp{err: err}
			return
		}

		mfaMap := map[string]bool{}
		for _, u := range details {
			if u.GetId() == nil || u.GetIsMfaRegistered() == nil {
				continue
			}
			mfaMap[*u.GetId()] = *u.GetIsMfaRegistered()
		}
		a.mfaResp = mfaResp{mfaMap: mfaMap}
	})
	return &a.mfaResp
}

// initIndex ensures the user indexes are initialized,
// can be called multiple times without side effects
func (a *mqlMicrosoft) initIndex() {
	if a.idxUsersById == nil {
		a.idxUsersById = make(map[string]*mqlMicrosoftUser)
	}
	if a.idxDevicesById == nil {
		a.idxDevicesById = make(map[string]*mqlMicrosoftDevice)
	}
}

// indexUser adds a user to the internal indexes
func (a *mqlMicrosoft) indexUser(user *mqlMicrosoftUser) {
	a.initIndex()
	idxUsersById.Lock()
	a.idxUsersById[user.Id.Data] = user
	idxUsersById.Unlock()
}

// userById returns a user by id if it exists in the indexUser
func (a *mqlMicrosoft) userById(id string) (*mqlMicrosoftUser, bool) {
	if a.idxUsersById == nil {
		return nil, false
	}

	idxUsersById.RLock()
	res, ok := a.idxUsersById[id]
	idxUsersById.RUnlock()
	return res, ok
}

// indexDevice adds a device to the internal indexes
func (a *mqlMicrosoft) indexDevice(device *mqlMicrosoftDevice) {
	a.initIndex()
	idxUsersById.Lock()
	a.idxDevicesById[device.Id.Data] = device
	idxUsersById.Unlock()
}

// deviceById returns a device by id if it exists in the indexDevice
func (a *mqlMicrosoft) deviceById(id string) (*mqlMicrosoftDevice, bool) {
	if a.idxDevicesById == nil {
		return nil, false
	}

	idxDevicesById.RLock()
	res, ok := a.idxDevicesById[id]
	idxDevicesById.RUnlock()
	return res, ok
}
