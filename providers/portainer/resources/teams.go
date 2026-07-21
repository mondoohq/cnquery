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

func newMqlPortainerTeam(runtime *plugin.Runtime, id int64, name string) (*mqlPortainerTeam, error) {
	res, err := CreateResource(runtime, "portainer.team", map[string]*llx.RawData{
		"__id": llx.StringData("portainer.team/" + strconv.FormatInt(id, 10)),
		"id":   llx.IntData(id),
		"name": llx.StringData(name),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlPortainerTeam), nil
}

func (r *mqlPortainer) teams() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.PortainerConnection)

	teams, err := conn.Teams()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(teams))
	for _, t := range teams {
		mqlTeam, err := newMqlPortainerTeam(r.MqlRuntime, t.ID, t.Name)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlTeam)
	}
	return res, nil
}

// members returns the users that belong to the team, resolved through the
// instance team memberships.
func (r *mqlPortainerTeam) members() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.PortainerConnection)

	memberships, err := conn.TeamMemberships()
	if err != nil {
		return nil, err
	}
	users, err := conn.Users()
	if err != nil {
		return nil, err
	}
	userByID := make(map[int64]*models.PortainereeUser, len(users))
	for _, u := range users {
		userByID[u.ID] = u
	}

	res := []any{}
	for _, m := range memberships {
		if m.TeamID != r.Id.Data {
			continue
		}
		u, ok := userByID[m.UserID]
		if !ok {
			continue
		}
		mqlUser, err := newMqlPortainerUser(r.MqlRuntime, u)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlUser)
	}
	return res, nil
}
