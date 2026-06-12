// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"fmt"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/rulesets"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
)

func (c *mqlCloudflareZoneWafRule) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.RulesetId.Data + "/" + c.Id.Data, nil
}

// rulesetDetail mirrors the ruleset-detail response. cloudflare-go v6's typed
// rule struct doesn't expose the per-rule score_threshold we surface, so we read
// the ruleset detail via the client's generic Get and decode it ourselves.
type rulesetDetail struct {
	Result struct {
		Version string `json:"version"`
		Rules   []struct {
			ID             string    `json:"id"`
			Action         string    `json:"action"`
			Expression     string    `json:"expression"`
			Description    string    `json:"description"`
			Ref            string    `json:"ref"`
			Enabled        bool      `json:"enabled"`
			ScoreThreshold int64     `json:"score_threshold"`
			Version        string    `json:"version"`
			LastUpdated    time.Time `json:"last_updated"`
		} `json:"rules"`
	} `json:"result"`
}

// wafRules expands every ruleset attached to the zone (managed and custom)
// into the individual rules that make it up. We surface ruleset metadata on
// each rule so downstream queries can distinguish managed-by-Cloudflare rules
// from zone-defined custom rules. Empty rulesets and rate-limit/transform
// phases are kept — callers can filter by `rulesetPhase` or `rulesetKind`.
func (c *mqlCloudflareZone) wafRules() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	var result []any
	iter := conn.Cf.Rulesets.ListAutoPaging(context.TODO(), rulesets.RulesetListParams{
		ZoneID: cloudflare.F(c.Id.Data),
	})
	for iter.Next() {
		rs := iter.Current()

		// The list only returns ruleset metadata; fetch each ruleset to get its
		// rules. Skip individual rulesets that the caller can't read (e.g.,
		// managed rulesets requiring extra entitlements) but surface
		// transient/unknown errors so they aren't silently swallowed.
		var full rulesetDetail
		uri := fmt.Sprintf("zones/%s/rulesets/%s", c.Id.Data, rs.ID)
		if err := conn.Cf.Get(context.TODO(), uri, nil, &full); err != nil {
			if isUnavailable(err) {
				continue
			}
			return nil, err
		}

		for j := range full.Result.Rules {
			r := full.Result.Rules[j]

			ruleVersion := full.Result.Version
			if r.Version != "" {
				ruleVersion = r.Version
			}

			res, err := CreateResource(c.MqlRuntime, "cloudflare.zone.wafRule", map[string]*llx.RawData{
				"__id":           llx.StringData("cloudflare.zone.wafRule@" + c.Id.Data + "/" + rs.ID + "/" + r.ID),
				"id":             llx.StringData(r.ID),
				"rulesetId":      llx.StringData(rs.ID),
				"rulesetName":    llx.StringData(rs.Name),
				"rulesetKind":    llx.StringData(string(rs.Kind)),
				"rulesetPhase":   llx.StringData(string(rs.Phase)),
				"action":         llx.StringData(r.Action),
				"expression":     llx.StringData(r.Expression),
				"description":    llx.StringData(r.Description),
				"ref":            llx.StringData(r.Ref),
				"enabled":        llx.BoolData(r.Enabled),
				"scoreThreshold": llx.IntData(r.ScoreThreshold),
				"version":        llx.StringData(ruleVersion),
				"lastUpdated":    timeOrNil(r.LastUpdated),
			})
			if err != nil {
				return nil, err
			}

			result = append(result, res)
		}
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
