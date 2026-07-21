// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"github.com/portainer/client-api-go/v2/pkg/models"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/portainer/connection"
)

func newMqlPortainerUser(runtime *plugin.Runtime, u *models.PortainereeUser) (*mqlPortainerUser, error) {
	res, err := CreateResource(runtime, "portainer.user", map[string]*llx.RawData{
		"__id":         llx.StringData("portainer.user/" + strconv.FormatInt(u.ID, 10)),
		"id":           llx.IntData(u.ID),
		"username":     llx.StringData(u.Username),
		"role":         llx.StringData(connection.UserRole(u.Role)),
		"useCache":     llx.BoolData(u.UseCache),
		"theme":        llx.StringData(u.UserTheme),
		"tokenIssueAt": llx.TimeDataPtr(unixTimePtr(u.TokenIssueAt)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlPortainerUser), nil
}

func (r *mqlPortainer) users() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.PortainerConnection)

	users, err := conn.Users()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(users))
	for _, u := range users {
		mqlUser, err := newMqlPortainerUser(r.MqlRuntime, u)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlUser)
	}
	return res, nil
}

// teams returns the teams the user is a member of, resolved through the
// instance team memberships.
func (r *mqlPortainerUser) teams() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.PortainerConnection)

	memberships, err := conn.TeamMemberships()
	if err != nil {
		return nil, err
	}
	teams, err := conn.Teams()
	if err != nil {
		return nil, err
	}
	teamNameByID := make(map[int64]string, len(teams))
	for _, t := range teams {
		teamNameByID[t.ID] = t.Name
	}

	res := []any{}
	for _, m := range memberships {
		if m.UserID != r.Id.Data {
			continue
		}
		mqlTeam, err := newMqlPortainerTeam(r.MqlRuntime, m.TeamID, teamNameByID[m.TeamID])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlTeam)
	}
	return res, nil
}
