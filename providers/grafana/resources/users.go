// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/grafana/connection"
)

// grafanaConnection is a helper that type-asserts the runtime connection to
// *connection.GrafanaConnection. All resource methods must use this instead of
// raw type assertions to get a clear error on misconfigured runtimes.
func grafanaConnection(runtime *plugin.Runtime) (*connection.GrafanaConnection, error) {
	conn, ok := runtime.Connection.(*connection.GrafanaConnection)
	if !ok {
		return nil, fmt.Errorf("grafana: unexpected connection type %T", runtime.Connection)
	}
	return conn, nil
}

// parseGrafanaTime parses a Grafana RFC3339 timestamp, returning the zero
// time.Time on any parse error (e.g. missing or invalid value).
func parseGrafanaTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// grafanaOrgJSON mirrors the /api/org response.
type grafanaOrgJSON struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// grafanaOrgUserJSON mirrors one element of the /api/org/users response.
type grafanaOrgUserJSON struct {
	OrgID         int    `json:"orgId"`
	UserID        int    `json:"userId"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Login         string `json:"login"`
	Role          string `json:"role"`
	LastSeenAt    string `json:"lastSeenAt"`
	LastSeenAtAge string `json:"lastSeenAtAge"`
}

// initGrafanaOrganization delegates to the parent grafana resource when the
// organization is accessed directly (e.g. grafana.organization.name). Without
// this, NewResource creates an empty stub with no field data.
func initGrafanaOrganization(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	grafanaRes, err := NewResource(runtime, "grafana", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	org := grafanaRes.(*mqlGrafana).GetOrganization()
	if org.Error != nil {
		return nil, nil, org.Error
	}
	return nil, org.Data, nil
}

func (g *mqlGrafana) organization() (*mqlGrafanaOrganization, error) {
	conn, err := grafanaConnection(g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	resp, err := conn.Get(context.Background(), "/api/org")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("grafana: GET /api/org returned status %d", resp.StatusCode)
	}

	var org grafanaOrgJSON
	if err := json.NewDecoder(resp.Body).Decode(&org); err != nil {
		return nil, fmt.Errorf("grafana: decoding /api/org response: %w", err)
	}

	res, err := CreateResource(g.MqlRuntime, "grafana.organization", map[string]*llx.RawData{
		"id":   llx.IntData(int64(org.ID)),
		"name": llx.StringData(org.Name),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGrafanaOrganization), nil
}

func (g *mqlGrafana) users() ([]interface{}, error) {
	conn, err := grafanaConnection(g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	resp, err := conn.Get(context.Background(), "/api/org/users")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("grafana: GET /api/org/users returned status %d", resp.StatusCode)
	}

	var raw []grafanaOrgUserJSON
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("grafana: decoding /api/org/users response: %w", err)
	}

	list := make([]interface{}, 0, len(raw))
	for _, u := range raw {
		res, err := CreateResource(g.MqlRuntime, "grafana.user", map[string]*llx.RawData{
			"userId":        llx.IntData(int64(u.UserID)),
			"orgId":         llx.IntData(int64(u.OrgID)),
			"email":         llx.StringData(u.Email),
			"name":          llx.StringData(u.Name),
			"login":         llx.StringData(u.Login),
			"role":          llx.StringData(u.Role),
			"lastSeenAt":    llx.TimeData(parseGrafanaTime(u.LastSeenAt)),
			"lastSeenAtAge": llx.StringData(u.LastSeenAtAge),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}

func (o *mqlGrafanaOrganization) id() (string, error) {
	return "grafana-org/" + strconv.FormatInt(o.Id.Data, 10), nil
}

func (u *mqlGrafanaUser) id() (string, error) {
	return "grafana-user/" + strconv.FormatInt(u.UserId.Data, 10), nil
}
