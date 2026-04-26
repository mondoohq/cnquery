// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"sync"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/gitlab/connection"
)

// mqlGitlabUserInternal caches a fetched *gitlab.User so that multiple computed
// methods (externalIdentities, etc.) only trigger a single GetUser API call.
//
// cacheIdentities lets producers (e.g. group/project members()) seed identities
// from the *gitlab.User they already have, so externalIdentities() doesn't need
// to call GetUser at all - eliminating an N+1 across N members.
type mqlGitlabUserInternal struct {
	fetched         bool
	user            *gitlab.User
	cacheIdentities []*gitlab.UserIdentity
	lock            sync.Mutex
}

// initGitlabUser supports `gitlab.user(id: <int>)` lookups so typed back-references
// from sshKey/externalIdentity resolve to the typed user via NewResource.
//
// When called with just an id (or empty args), we lazily fetch the full user
// from the API. If callers already supplied populated args (e.g. from members())
// we leave them alone.
func initGitlabUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// Only auto-fetch if just id was provided (avoids re-fetching when members()
	// or other producers already populated all fields). Other init functions in
	// this provider use the same > 2 threshold to account for an implicit __id arg.
	if len(args) > 2 {
		return args, nil, nil
	}
	idArg, ok := args["id"]
	if !ok || idArg == nil || idArg.Error != nil {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.GitLabConnection)
	user, resp, err := conn.Client().Users.GetUser(idArg.Value.(int64), gitlab.GetUsersOptions{})
	if err != nil {
		// Non-admin tokens get 403/404 from /users/:id. Let the resource exist
		// with whatever id was passed in so typed back-refs (sshKey.user, etc.)
		// don't blow up the whole resource graph.
		if resp != nil && (resp.StatusCode == 403 || resp.StatusCode == 404) {
			return args, nil, nil
		}
		return nil, nil, err
	}

	args["id"] = llx.IntData(user.ID)
	args["username"] = llx.StringData(user.Username)
	args["name"] = llx.StringData(user.Name)
	args["state"] = llx.StringData(user.State)
	args["email"] = llx.StringData(user.Email)
	args["webURL"] = llx.StringData(user.WebURL)
	args["avatarURL"] = llx.StringData(user.AvatarURL)
	args["createdAt"] = llx.TimeDataPtr(user.CreatedAt)
	args["jobTitle"] = llx.StringData(user.JobTitle)
	args["organization"] = llx.StringData(user.Organization)
	args["location"] = llx.StringData(user.Location)
	args["locked"] = llx.BoolData(user.Locked)
	args["bot"] = llx.BoolData(user.Bot)
	args["twoFactorEnabled"] = llx.BoolData(user.TwoFactorEnabled)

	return args, nil, nil
}

// fetchUser loads the full user record by ID (with double-checked locking).
// Used as a fallback when no creator seeded the user data. Returns (nil, nil)
// on 403/404 - non-admin tokens lack permission to read /users/:id but should
// not fail the whole resource graph.
func (u *mqlGitlabUser) fetchUser() (*gitlab.User, error) {
	if u.fetched {
		return u.user, nil
	}
	u.lock.Lock()
	defer u.lock.Unlock()
	if u.fetched {
		return u.user, nil
	}
	conn := u.MqlRuntime.Connection.(*connection.GitLabConnection)
	user, resp, err := conn.Client().Users.GetUser(u.Id.Data, gitlab.GetUsersOptions{})
	if err != nil {
		if resp != nil && (resp.StatusCode == 403 || resp.StatusCode == 404) {
			u.fetched = true
			return nil, nil
		}
		return nil, err
	}
	u.user = user
	u.fetched = true
	return u.user, nil
}

// id function for gitlab.user.externalIdentity
func (i *mqlGitlabUserExternalIdentity) id() (string, error) {
	return "gitlab.user.externalIdentity/" + i.Provider.Data + "/" + i.ExternUID.Data, nil
}

// mqlGitlabUserExternalIdentityInternal carries the parent user ID so the
// `user()` accessor can resolve back to the typed gitlab.user resource.
type mqlGitlabUserExternalIdentityInternal struct {
	userID int64
}

