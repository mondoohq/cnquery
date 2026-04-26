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
)

// grafanaApiKeyJSON mirrors one element of /api/auth/keys.
type grafanaApiKeyJSON struct {
	ID               int    `json:"id"`
	OrgID            int    `json:"orgId"`
	Name             string `json:"name"`
	Role             string `json:"role"`
	Expiration       string `json:"expiration"`
	ServiceAccountID int    `json:"serviceAccountId"`
}

// apiKeys queries /api/auth/keys with includeExpired=true so security audits
// can flag expired but un-deleted keys. Grafana deprecated this endpoint in
// favor of service accounts, but many instances still carry legacy keys.
func (g *mqlGrafana) apiKeys() ([]any, error) {
	conn, err := grafanaConnection(g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	resp, err := conn.Get(context.Background(), "/api/auth/keys?includeExpired=true")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// 410 → api keys fully disabled; 403/404 → caller lacks admin or endpoint
	// removed. Return empty so iteration over apiKeys is safe in all cases.
	if resp.StatusCode == http.StatusGone ||
		resp.StatusCode == http.StatusNotFound ||
		resp.StatusCode == http.StatusForbidden {
		return []any{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("grafana: GET /api/auth/keys returned status %d", resp.StatusCode)
	}

	var raw []grafanaApiKeyJSON
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("grafana: decoding /api/auth/keys response: %w", err)
	}

	now := time.Now()
	list := make([]any, 0, len(raw))
	for _, k := range raw {
		expiration := parseGrafanaTime(k.Expiration)
		hasExpiration := !expiration.IsZero() && k.Expiration != ""
		isExpired := hasExpiration && expiration.Before(now)

		res, err := CreateResource(g.MqlRuntime, "grafana.apiKey", map[string]*llx.RawData{
			"id":               llx.IntData(int64(k.ID)),
			"orgId":            llx.IntData(int64(k.OrgID)),
			"name":             llx.StringData(k.Name),
			"role":             llx.StringData(k.Role),
			"expiration":       llx.TimeData(expiration),
			"hasExpiration":    llx.BoolData(hasExpiration),
			"isExpired":        llx.BoolData(isExpired),
			"serviceAccountId": llx.IntData(int64(k.ServiceAccountID)),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}

func (k *mqlGrafanaApiKey) id() (string, error) {
	return "grafana-apikey/" + strconv.FormatInt(k.Id.Data, 10), nil
}
