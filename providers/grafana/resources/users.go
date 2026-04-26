// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/grafana/connection"
)

// mqlGrafanaUserInternal holds cached fields populated lazily from
// /api/users/{id}. The detail endpoint is only called once per user; subsequent
// computed-field accesses reuse the cached response.
type mqlGrafanaUserInternal struct {
	detailFetched bool
	detail        grafanaUserDetailJSON
	detailErr     error
	detailLock    sync.Mutex

	permsFetched bool
	perms        map[string]any
	permsErr     error
	permsLock    sync.Mutex
}

// grafanaUserDetailJSON mirrors /api/users/{id}. Fields like isMFAEnabled may
// only be present on Grafana Cloud / Enterprise; they decode to zero when absent.
type grafanaUserDetailJSON struct {
	ID             int      `json:"id"`
	Email          string   `json:"email"`
	Name           string   `json:"name"`
	Login          string   `json:"login"`
	OrgID          int      `json:"orgId"`
	IsGrafanaAdmin bool     `json:"isGrafanaAdmin"`
	IsDisabled     bool     `json:"isDisabled"`
	IsExternal     bool     `json:"isExternal"`
	AuthLabels     []string `json:"authLabels"`
	IsMFAEnabled   bool     `json:"isMFAEnabled"`
	AuthModule     string   `json:"authModule"`
}

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

// fetchDetail loads the full /api/users/{id} response once and caches it on
// the resource. All computed user-detail fields share this fetch.
func (u *mqlGrafanaUser) fetchDetail() (grafanaUserDetailJSON, error) {
	if u.detailFetched {
		return u.detail, u.detailErr
	}
	u.detailLock.Lock()
	defer u.detailLock.Unlock()
	if u.detailFetched {
		return u.detail, u.detailErr
	}

	conn, err := grafanaConnection(u.MqlRuntime)
	if err != nil {
		u.detailFetched = true
		u.detailErr = err
		return u.detail, err
	}
	path := "/api/users/" + strconv.FormatInt(u.UserId.Data, 10)
	resp, err := conn.Get(context.Background(), path)
	if err != nil {
		u.detailFetched = true
		u.detailErr = err
		return u.detail, err
	}
	defer resp.Body.Close()
	// 403/404 → caller lacks server-admin permissions or user is gone. Return
	// the zero-value detail so org-admin queries don't fail outright.
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		u.detailFetched = true
		return u.detail, nil
	}
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("grafana: GET %s returned status %d", path, resp.StatusCode)
		u.detailFetched = true
		u.detailErr = err
		return u.detail, err
	}
	if err := json.NewDecoder(resp.Body).Decode(&u.detail); err != nil {
		err = fmt.Errorf("grafana: decoding %s response: %w", path, err)
		u.detailFetched = true
		u.detailErr = err
		return u.detail, err
	}
	u.detailFetched = true
	return u.detail, nil
}

func (u *mqlGrafanaUser) authModule() (string, error) {
	d, err := u.fetchDetail()
	if err != nil {
		return "", err
	}
	if d.AuthModule != "" {
		return d.AuthModule, nil
	}
	// Older Grafana versions only expose the provider in authLabels (e.g., "OAuth").
	if len(d.AuthLabels) > 0 {
		return d.AuthLabels[0], nil
	}
	return "", nil
}

func (u *mqlGrafanaUser) authLabels() ([]any, error) {
	d, err := u.fetchDetail()
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(d.AuthLabels))
	for _, l := range d.AuthLabels {
		out = append(out, l)
	}
	return out, nil
}

func (u *mqlGrafanaUser) isExternal() (bool, error) {
	d, err := u.fetchDetail()
	if err != nil {
		return false, err
	}
	if d.IsExternal {
		return true, nil
	}
	// authLabels presence is a reliable secondary signal of external auth.
	return len(d.AuthLabels) > 0, nil
}

func (u *mqlGrafanaUser) isGrafanaAdmin() (bool, error) {
	d, err := u.fetchDetail()
	if err != nil {
		return false, err
	}
	return d.IsGrafanaAdmin, nil
}

func (u *mqlGrafanaUser) isDisabled() (bool, error) {
	d, err := u.fetchDetail()
	if err != nil {
		return false, err
	}
	return d.IsDisabled, nil
}

// mfaEnabled reports the isMFAEnabled flag from /api/users/{id}. The field is
// only populated on Grafana Cloud and Enterprise; OSS Grafana has no per-user
// MFA state and will always return false.
func (u *mqlGrafanaUser) mfaEnabled() (bool, error) {
	d, err := u.fetchDetail()
	if err != nil {
		return false, err
	}
	return d.IsMFAEnabled, nil
}

// permissions returns the RBAC permissions granted to this user as a map of
// action -> []scope. Requires Grafana Enterprise/Cloud with RBAC enabled.
func (u *mqlGrafanaUser) permissions() (map[string]any, error) {
	if u.permsFetched {
		return u.perms, u.permsErr
	}
	u.permsLock.Lock()
	defer u.permsLock.Unlock()
	if u.permsFetched {
		return u.perms, u.permsErr
	}

	conn, err := grafanaConnection(u.MqlRuntime)
	if err != nil {
		u.permsFetched = true
		u.permsErr = err
		return nil, err
	}
	path := "/api/access-control/users/" + strconv.FormatInt(u.UserId.Data, 10) + "/permissions"
	resp, err := conn.Get(context.Background(), path)
	if err != nil {
		u.permsFetched = true
		u.permsErr = err
		return nil, err
	}
	defer resp.Body.Close()

	// 403/404 → endpoint not available (OSS), RBAC disabled, or caller lacks
	// access. Return empty map so org-admin queries don't fail.
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		u.permsFetched = true
		u.perms = map[string]any{}
		return u.perms, nil
	}
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("grafana: GET %s returned status %d", path, resp.StatusCode)
		u.permsFetched = true
		u.permsErr = err
		return nil, err
	}

	// Grafana returns permissions as a map of action -> []scope.
	var raw map[string][]string
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		err = fmt.Errorf("grafana: decoding %s response: %w", path, err)
		u.permsFetched = true
		u.permsErr = err
		return nil, err
	}

	out := make(map[string]any, len(raw))
	for k, scopes := range raw {
		anyScopes := make([]any, len(scopes))
		for i, s := range scopes {
			anyScopes[i] = s
		}
		out[k] = anyScopes
	}
	u.perms = out
	u.permsFetched = true
	return out, nil
}
