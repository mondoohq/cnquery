// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"
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
	// the response when asking for the user registration details
	mfaResp mfaResp
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
