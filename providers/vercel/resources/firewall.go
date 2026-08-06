// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"strings"

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
	CRS             any                  `json:"crs"`
	BotIDEnabled    *bool                `json:"botIdEnabled"`
	LogHeaders      logHeaders           `json:"logHeaders"`
	Version         *int64               `json:"version"`
	UpdatedAt       flexTime             `json:"updatedAt"`
	Rules           []firewallRuleRecord `json:"rules"`
	IPs             []firewallIPRecord   `json:"ips"`
}

// logHeaders decodes the recorded-header setting, which Vercel returns either as
// a list of header names or as the string "*" meaning every header.
type logHeaders struct {
	values []string
}

func (l *logHeaders) UnmarshalJSON(b []byte) error {
	if s := strings.TrimSpace(string(b)); s == "null" || s == "" {
		return nil
	}
	var list []string
	if err := json.Unmarshal(b, &list); err == nil {
		l.values = list
		return nil
	}
	var single string
	if err := json.Unmarshal(b, &single); err != nil {
		return err
	}
	l.values = []string{single}
	return nil
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
	crs := cfg.CRS
	if crs == nil {
		crs = map[string]any{}
	}

	res, err := CreateResource(p.MqlRuntime, "vercel.firewall", map[string]*llx.RawData{
		"__id":            llx.StringData(p.Id.Data + "/firewall"),
		"enabled":         llx.BoolData(cfg.FirewallEnabled),
		"managedRulesets": llx.DictData(managed),
		"coreRuleSet":     llx.DictData(crs),
		"botIdEnabled":    llx.BoolData(cfg.BotIDEnabled != nil && *cfg.BotIDEnabled),
		"logHeaders":      llx.ArrayData(strSliceToAny(cfg.LogHeaders.values), types.String),
		"configVersion":   llx.IntData(intPtrOrZero(cfg.Version)),
		"updatedAt":       llx.TimeDataPtr(cfg.UpdatedAt.Time()),
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

// --- system bypass --------------------------------------------------------

type bypassRuleRecord struct {
	ID            string   `json:"Id"`
	IP            string   `json:"Ip"`
	Domain        string   `json:"Domain"`
	Action        string   `json:"Action"`
	IsProjectRule *bool    `json:"IsProjectRule"`
	Note          *string  `json:"Note"`
	ActorID       string   `json:"ActorId"`
	ExpiresAt     flexTime `json:"ExpiresAt"`
	CreatedAt     flexTime `json:"CreatedAt"`
	UpdatedAt     flexTime `json:"UpdatedAt"`
}

// bypassRules lists the entries that skip firewall evaluation for a source
// address and hostname. The endpoint uses PascalCase keys and an offset cursor
// of its own rather than the shared pagination envelope, so it pages here.
func (f *mqlVercelFirewall) bypassRules() ([]any, error) {
	conn := f.MqlRuntime.Connection.(*connection.VercelConnection)

	records, err := f.fetchBypassRules(conn)
	if err != nil {
		// System bypass is gated with the rest of the configurable firewall;
		// treat an absent or forbidden list as empty rather than failing.
		if connection.IsForbidden(err) || connection.IsNotFound(err) {
			return []any{}, nil
		}
		return nil, err
	}

	res := []any{}
	for i := range records {
		rec := records[i]
		rule, err := CreateResource(f.MqlRuntime, "vercel.firewall.bypassRule", map[string]*llx.RawData{
			"id":            llx.StringData(rec.ID),
			"ip":            llx.StringData(rec.IP),
			"domain":        llx.StringData(rec.Domain),
			"action":        llx.StringData(rec.Action),
			"projectScoped": llx.BoolData(rec.IsProjectRule != nil && *rec.IsProjectRule),
			"note":          llx.StringDataPtr(rec.Note),
			"actorId":       llx.StringData(rec.ActorID),
			"expiresAt":     llx.TimeDataPtr(rec.ExpiresAt.Time()),
			"createdAt":     llx.TimeDataPtr(rec.CreatedAt.Time()),
			"updatedAt":     llx.TimeDataPtr(rec.UpdatedAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, rule)
	}
	return res, nil
}

// fetchBypassRules walks the offset cursor, which carries the id of the last
// item returned rather than a page number.
func (f *mqlVercelFirewall) fetchBypassRules(conn *connection.VercelConnection) ([]bypassRuleRecord, error) {
	query := connection.TeamQuery(f.teamID)
	query.Set("projectId", f.projectID)
	query.Set("limit", "256")

	var all []bypassRuleRecord
	for {
		var page struct {
			Result     []bypassRuleRecord `json:"result"`
			Pagination *struct {
				ID string `json:"Id"`
			} `json:"pagination"`
		}
		if err := conn.Get(context.Background(), "/v1/security/firewall/bypass", query, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Result...)

		if page.Pagination == nil || page.Pagination.ID == "" || query.Get("offset") == page.Pagination.ID {
			break
		}
		query.Set("offset", page.Pagination.ID)
	}
	return all, nil
}

func (c *mqlVercelFirewallBypassRule) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

// attackAnomalies reports traffic Vercel flagged as an attack over the last
// day. The endpoint returns an empty object rather than an empty list when
// nothing was flagged.
func (f *mqlVercelFirewall) attackAnomalies() ([]any, error) {
	conn := f.MqlRuntime.Connection.(*connection.VercelConnection)

	query := connection.TeamQuery(f.teamID)
	query.Set("projectId", f.projectID)

	var resp struct {
		Anomalies []map[string]any `json:"anomalies"`
	}
	if err := conn.Get(context.Background(), "/v1/security/firewall/attack-status", query, &resp); err != nil {
		if connection.IsForbidden(err) || connection.IsNotFound(err) {
			return []any{}, nil
		}
		return nil, err
	}
	return dictSliceToAny(resp.Anomalies), nil
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
