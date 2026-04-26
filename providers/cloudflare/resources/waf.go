// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"errors"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
)

func (c *mqlCloudflareZoneWafRule) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.RulesetId.Data + "/" + c.Id.Data, nil
}

// wafRules expands every ruleset attached to the zone (managed and custom)
// into the individual rules that make it up. We surface ruleset metadata on
// each rule so downstream queries can distinguish managed-by-Cloudflare rules
// from zone-defined custom rules. Empty rulesets and rate-limit/transform
// phases are kept — callers can filter by `rulesetPhase` or `rulesetKind`.
func (c *mqlCloudflareZone) wafRules() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	zone := &cloudflare.ResourceContainer{
		Identifier: c.Id.Data,
		Level:      cloudflare.ZoneRouteLevel,
	}

	rulesets, err := conn.Cf.ListRulesets(context.TODO(), zone, cloudflare.ListRulesetsParams{})
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range rulesets {
		rs := rulesets[i]

		// `ListRulesets` only returns ruleset metadata; fetch each ruleset to
		// get its rules. Skip individual rulesets that the caller can't read
		// (e.g., managed rulesets requiring extra entitlements) but surface
		// transient/unknown errors so they aren't silently swallowed.
		full, err := conn.Cf.GetRuleset(context.TODO(), zone, rs.ID)
		if err != nil {
			var notFound *cloudflare.NotFoundError
			var authN *cloudflare.AuthenticationError
			var authZ *cloudflare.AuthorizationError
			if errors.As(err, &notFound) || errors.As(err, &authN) || errors.As(err, &authZ) {
				continue
			}
			return nil, err
		}

		version := ""
		if full.Version != nil {
			version = *full.Version
		}

		for j := range full.Rules {
			r := full.Rules[j]

			ruleVersion := version
			if r.Version != nil {
				ruleVersion = *r.Version
			}

			enabled := false
			if r.Enabled != nil {
				enabled = *r.Enabled
			}

			res, err := CreateResource(c.MqlRuntime, "cloudflare.zone.wafRule", map[string]*llx.RawData{
				"__id":           llx.StringData("cloudflare.zone.wafRule@" + c.Id.Data + "/" + rs.ID + "/" + r.ID),
				"id":             llx.StringData(r.ID),
				"rulesetId":      llx.StringData(rs.ID),
				"rulesetName":    llx.StringData(rs.Name),
				"rulesetKind":    llx.StringData(rs.Kind),
				"rulesetPhase":   llx.StringData(rs.Phase),
				"action":         llx.StringData(r.Action),
				"expression":     llx.StringData(r.Expression),
				"description":    llx.StringData(r.Description),
				"ref":            llx.StringData(r.Ref),
				"enabled":        llx.BoolData(enabled),
				"scoreThreshold": llx.IntData(int64(r.ScoreThreshold)),
				"version":        llx.StringData(ruleVersion),
				"lastUpdated":    llx.TimeDataPtr(r.LastUpdated),
			})
			if err != nil {
				return nil, err
			}

			result = append(result, res)
		}
	}

	return result, nil
}