// user returns the typed gitlab.user this external identity is linked to.
func (i *mqlGitlabUserExternalIdentity) user() (*mqlGitlabUser, error) {
	if i.userID == 0 {
		i.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlUser, err := NewResource(i.MqlRuntime, "gitlab.user", map[string]*llx.RawData{
		"id": llx.IntData(i.userID),
	})
	if err != nil {
		return nil, err
	}
	return mqlUser.(*mqlGitlabUser), nil
}

// externalIdentities lists the SSO/external identities linked to this user.
//
// Uses cacheIdentities seeded by the creator (e.g. members()) to avoid an
// N+1 GetUser call per user. Falls back to fetchUser() only when a user was
// constructed via NewResource (lazy lookup), and that fall-back gracefully
// returns an empty list on 403/404 instead of failing the whole graph.
func (u *mqlGitlabUser) externalIdentities() ([]any, error) {
	identities := u.cacheIdentities
	if identities == nil {
		user, err := u.fetchUser()
		if err != nil {
			return nil, err
		}
		if user != nil {
			identities = user.Identities
		}
	}

	var mqlIdentities []any
	for _, identity := range identities {
		if identity == nil {
			continue
		}
		identityInfo := map[string]*llx.RawData{
			"provider":  llx.StringData(identity.Provider),
			"externUID": llx.StringData(identity.ExternUID),
		}
		mqlIdentity, err := CreateResource(u.MqlRuntime, "gitlab.user.externalIdentity", identityInfo)
		if err != nil {
			return nil, err
		}
		mqlIdentity.(*mqlGitlabUserExternalIdentity).userID = u.Id.Data
		mqlIdentities = append(mqlIdentities, mqlIdentity)
	}

	return mqlIdentities, nil
}

// id function for gitlab.user.sshKey
func (k *mqlGitlabUserSshKey) id() (string, error) {
	return "gitlab.user.sshKey/" + strconv.FormatInt(k.Id.Data, 10), nil
}

// mqlGitlabUserSshKeyInternal carries the parent user ID for the typed user() accessor.
type mqlGitlabUserSshKeyInternal struct {
	userID int64
}

// daysOld returns the age of the SSH key in days. Returns -1 when createdAt
// isn't set so callers can distinguish "missing data" from "fresh key".
func (k *mqlGitlabUserSshKey) daysOld() (int64, error) {
	if !k.CreatedAt.IsSet() || k.CreatedAt.Data == nil || k.CreatedAt.Data.IsZero() {
		return -1, nil
	}
	return int64(time.Since(*k.CreatedAt.Data).Hours() / 24), nil
}

// user returns the typed gitlab.user that owns this SSH key.
func (k *mqlGitlabUserSshKey) user() (*mqlGitlabUser, error) {
	if k.userID == 0 {
		k.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlUser, err := NewResource(k.MqlRuntime, "gitlab.user", map[string]*llx.RawData{
		"id": llx.IntData(k.userID),
	})
	if err != nil {
		return nil, err
	}
	return mqlUser.(*mqlGitlabUser), nil
}

// sshKeys lists SSH keys registered to this user. Requires admin or self-access;
// if the caller lacks permission GitLab returns 403/404 and we surface an empty list.
func (u *mqlGitlabUser) sshKeys() ([]any, error) {
	conn := u.MqlRuntime.Connection.(*connection.GitLabConnection)

	perPage := int64(50)
	page := int64(1)
	var allKeys []*gitlab.SSHKey

	for {
		keys, resp, err := conn.Client().Users.ListSSHKeysForUser(u.Id.Data, &gitlab.ListSSHKeysForUserOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			if resp != nil && (resp.StatusCode == 403 || resp.StatusCode == 404) {
				return []any{}, nil // no permission to list keys for this user
			}
			return nil, err
		}

		allKeys = append(allKeys, keys...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	var mqlKeys []any
	for _, key := range allKeys {
		keyInfo := map[string]*llx.RawData{
			"id":        llx.IntData(key.ID),
			"title":     llx.StringData(key.Title),
			"key":       llx.StringData(key.Key),
			"createdAt": llx.TimeDataPtr(key.CreatedAt),
			"expiresAt": llx.TimeDataPtr(key.ExpiresAt),
			"usageType": llx.StringData(key.UsageType),
		}
		mqlKey, err := CreateResource(u.MqlRuntime, "gitlab.user.sshKey", keyInfo)
		if err != nil {
			return nil, err
		}
		mqlKey.(*mqlGitlabUserSshKey).userID = u.Id.Data
		mqlKeys = append(mqlKeys, mqlKey)
	}

	return mqlKeys, nil
}
