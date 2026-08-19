// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"

	"github.com/digitalocean/godo"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/digitalocean/connection"
)

// godo does not wrap the Organizations endpoints, so the request goes through
// the client's own NewRequest/Do. That keeps the token, the rate limiter and
// the error decoding the rest of the provider relies on, rather than standing
// up a second HTTP client beside them.
const organizationTeamsPath = "v2/organizations/teams"

type organizationTeam struct {
	ID                   uint64 `json:"id"`
	UUID                 string `json:"uuid"`
	Name                 string `json:"name"`
	Email                string `json:"email"`
	MemberCount          uint64 `json:"member_count"`
	Status               string `json:"status"`
	JoinedOrganizationAt string `json:"joined_organization_at"`
}

type organizationTeamsRoot struct {
	Teams []organizationTeam `json:"teams"`
}

// noOrganizationContext reports the answers that mean this token has no
// organization to enumerate, rather than that the read failed.
//
// The endpoint "must be called in an organization context", so a token scoped
// to a plain account is not a failure to report: it is an account with no
// teams. A 401 is deliberately excluded, since a rejected token is a real
// problem and must not be laundered into an empty list.
func noOrganizationContext(err error) bool {
	var er *godo.ErrorResponse
	if !errors.As(err, &er) || er.Response == nil {
		return false
	}
	switch er.Response.StatusCode {
	case http.StatusForbidden, http.StatusNotFound, http.StatusPreconditionFailed:
		return true
	}
	return false
}

func (r *mqlDigitaloceanTeam) id() (string, error) {
	return resourceID("digitalocean.team", r.Uuid.Data)
}

func (r *mqlDigitalocean) teams() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()
	ctx := context.Background()

	req, err := client.NewRequest(ctx, http.MethodGet, organizationTeamsPath, nil)
	if err != nil {
		return nil, err
	}

	root := new(organizationTeamsRoot)
	if _, err := client.Do(ctx, req, root); err != nil {
		if noOrganizationContext(err) {
			log.Debug().Err(err).Msg("digitalocean> token is not scoped to an organization; reporting no teams")
			return []interface{}{}, nil
		}
		return nil, err
	}

	all := make([]interface{}, 0, len(root.Teams))
	for i := range root.Teams {
		t := root.Teams[i]
		res, err := CreateResource(r.MqlRuntime, "digitalocean.team", map[string]*llx.RawData{
			"id":                   llx.IntData(int64(t.ID)),
			"uuid":                 llx.StringData(t.UUID),
			"name":                 llx.StringData(t.Name),
			"email":                llx.StringData(t.Email),
			"memberCount":          llx.IntData(int64(t.MemberCount)),
			"status":               llx.StringData(t.Status),
			"joinedOrganizationAt": llx.TimeDataPtr(parseDoTime(t.JoinedOrganizationAt)),
		})
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}
