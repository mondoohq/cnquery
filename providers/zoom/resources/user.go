// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"

	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/zoom/connection"
	"go.mondoo.com/mql/types"
)

// mqlZoomUserInternal caches the raw role and group IDs from list/init time
// so the role and groups accessors can resolve them into typed references
// without re-parsing the source payload.
type mqlZoomUserInternal struct {
	cacheRoleId   string
	cacheGroupIds []string
}

const usersPageSize = 300

// zoomSsoLoginType is the login_type Zoom reports for users who authenticate
// through the account's SSO configuration.
const zoomSsoLoginType = 100

// userVerified reports whether the user's email address is verified. Zoom
// encodes this as 0 (unverified) or 1 (verified).
func userVerified(u *connection.User) bool {
	return u.Verified != 0
}

// userSsoLinked reports whether the user signs in through the account's SSO
// configuration, derived from the login_type Zoom assigns to SSO users.
func userSsoLinked(u *connection.User) bool {
	return u.LoginType == zoomSsoLoginType
}

// resolveZoomUsers turns a list of member IDs into typed zoom.user resources.
// Each ID is resolved from the account's user index (fetched at most once per
// connection), so a role or group with N members costs one paginated user list
// rather than N per-member GetUser calls. IDs absent from the index (for
// example inactive or pending users the active-user filter excludes) fall back
// to a direct per-user lookup, and a stale or deleted ID is skipped rather than
// failing the whole list.
func resolveZoomUsers(runtime *plugin.Runtime, conn *connection.ZoomConnection, memberIds []string) ([]any, error) {
	index, err := conn.UserIndex(context.Background())
	if err != nil {
		return nil, err
	}

	all := make([]any, 0, len(memberIds))
	for _, id := range memberIds {
		if u, ok := index[id]; ok {
			res, err := newMqlZoomUser(runtime, u)
			if err != nil {
				return nil, err
			}
			all = append(all, res)
			continue
		}

		res, err := NewResource(runtime, "zoom.user", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			log.Debug().Err(err).Str("user", id).Msg("zoom> unable to resolve member")
			continue
		}
		all = append(all, res)
	}
	return all, nil
}

// users lists every user provisioned on the account.
func (r *mqlZoom) users() ([]any, error) {
	conn := r.conn()
	client := conn.Client()

	var all []any
	nextPageToken := ""
	for {
		list, err := client.ListUsers(context.Background(), usersPageSize, nextPageToken)
		if err != nil {
			return nil, err
		}
		for _, u := range list.Users {
			res, err := newMqlZoomUser(r.MqlRuntime, &u)
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if list.NextPageToken == "" {
			break
		}
		nextPageToken = list.NextPageToken
	}
	return all, nil
}

// newMqlZoomUser maps a single Zoom user to its MQL resource.
func newMqlZoomUser(runtime *plugin.Runtime, u *connection.User) (plugin.Resource, error) {
	if u.ID == "" {
		return nil, errors.New("zoom.user requires a valid id")
	}

	res, err := CreateResource(runtime, "zoom.user", map[string]*llx.RawData{
		"__id":          llx.StringData(u.ID),
		"id":            llx.StringData(u.ID),
		"email":         llx.StringData(u.Email),
		"firstName":     llx.StringData(u.FirstName),
		"lastName":      llx.StringData(u.LastName),
		"displayName":   llx.StringData(u.DisplayName),
		"type":          llx.IntData(u.Type),
		"status":        llx.StringData(u.Status),
		"verified":      llx.BoolData(userVerified(u)),
		"loginType":     llx.IntData(u.LoginType),
		"ssoLinked":     llx.BoolData(userSsoLinked(u)),
		"lastLoginTime": llx.TimeDataPtr(u.LastLoginTime),
		"createdAt":     llx.TimeDataPtr(u.CreatedAt),
		"roleId":        llx.StringData(u.RoleID),
		"groupIds":      llx.ArrayData(strToAnyList(u.GroupIDs), types.String),
	})
	if err != nil {
		return nil, err
	}

	mqlUser := res.(*mqlZoomUser)
	mqlUser.cacheRoleId = u.RoleID
	mqlUser.cacheGroupIds = u.GroupIDs
	return res, nil
}

// id returns the resource-name-prefixed natural key for this user, so
// createZoomUser has a stable fallback even on a path that omits the
// explicit "__id" argument.
func (u *mqlZoomUser) id() (string, error) {
	return "zoom.user/" + u.Id.Data, nil
}

// initZoomUser resolves a user by ID on demand, for typed references (e.g.
// zoom.role.members) and direct lookups such as zoom.user(id: "...").
func initZoomUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// fast path: the caller already provided a fully populated resource
	if len(args) > 2 {
		return args, nil, nil
	}

	idArg, ok := args["id"]
	if !ok {
		return args, nil, nil
	}
	id, ok := idArg.Value.(string)
	if !ok || id == "" {
		return nil, nil, fmt.Errorf("zoom.user requires a valid id")
	}

	conn := runtime.Connection.(*connection.ZoomConnection)
	u, err := conn.Client().GetUser(context.Background(), id)
	if err != nil {
		return nil, nil, fmt.Errorf("zoom.user with id %q not found: %w", id, err)
	}

	res, err := newMqlZoomUser(runtime, u)
	if err != nil {
		return nil, nil, err
	}
	return args, res, nil
}

// role resolves the role assigned to the user, governing their admin
// privileges.
func (u *mqlZoomUser) role() (*mqlZoomRole, error) {
	if u.cacheRoleId == "" {
		u.Role.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	r, err := NewResource(u.MqlRuntime, "zoom.role", map[string]*llx.RawData{
		"id": llx.StringData(u.cacheRoleId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlZoomRole), nil
}

// groups resolves the groups the user belongs to.
func (u *mqlZoomUser) groups() ([]any, error) {
	if len(u.cacheGroupIds) == 0 {
		return []any{}, nil
	}

	var all []any
	for _, id := range u.cacheGroupIds {
		r, err := NewResource(u.MqlRuntime, "zoom.group", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			// A stale or deleted group ID should not fail the whole user.
			log.Debug().Err(err).Str("group", id).Msg("zoom> unable to resolve group")
			continue
		}
		all = append(all, r)
	}
	return all, nil
}
