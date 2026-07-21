// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/nextdns/connection"
)

func (r *mqlNextdns) id() (string, error) {
	return "nextdns", nil
}

// profileData is one entry of the GET /profiles response.
type profileData struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
}

type profilesResponse struct {
	Data []profileData `json:"data"`
}

// fetchProfiles returns the profiles visible to the connection. When the
// connection is scoped to a single profile, that profile is fetched directly
// so we never list (or expose) profiles the connection shouldn't see.
func fetchProfiles(conn *connection.NextdnsConnection) ([]profileData, error) {
	if scoped := conn.ProfileID(); scoped != "" {
		var resp profileDetailResponse
		if err := conn.Get(context.Background(), "/profiles/"+scoped, &resp); err != nil {
			return nil, err
		}
		return []profileData{{
			ID:          scoped,
			Name:        resp.Data.Name,
			Fingerprint: resp.Data.Fingerprint,
		}}, nil
	}

	var resp profilesResponse
	if err := conn.Get(context.Background(), "/profiles", &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// profilesToResources fetches profiles for the connection and maps them to
// nextdns.profile resources.
func profilesToResources(runtime *plugin.Runtime, conn *connection.NextdnsConnection) ([]any, error) {
	profiles, err := fetchProfiles(conn)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(profiles))
	for _, p := range profiles {
		mqlProfile, err := NewResource(runtime, "nextdns.profile", map[string]*llx.RawData{
			"id":          llx.StringData(p.ID),
			"name":        llx.StringData(p.Name),
			"fingerprint": llx.StringData(p.Fingerprint),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlProfile)
	}
	return res, nil
}

func (r *mqlNextdns) profiles() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.NextdnsConnection)
	return profilesToResources(r.MqlRuntime, conn)
}

func (r *mqlNextdns) account() (*mqlNextdnsAccount, error) {
	conn := r.MqlRuntime.Connection.(*connection.NextdnsConnection)
	res, err := CreateResource(r.MqlRuntime, "nextdns.account", map[string]*llx.RawData{
		"id": llx.StringData(conn.AccountID()),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlNextdnsAccount), nil
}
