// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"golang.org/x/sync/errgroup"

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

const (
	serviceAccountPageSize = 1000
	// pageFanout bounds how many pagination requests are issued concurrently
	// across the service-account pages. Eight is enough to keep wall time
	// dominated by the slowest page on large orgs without overwhelming the
	// Grafana instance with bursty traffic.
	pageFanout = 8
)

// fetchServiceAccountPage fetches a single page of service accounts and closes
// the response body before returning, avoiding FD leaks in pagination loops.
func fetchServiceAccountPage(ctx context.Context, conn *connection.GrafanaConnection, page int) (*grafanaServiceAccountsResponse, error) {
	path := fmt.Sprintf("/api/serviceaccounts/search?perpage=%d&page=%d", serviceAccountPageSize, page)
	resp, err := conn.Get(ctx, path)
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

	// Fetch page 1 first to learn totalCount, then fan out remaining pages
	// concurrently. The previous sequential loop was O(N) round trips on the
	// critical path; this is O(N/pageFanout) for the same byte volume.
	first, err := fetchServiceAccountPage(context.Background(), conn, 1)
	if err != nil {
		return nil, err
	}

	allSAs := first.ServiceAccounts
	// Compute remaining pages from totalCount. Guard against a short first page
	// (server returned all results in one call) before fanning out.
	totalPages := 1
	if first.TotalCount > serviceAccountPageSize && len(first.ServiceAccounts) >= serviceAccountPageSize {
		totalPages = (first.TotalCount + serviceAccountPageSize - 1) / serviceAccountPageSize
	}
	if totalPages > 1 {
		pages := make([][]grafanaServiceAccountJSON, totalPages-1)
		grp, ctx := errgroup.WithContext(context.Background())
		grp.SetLimit(pageFanout)
		for i := range totalPages - 1 {
			page := i + 2 // pages 2..totalPages
			grp.Go(func() error {
				result, err := fetchServiceAccountPage(ctx, conn, page)
				if err != nil {
					return err
				}
				pages[i] = result.ServiceAccounts
				return nil
			})
		}
		if err := grp.Wait(); err != nil {
			return nil, err
		}
		for _, p := range pages {
			allSAs = append(allSAs, p...)
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

		// Grafana uses "0001-01-01T00:00:00Z" as a sentinel for "no expiration",
		// which parses to time.Time{} — IsZero() catches both that and parse errors.
		hasExpiration := !expiration.IsZero()
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
