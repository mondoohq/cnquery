// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

type mqlMicrosoftInternal struct {
	// index users by id
	idxUsersById map[string]*mqlMicrosoftUser
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
	a.idxUsersById[user.Id.Data] = user
}

// userById returns a user by id if it exists in the index
func (a *mqlMicrosoft) userById(id string) (*mqlMicrosoftUser, bool) {
	if a.idxUsersById == nil {
		return nil, false
	}

	res, ok := a.idxUsersById[id]
	return res, ok
}
