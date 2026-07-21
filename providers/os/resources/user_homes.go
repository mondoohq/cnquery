// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// targetUser is a real (non-system) user account on the target.
type targetUser struct {
	name string
	home string
	uid  int64
}

// sharedProfileLeaves are home leaf names for shared/system profiles, not real
// users (e.g. /Users/Shared, C:\Users\Public).
var sharedProfileLeaves = map[string]struct{}{
	"shared":       {},
	"public":       {},
	"default":      {},
	"default user": {},
	"all users":    {},
}

// isRealUserHome reports whether home is a genuine per-user home. Superset of
// the browser/editor filters: a known user-home prefix (incl. FreeBSD's
// /usr/home) minus shared/system profile dirs.
func isRealUserHome(home string) bool {
	if isSystemHomeDir(home) {
		return false
	}
	leaf := home
	if i := strings.LastIndexAny(leaf, `/\`); i >= 0 {
		leaf = leaf[i+1:]
	}
	if _, ok := sharedProfileLeaves[strings.ToLower(leaf)]; ok {
		return false
	}
	return true
}

// targetUserHomes enumerates every real user on the target, deduped by home.
// Shared basis for resources inspecting per-user state across all accounts
// (browser/editor extensions, AI-agent skills). Missing uid defaults to -1.
func targetUserHomes(runtime *plugin.Runtime) ([]targetUser, error) {
	usersResource, err := CreateResource(runtime, "users", map[string]*llx.RawData{})
	if err != nil {
		return nil, fmt.Errorf("cannot list users on target: %w", err)
	}

	userList := usersResource.(*mqlUsers).GetList()
	if userList.Error != nil {
		return nil, fmt.Errorf("cannot list users on target: %w", userList.Error)
	}

	seen := map[string]struct{}{}
	var result []targetUser
	for _, u := range userList.Data {
		user := u.(*mqlUser)
		home := user.GetHome().Data
		if !isRealUserHome(home) {
			continue
		}
		if _, ok := seen[home]; ok {
			continue
		}
		seen[home] = struct{}{}

		uid := int64(-1)
		if uidVal := user.GetUid(); uidVal.Error == nil {
			uid = uidVal.Data
		}
		result = append(result, targetUser{name: user.GetName().Data, home: home, uid: uid})
	}
	return result, nil
}
