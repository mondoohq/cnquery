// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/zoom/connection"
	"go.mondoo.com/mql/v13/types"
)

const roleMembersPageSize = 300

// roles lists every role that can be assigned to users on the account.
func (r *mqlZoom) roles() ([]any, error) {
	conn := r.conn()
	client := conn.Client()

	list, err := client.ListRoles(context.Background())
	if err != nil {
		return nil, err
	}

	var all []any
	for _, role := range list.Roles {
		res, err := newMqlZoomRole(r.MqlRuntime, &role)
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}

// newMqlZoomRole maps a single Zoom role to its MQL resource.
func newMqlZoomRole(runtime *plugin.Runtime, role *connection.Role) (plugin.Resource, error) {
	res, err := CreateResource(runtime, "zoom.role", map[string]*llx.RawData{
		"__id":         llx.StringData(role.ID),
		"id":           llx.StringData(role.ID),
		"name":         llx.StringData(role.Name),
		"description":  llx.StringData(role.Description),
		"totalMembers": llx.IntData(role.TotalMembers),
		"privileges":   llx.ArrayData(strToAnyList(role.Privileges), types.String),
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// id returns the resource-name-prefixed natural key for this role, so
// createZoomRole has a stable fallback even on a path that omits the
// explicit "__id" argument.
func (r *mqlZoomRole) id() (string, error) {
	return "zoom.role/" + r.Id.Data, nil
}

// initZoomRole resolves a role by ID on demand.
func initZoomRole(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	idArg, ok := args["id"]
	if !ok {
		return args, nil, nil
	}
	id, ok := idArg.Value.(string)
	if !ok || id == "" {
		return nil, nil, fmt.Errorf("zoom.role requires a valid id")
	}

	conn := runtime.Connection.(*connection.ZoomConnection)
	role, err := conn.Client().GetRole(context.Background(), id)
	if err != nil {
		return nil, nil, fmt.Errorf("zoom.role with id %q not found: %w", id, err)
	}

	res, err := newMqlZoomRole(runtime, role)
	if err != nil {
		return nil, nil, err
	}
	return args, res, nil
}

// members resolves the users currently assigned this role, so admin-equivalent
// access can be audited directly from the role.
func (r *mqlZoomRole) members() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.ZoomConnection)
	client := conn.Client()

	var memberIds []string
	nextPageToken := ""
	for {
		list, err := client.ListRoleMembers(context.Background(), r.Id.Data, roleMembersPageSize, nextPageToken)
		if err != nil {
			return nil, err
		}
		for _, m := range list.Members {
			memberIds = append(memberIds, m.ID)
		}
		if list.NextPageToken == "" {
			break
		}
		nextPageToken = list.NextPageToken
	}

	var all []any
	for _, id := range memberIds {
		res, err := NewResource(r.MqlRuntime, "zoom.user", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			// A stale or deleted user ID should not fail the whole role.
			log.Debug().Err(err).Str("user", id).Msg("zoom> unable to resolve role member")
			continue
		}
		all = append(all, res)
	}
	return all, nil
}
