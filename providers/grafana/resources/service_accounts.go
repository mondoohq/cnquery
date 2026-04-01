// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/grafana/connection"
)

// grafanaServiceAccountJSON mirrors one element of the /api/serviceaccounts/search response.
type grafanaServiceAccountJSON struct {
	ID         int    `json:"id"`
	OrgID      int    `json:"orgId"`
	Name       string `json:"name"`
	Login      string `json:"login"`
	Role       string `json:"role"`
	IsDisabled bool   `json:"isDisabled"`
	IsExternal bool   `json:"isExternal"`
}

// grafanaServiceAccountsResponse wraps the paginated service accounts endpoint.
type grafanaServiceAccountsResponse struct {
	TotalCount      int                         `json:"totalCount"`
	ServiceAccounts []grafanaServiceAccountJSON `json:"serviceAccounts"`
	Page            int                         `json:"page"`
	PerPage         int                         `json:"perPage"`
}

// grafanaTokenJSON mirrors one element of the /api/serviceaccounts/{id}/tokens response.
type grafanaTokenJSON struct {
	ID                    int     `json:"id"`
	Name                  string  `json:"name"`
	Created               string  `json:"created"`
	Expiration            string  `json:"expiration"`
	HasExpired            bool    `json:"hasExpired"`
	SecondsTillExpiration float64 `json:"secondsUntilExpiration"`
	IsRevoked             bool    `json:"isRevoked"`
}

const serviceAccountPageSize = 1000

// fetchServiceAccountPage fetches a single page of service accounts and closes
// the response body before returning, avoiding FD leaks in pagination loops.
func fetchServiceAccountPage(conn *connection.GrafanaConnection, page int) (*grafanaServiceAccountsResponse, error) {
	path := fmt.Sprintf("/api/serviceaccounts/search?perpage=%d&page=%d", serviceAccountPageSize, page)
	resp, err := conn.Get(context.Background(), path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("grafana: GET /api/serviceaccounts/search returned status %d", resp.StatusCode)
	}

	var result grafanaServiceAccountsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("grafana: decoding /api/serviceaccounts/search response: %w", err)
	}
	return &result, nil
}

func (g *mqlGrafana) serviceAccounts() ([]interface{}, error) {
	conn, err := grafanaConnection(g.MqlRuntime)
	if err != nil {
		return nil, err
	}

	// Paginate through all service accounts. The API returns at most perpage
	// results per request; we stop when we've collected totalCount or the
	// page returns fewer results than requested.
	var allSAs []grafanaServiceAccountJSON
	for page := 1; ; page++ {
		result, err := fetchServiceAccountPage(conn, page)
		if err != nil {
			return nil, err
		}

		allSAs = append(allSAs, result.ServiceAccounts...)

		// Stop when we've fetched everything or the page was not full.
		if len(allSAs) >= result.TotalCount || len(result.ServiceAccounts) < serviceAccountPageSize {
			break
		}
	}

	list := make([]interface{}, 0, len(allSAs))
	for _, sa := range allSAs {
		res, err := CreateResource(g.MqlRuntime, "grafana.serviceAccount", map[string]*llx.RawData{
			"id":         llx.IntData(int64(sa.ID)),
			"orgId":      llx.IntData(int64(sa.OrgID)),
			"name":       llx.StringData(sa.Name),
			"login":      llx.StringData(sa.Login),
			"role":       llx.StringData(sa.Role),
			"isDisabled": llx.BoolData(sa.IsDisabled),
			"isExternal": llx.BoolData(sa.IsExternal),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}

func (g *mqlGrafanaServiceAccount) id() (string, error) {
	return "grafana-sa/" + strconv.FormatInt(g.Id.Data, 10), nil
}

func (g *mqlGrafanaServiceAccount) tokens() ([]interface{}, error) {
	conn, err := grafanaConnection(g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	saID := g.Id.Data
	path := "/api/serviceaccounts/" + strconv.FormatInt(saID, 10) + "/tokens"

	resp, err := conn.Get(context.Background(), path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("grafana: GET %s returned status %d", path, resp.StatusCode)
	}

	var raw []grafanaTokenJSON
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("grafana: decoding %s response: %w", path, err)
	}

	list := make([]interface{}, 0, len(raw))
	for _, tok := range raw {
		created := parseGrafanaTime(tok.Created)
		expiration := parseGrafanaTime(tok.Expiration)

		// Grafana uses "0001-01-01T00:00:00Z" as a sentinel for "no expiration".
		// Treat any zero-like expiration as no expiration.
		zeroGrafana := parseGrafanaTime("0001-01-01T00:00:00Z")
		hasExpiration := !expiration.IsZero() && !expiration.Equal(zeroGrafana)
		secondsUntilExp := tok.SecondsTillExpiration
		if !hasExpiration {
			secondsUntilExp = 0
		}

		res, err := CreateResource(g.MqlRuntime, "grafana.serviceAccountToken", map[string]*llx.RawData{
			"id":                     llx.IntData(int64(tok.ID)),
			"serviceAccountId":       llx.IntData(saID),
			"name":                   llx.StringData(tok.Name),
			"created":                llx.TimeData(created),
			"expiration":             llx.TimeData(expiration),
			"hasExpiration":          llx.BoolData(hasExpiration),
			"secondsUntilExpiration": llx.FloatData(secondsUntilExp),
			"isExpired":              llx.BoolData(tok.HasExpired),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}

func (t *mqlGrafanaServiceAccountToken) id() (string, error) {
	return "grafana-sa-token/" +
		strconv.FormatInt(t.ServiceAccountId.Data, 10) + "/" +
		strconv.FormatInt(t.Id.Data, 10), nil
}
