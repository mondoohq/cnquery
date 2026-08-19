// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/zoom/connection"
)

const groupMembersPageSize = 300

// mqlZoomGroupInternal caches the group's meeting-security overrides behind a
// single lazy fetch. All the settings* accessors share this fetch, so querying
// any subset of them costs exactly one GET /groups/{id}/settings call, guarded
// by double-check locking against concurrent field reads. Keeping it lazy means
// listing groups by id/name/totalMembers issues no per-group settings call.
type mqlZoomGroupInternal struct {
	fetched  bool
	settings *connection.GroupSettings
	lock     sync.Mutex
}

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
		res, err := newMqlZoomGroup(r.MqlRuntime, &group)
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}

// newMqlZoomGroup maps a single Zoom group to its MQL resource. The group's
// meeting-security overrides are resolved lazily through the settings*
// accessors, so listing groups issues no per-group settings call.
func newMqlZoomGroup(runtime *plugin.Runtime, group *connection.Group) (plugin.Resource, error) {
	res, err := CreateResource(runtime, "zoom.group", map[string]*llx.RawData{
		"__id":         llx.StringData(group.ID),
		"id":           llx.StringData(group.ID),
		"name":         llx.StringData(group.Name),
		"totalMembers": llx.IntData(group.TotalMembers),
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

	res, err := newMqlZoomGroup(runtime, group)
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

	return resolveZoomUsers(g.MqlRuntime, conn, memberIds)
}

// fetchSettings performs the single group-settings GET this group's settings*
// fields all read from, caching the result behind double-check locking so
// concurrent field access only fetches once.
func (g *mqlZoomGroup) fetchSettings() (*connection.GroupSettings, error) {
	if g.fetched {
		return g.settings, nil
	}
	g.lock.Lock()
	defer g.lock.Unlock()
	if g.fetched {
		return g.settings, nil
	}

	conn := g.MqlRuntime.Connection.(*connection.ZoomConnection)
	settings, err := conn.Client().GetGroupSettings(context.Background(), g.Id.Data)
	if err != nil {
		return nil, err
	}
	g.settings = settings
	g.fetched = true
	return g.settings, nil
}

func (g *mqlZoomGroup) settingsWaitingRoomEnabled() (bool, error) {
	s, err := g.fetchSettings()
	if err != nil {
		return false, err
	}
	return s.MeetingSecurity.WaitingRoom, nil
}

func (g *mqlZoomGroup) settingsMeetingPasscodeRequired() (bool, error) {
	s, err := g.fetchSettings()
	if err != nil {
		return false, err
	}
	return s.MeetingSecurity.MeetingPasswordRequired, nil
}

func (g *mqlZoomGroup) settingsE2eeAvailable() (bool, error) {
	s, err := g.fetchSettings()
	if err != nil {
		return false, err
	}
	return s.MeetingSecurity.E2eeAvailable, nil
}

func (g *mqlZoomGroup) settingsOnlyAuthenticatedUsersCanJoin() (bool, error) {
	s, err := g.fetchSettings()
	if err != nil {
		return false, err
	}
	return s.MeetingSecurity.OnlyAuthenticatedCanJoinFromWebclient, nil
}
