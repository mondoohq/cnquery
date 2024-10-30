// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"
)

var idxUsersById = &sync.RWMutex{}

type mfaResp struct {
	// holds the error if that is what the request returned
	err    error
	mfaMap map[string]bool
}

type mqlMicrosoftInternal struct {
	permissionIndexer
	// index users by id
	idxUsersById map[string]*mqlMicrosoftUser
	// the response when asking for the user registration details
	mfaResp mfaResp
}

// initIndex ensures the user indexes are initialized,
// can be called multiple times without side effects
func (a *mqlMicrosoft) initIndex() {
	if a.idxUsersById == nil {
		a.idxUsersById = make(map[string]*mqlMicrosoftUser)
	}
}

// index adds a user to the internal indexes
func (a *mqlMicrosoft) index(user *mqlMicrosoftUser) {
	a.initIndex()
	idxUsersById.Lock()
	a.idxUsersById[user.Id.Data] = user
	idxUsersById.Unlock()
}

// userById returns a user by id if it exists in the index
func (a *mqlMicrosoft) userById(id string) (*mqlMicrosoftUser, bool) {
	if a.idxUsersById == nil {
		return nil, false
	}

	idxUsersById.RLock()
	res, ok := a.idxUsersById[id]
	idxUsersById.RUnlock()
	return res, ok
}
