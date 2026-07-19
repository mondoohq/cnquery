// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/vercel/connection"
	"go.mondoo.com/mql/v13/types"
)

// mqlVercelFirewallInternal caches the firewall scope and the rules fetched with
// the active configuration so rules() and ipRules() avoid a second API call.
type mqlVercelFirewallInternal struct {
	teamID     string
	projectID  string
	cacheRules []firewallRuleRecord
	cacheIPs   []firewallIPRecord
}

type firewallConfig struct {
	FirewallEnabled bool                 `json:"firewallEnabled"`
	ManagedRules    any                  `json:"managedRules"`
	Rules           []firewallRuleRecord `json:"rules"`
	IPs             []firewallIPRecord   `json:"ips"`
}

type firewallRuleRecord struct {
	ID             string           `json:"id"`
	Name           string           `json:"name"`
	Description    *string          `json:"description"`
	Active         bool             `json:"active"`
	Action         any              `json:"action"`
	ConditionGroup []map[string]any `json:"conditionGroup"`
}

type firewallIPRecord struct {
	ID       string  `json:"id"`
	Hostname *string `json:"hostname"`
	IP       string  `json:"ip"`
	Action   string  `json:"action"`
	Notes    *string `json:"notes"`
}

// firewallRuleAction reduces a custom rule action, which may be a plain string
// or a nested mitigation object, to its action verb.
func firewallRuleAction(a any) string {
	switch v := a.(type) {
	case string:
		return v
	case map[string]any:
		if m, ok := v["mitigate"].(map[string]any); ok {
			if s, ok := m["action"].(string); ok {
				return s
			}
		}
		if s, ok := v["action"].(string); ok {
			return s
		}
	}
	return ""
}

func (p *mqlVercelProject) firewall() (*mqlVercelFirewall, error) {
	conn := p.MqlRuntime.Connection.(*connection.VercelConnection)
	query := connection.TeamQuery(p.teamID)
	query.Set("projectId", p.Id.Data)

	var cfg firewallConfig
	if err := conn.Get(context.Background(), "/v1/security/firewall/config/active", query, &cfg); err != nil {
		// The configurable WAF is an Enterprise feature; treat an absent or
		// forbidden configuration as null rather than failing the scan.
		if connection.IsForbidden(err) || connection.IsNotFound(err) {
			p.Firewall.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}

	managed := cfg.ManagedRules
	if managed == nil {
		managed = map[string]any{}
	}

	res, err := CreateResource(p.MqlRuntime, "vercel.firewall", map[string]*llx.RawData{
		"__id":            llx.StringData(p.Id.Data + "/firewall"),
		"enabled":         llx.BoolData(cfg.FirewallEnabled),
		"managedRulesets": llx.DictData(managed),
	})
	if err != nil {
		return nil, err
	}

	fw := res.(*mqlVercelFirewall)
	fw.teamID = p.teamID
	fw.projectID = p.Id.Data
	fw.cacheRules = cfg.Rules
	fw.cacheIPs = cfg.IPs
	return fw, nil
}

func (f *mqlVercelFirewall) rules() ([]any, error) {
	var res []any
	for i := range f.cacheRules {
		rec := f.cacheRules[i]
		rule, err := CreateResource(f.MqlRuntime, "vercel.firewall.rule", map[string]*llx.RawData{
			"id":             llx.StringData(rec.ID),
			"name":           llx.StringData(rec.Name),
			"description":    llx.StringDataPtr(rec.Description),
			"active":         llx.BoolData(rec.Active),
			"action":         llx.StringData(firewallRuleAction(rec.Action)),
			"conditionGroup": llx.ArrayData(dictSliceToAny(rec.ConditionGroup), types.Dict),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, rule)
	}
	return res, nil
}

func (c *mqlVercelFirewallRule) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

func (f *mqlVercelFirewall) ipRules() ([]any, error) {
	var res []any
	for i := range f.cacheIPs {
		rec := f.cacheIPs[i]
		rule, err := CreateResource(f.MqlRuntime, "vercel.firewall.ipRule", map[string]*llx.RawData{
			"id":       llx.StringData(rec.ID),
			"ip":       llx.StringData(rec.IP),
			"hostname": llx.StringDataPtr(rec.Hostname),
			"action":   llx.StringData(rec.Action),
			"notes":    llx.StringDataPtr(rec.Notes),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, rule)
	}
	return res, nil
}

func (c *mqlVercelFirewallIpRule) id() (string, error) {
	return c.Id.Data, c.Id.Error
}
