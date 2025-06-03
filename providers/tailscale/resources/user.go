// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/tailscale/connection"
)

func (r *mqlTailscaleUser) id() (string, error) {
	return "tailscale/user/" + r.Id.Data, nil
}

func initTailscaleUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := args["id"]
	if !ok {
		return nil, nil, errors.New("missing required argument 'id'")
	}

	conn := runtime.Connection.(*connection.TailscaleConnection)
	user, err := conn.Client().Users().Get(context.Background(), id.Value.(string))
	if err != nil {
		return nil, nil, err
	}

	resource, err := createTailscaleUserResource(runtime, user)
	if err != nil {
		return nil, nil, err
	}

	return args, resource.(*mqlTailscaleUser), nil
}

func createTailscaleUserResource(runtime *plugin.Runtime, user *tsclient.User) (plugin.Resource, error) {
	return CreateResource(runtime, "tailscale.user", map[string]*llx.RawData{
		"id":            llx.StringData(user.ID),
		"displayName":   llx.StringData(user.DisplayName),
		"loginName":     llx.StringData(user.LoginName),
		"profilePicUrl": llx.StringData(user.ProfilePicURL),
		"tailnetId":     llx.StringData(user.TailnetID),
		"type":          llx.StringData(string(user.Type)),
		"role":          llx.StringData(string(user.Role)),
		"status":        llx.StringData(string(user.Status)),
		"deviceCount":   llx.IntData(user.DeviceCount),
		"createdAt":     llx.TimeData(user.Created),
		"lastSeenAt":    llx.TimeData(user.LastSeen),
	})
}
