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
)

const groupMembersPageSize = 300

// groups lists every user group defined on the account, each carrying its
// own meeting-security overrides.
func (r *mqlZoom) groups() ([]any, error) {
	conn := r.conn()
	client := conn.Client()

	list, err := client.ListGroups(context.Background())
	if err != nil {
		return nil, err
	}

	var all []any
	for _, group := range list.Groups {
		res, err := newMqlZoomGroup(r.MqlRuntime, conn, &group)
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}

// newMqlZoomGroup maps a single Zoom group, along with its meeting-security
// overrides, to its MQL resource. Zoom has no batch groups-settings
// endpoint, so this issues one extra GET per group.
func newMqlZoomGroup(runtime *plugin.Runtime, conn *connection.ZoomConnection, group *connection.Group) (plugin.Resource, error) {
	settings, err := conn.Client().GetGroupSettings(context.Background(), group.ID)
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(runtime, "zoom.group", map[string]*llx.RawData{
		"__id":                                  llx.StringData(group.ID),
		"id":                                    llx.StringData(group.ID),
		"name":                                  llx.StringData(group.Name),
		"totalMembers":                          llx.IntData(group.TotalMembers),
		"settingsWaitingRoomEnabled":            llx.BoolData(settings.MeetingSecurity.WaitingRoom),
		"settingsMeetingPasscodeRequired":       llx.BoolData(settings.MeetingSecurity.MeetingPasswordRequired),
		"settingsE2eeAvailable":                 llx.BoolData(settings.MeetingSecurity.E2eeAvailable),
		"settingsOnlyAuthenticatedUsersCanJoin": llx.BoolData(settings.MeetingSecurity.OnlyAuthenticatedCanJoin),
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// id returns the resource-name-prefixed natural key for this group, so
// createZoomGroup has a stable fallback even on a path that omits the
// explicit "__id" argument.
func (g *mqlZoomGroup) id() (string, error) {
	return "zoom.group/" + g.Id.Data, nil
}

// initZoomGroup resolves a group by ID on demand.
func initZoomGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	idArg, ok := args["id"]
	if !ok {
		return args, nil, nil
	}
	id, ok := idArg.Value.(string)
	if !ok || id == "" {
		return nil, nil, fmt.Errorf("zoom.group requires a valid id")
	}

	conn := runtime.Connection.(*connection.ZoomConnection)
	group, err := conn.Client().GetGroup(context.Background(), id)
	if err != nil {
		return nil, nil, fmt.Errorf("zoom.group with id %q not found: %w", id, err)
	}

	res, err := newMqlZoomGroup(runtime, conn, group)
	if err != nil {
		return nil, nil, err
	}
	return args, res, nil
}

// members resolves the users belonging to this group.
func (g *mqlZoomGroup) members() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.ZoomConnection)
	client := conn.Client()

	var memberIds []string
	nextPageToken := ""
	for {
		list, err := client.ListGroupMembers(context.Background(), g.Id.Data, groupMembersPageSize, nextPageToken)
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
		res, err := NewResource(g.MqlRuntime, "zoom.user", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			// A stale or deleted user ID should not fail the whole group.
			log.Debug().Err(err).Str("user", id).Msg("zoom> unable to resolve group member")
			continue
		}
		all = append(all, res)
	}
	return all, nil
}
