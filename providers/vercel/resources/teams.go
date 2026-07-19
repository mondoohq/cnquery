// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/vercel/connection"
	"go.mondoo.com/mql/v13/types"
)

type teamRecord struct {
	ID                                 string         `json:"id"`
	Slug                               string         `json:"slug"`
	Name                               string         `json:"name"`
	Avatar                             string         `json:"avatar"`
	CreatedAt                          flexTime       `json:"createdAt"`
	Saml                               *samlRecord    `json:"saml"`
	SensitiveEnvironmentVariablePolicy *string        `json:"sensitiveEnvironmentVariablePolicy"`
	RemoteCaching                      *remoteCaching `json:"remoteCaching"`
}

type samlRecord struct {
	Enforced bool              `json:"enforced"`
	Roles    map[string]string `json:"roles"`
}

type remoteCaching struct {
	Enabled *bool `json:"enabled"`
}

func newVercelTeam(runtime *plugin.Runtime, rec *teamRecord) (*mqlVercelTeam, error) {
	samlEnforced := false
	samlRoles := map[string]any{}
	if rec.Saml != nil {
		samlEnforced = rec.Saml.Enforced
		samlRoles = mapStrToAny(rec.Saml.Roles)
	}

	remoteCachingEnabled := false
	if rec.RemoteCaching != nil && rec.RemoteCaching.Enabled != nil {
		remoteCachingEnabled = *rec.RemoteCaching.Enabled
	}

	res, err := CreateResource(runtime, "vercel.team", map[string]*llx.RawData{
		"id":                                 llx.StringData(rec.ID),
		"slug":                               llx.StringData(rec.Slug),
		"name":                               llx.StringData(rec.Name),
		"avatar":                             llx.StringData(rec.Avatar),
		"createdAt":                          llx.TimeDataPtr(rec.CreatedAt.Time()),
		"samlEnforced":                       llx.BoolData(samlEnforced),
		"samlRoles":                          llx.MapData(samlRoles, types.String),
		"sensitiveEnvironmentVariablePolicy": llx.StringDataPtr(rec.SensitiveEnvironmentVariablePolicy),
		"remoteCachingEnabled":               llx.BoolData(remoteCachingEnabled),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlVercelTeam), nil
}

// initVercelTeam resolves the team a query targets: an explicit id argument, the
// team a discovered asset is scoped to, or the --team connection option.
func initVercelTeam(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.VercelConnection)

	teamID := ""
	if idData, ok := args["id"]; ok {
		if s, ok := idData.Value.(string); ok {
			teamID = s
		}
	}
	if teamID == "" && conn.Asset() != nil {
		for _, pid := range conn.Asset().PlatformIds {
			if t := strings.TrimPrefix(pid, connection.PlatformIdVercelTeam); t != pid {
				teamID = t
				break
			}
		}
	}
	if teamID == "" {
		teamID = conn.TeamID()
	}
	if teamID == "" {
		return nil, nil, errors.New("vercel.team requires a team id")
	}

	var rec teamRecord
	if err := conn.Get(context.Background(), "/v2/teams/"+teamID, nil, &rec); err != nil {
		return nil, nil, err
	}
	if rec.ID == "" {
		rec.ID = teamID
	}

	team, err := newVercelTeam(runtime, &rec)
	if err != nil {
		return nil, nil, err
	}
	return args, team, nil
}

func (c *mqlVercelTeam) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

// --- team members ---------------------------------------------------------

type memberRecord struct {
	UID       string   `json:"uid"`
	Email     string   `json:"email"`
	Username  string   `json:"username"`
	Name      string   `json:"name"`
	Role      string   `json:"role"`
	Confirmed bool     `json:"confirmed"`
	CreatedAt flexTime `json:"createdAt"`
}

func (c *mqlVercelTeam) members() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.VercelConnection)
	records, err := connection.GetPaged[memberRecord](context.Background(), conn, "/v2/teams/"+c.Id.Data+"/members", nil, "members")
	if err != nil {
		if connection.IsForbidden(err) {
			return []any{}, nil
		}
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]
		member, err := CreateResource(c.MqlRuntime, "vercel.team.member", map[string]*llx.RawData{
			"__id":      llx.StringData(c.Id.Data + "/" + rec.UID),
			"uid":       llx.StringData(rec.UID),
			"email":     llx.StringData(rec.Email),
			"username":  llx.StringData(rec.Username),
			"name":      llx.StringData(rec.Name),
			"role":      llx.StringData(rec.Role),
			"confirmed": llx.BoolData(rec.Confirmed),
			"createdAt": llx.TimeDataPtr(rec.CreatedAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, member)
	}
	return res, nil
}

// --- team domains ---------------------------------------------------------

type domainRecord struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	ServiceType         string   `json:"serviceType"`
	Verified            bool     `json:"verified"`
	Nameservers         []string `json:"nameservers"`
	IntendedNameservers []string `json:"intendedNameservers"`
	CdnEnabled          bool     `json:"cdnEnabled"`
	Renew               *bool    `json:"renew"`
	CreatedAt           flexTime `json:"createdAt"`
	ExpiresAt           flexTime `json:"expiresAt"`
	BoughtAt            flexTime `json:"boughtAt"`
}

func (c *mqlVercelTeam) domains() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.VercelConnection)
	records, err := connection.GetPaged[domainRecord](context.Background(), conn, "/v5/domains", connection.TeamQuery(c.Id.Data), "domains")
	if err != nil {
		if connection.IsForbidden(err) {
			return []any{}, nil
		}
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]
		renew := rec.Renew != nil && *rec.Renew
		domain, err := CreateResource(c.MqlRuntime, "vercel.domain", map[string]*llx.RawData{
			"id":                  llx.StringData(rec.ID),
			"name":                llx.StringData(rec.Name),
			"serviceType":         llx.StringData(rec.ServiceType),
			"verified":            llx.BoolData(rec.Verified),
			"nameservers":         llx.ArrayData(strSliceToAny(rec.Nameservers), types.String),
			"intendedNameservers": llx.ArrayData(strSliceToAny(rec.IntendedNameservers), types.String),
			"cdnEnabled":          llx.BoolData(rec.CdnEnabled),
			"renewAutomatically":  llx.BoolData(renew),
			"createdAt":           llx.TimeDataPtr(rec.CreatedAt.Time()),
			"expiresAt":           llx.TimeDataPtr(rec.ExpiresAt.Time()),
			"boughtAt":            llx.TimeDataPtr(rec.BoughtAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		mqlDomain := domain.(*mqlVercelDomain)
		mqlDomain.teamID = c.Id.Data
		res = append(res, mqlDomain)
	}
	return res, nil
}

func (c *mqlVercelDomain) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

// --- edge config ----------------------------------------------------------

type edgeConfigRecord struct {
	ID          string   `json:"id"`
	Slug        string   `json:"slug"`
	ItemCount   *int64   `json:"itemCount"`
	SizeInBytes *int64   `json:"sizeInBytes"`
	CreatedAt   flexTime `json:"createdAt"`
	UpdatedAt   flexTime `json:"updatedAt"`
}

func (c *mqlVercelTeam) edgeConfigs() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.VercelConnection)
	var records []edgeConfigRecord
	if err := conn.Get(context.Background(), "/v1/edge-config", connection.TeamQuery(c.Id.Data), &records); err != nil {
		if connection.IsForbidden(err) {
			return []any{}, nil
		}
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]
		ec, err := CreateResource(c.MqlRuntime, "vercel.edgeConfig", map[string]*llx.RawData{
			"id":          llx.StringData(rec.ID),
			"slug":        llx.StringData(rec.Slug),
			"itemCount":   llx.IntData(intPtrOrZero(rec.ItemCount)),
			"sizeInBytes": llx.IntData(intPtrOrZero(rec.SizeInBytes)),
			"createdAt":   llx.TimeDataPtr(rec.CreatedAt.Time()),
			"updatedAt":   llx.TimeDataPtr(rec.UpdatedAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, ec)
	}
	return res, nil
}

func (c *mqlVercelEdgeConfig) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

// --- log drains -----------------------------------------------------------

type logDrainRecord struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	URL            string   `json:"url"`
	DeliveryFormat string   `json:"deliveryFormat"`
	Sources        []string `json:"sources"`
	Environments   []string `json:"environments"`
	CreatedAt      flexTime `json:"createdAt"`
}

func (c *mqlVercelTeam) logDrains() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.VercelConnection)
	var records []logDrainRecord
	if err := conn.Get(context.Background(), "/v1/log-drains", connection.TeamQuery(c.Id.Data), &records); err != nil {
		if connection.IsForbidden(err) {
			return []any{}, nil
		}
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]
		drain, err := CreateResource(c.MqlRuntime, "vercel.logDrain", map[string]*llx.RawData{
			"id":             llx.StringData(rec.ID),
			"name":           llx.StringData(rec.Name),
			"url":            llx.StringData(rec.URL),
			"deliveryFormat": llx.StringData(rec.DeliveryFormat),
			"sources":        llx.ArrayData(strSliceToAny(rec.Sources), types.String),
			"environments":   llx.ArrayData(strSliceToAny(rec.Environments), types.String),
			"createdAt":      llx.TimeDataPtr(rec.CreatedAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, drain)
	}
	return res, nil
}

func (c *mqlVercelLogDrain) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

// --- webhooks -------------------------------------------------------------

type webhookRecord struct {
	ID         string   `json:"id"`
	URL        string   `json:"url"`
	Events     []string `json:"events"`
	ProjectIds []string `json:"projectIds"`
	CreatedAt  flexTime `json:"createdAt"`
	UpdatedAt  flexTime `json:"updatedAt"`
}

func (c *mqlVercelTeam) webhooks() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.VercelConnection)
	var records []webhookRecord
	if err := conn.Get(context.Background(), "/v1/webhooks", connection.TeamQuery(c.Id.Data), &records); err != nil {
		if connection.IsForbidden(err) {
			return []any{}, nil
		}
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]
		hook, err := CreateResource(c.MqlRuntime, "vercel.webhook", map[string]*llx.RawData{
			"id":         llx.StringData(rec.ID),
			"url":        llx.StringData(rec.URL),
			"events":     llx.ArrayData(strSliceToAny(rec.Events), types.String),
			"projectIds": llx.ArrayData(strSliceToAny(rec.ProjectIds), types.String),
			"createdAt":  llx.TimeDataPtr(rec.CreatedAt.Time()),
			"updatedAt":  llx.TimeDataPtr(rec.UpdatedAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, hook)
	}
	return res, nil
}

func (c *mqlVercelWebhook) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

// --- integration configurations -------------------------------------------

type integrationConfigurationRecord struct {
	ID               string   `json:"id"`
	Slug             string   `json:"slug"`
	Scopes           []string `json:"scopes"`
	InstallationType string   `json:"installationType"`
	// Source is the fallback for the installationType field: older
	// configurations report how they were installed under "source" instead.
	Source           string   `json:"source"`
	ProjectSelection string   `json:"projectSelection"`
	Projects         []string `json:"projects"`
	CreatedAt        flexTime `json:"createdAt"`
	UpdatedAt        flexTime `json:"updatedAt"`
}

func (c *mqlVercelTeam) integrationConfigurations() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.VercelConnection)
	// The endpoint requires a view; "account" lists every configuration
	// installed on the team.
	query := connection.TeamQuery(c.Id.Data)
	query.Set("view", "account")
	var records []integrationConfigurationRecord
	if err := conn.Get(context.Background(), "/v1/integrations/configurations", query, &records); err != nil {
		if connection.IsForbidden(err) {
			return []any{}, nil
		}
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]
		installationType := rec.InstallationType
		if installationType == "" {
			installationType = rec.Source
		}
		cfg, err := CreateResource(c.MqlRuntime, "vercel.integrationConfiguration", map[string]*llx.RawData{
			"id":               llx.StringData(rec.ID),
			"slug":             llx.StringData(rec.Slug),
			"scopes":           llx.ArrayData(strSliceToAny(rec.Scopes), types.String),
			"installationType": llx.StringData(installationType),
			"projectSelection": llx.StringData(rec.ProjectSelection),
			"projectIds":       llx.ArrayData(strSliceToAny(rec.Projects), types.String),
			"createdAt":        llx.TimeDataPtr(rec.CreatedAt.Time()),
			"updatedAt":        llx.TimeDataPtr(rec.UpdatedAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, cfg)
	}
	return res, nil
}

func (c *mqlVercelIntegrationConfiguration) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

// --- access groups (enterprise) -------------------------------------------

type accessGroupRecord struct {
	ID            string   `json:"accessGroupId"`
	Name          string   `json:"name"`
	MembersCount  *int64   `json:"membersCount"`
	ProjectsCount *int64   `json:"projectsCount"`
	CreatedAt     flexTime `json:"createdAt"`
	UpdatedAt     flexTime `json:"updatedAt"`
}

func (c *mqlVercelTeam) accessGroups() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.VercelConnection)
	records, err := connection.GetPaged[accessGroupRecord](context.Background(), conn, "/v1/access-groups", connection.TeamQuery(c.Id.Data), "accessGroups")
	if err != nil {
		// Access groups are an Enterprise feature; degrade to empty on plans
		// that do not expose the endpoint.
		if connection.IsForbidden(err) || connection.IsNotFound(err) {
			return []any{}, nil
		}
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]
		group, err := CreateResource(c.MqlRuntime, "vercel.accessGroup", map[string]*llx.RawData{
			"id":            llx.StringData(rec.ID),
			"name":          llx.StringData(rec.Name),
			"membersCount":  llx.IntData(intPtrOrZero(rec.MembersCount)),
			"projectsCount": llx.IntData(intPtrOrZero(rec.ProjectsCount)),
			"createdAt":     llx.TimeDataPtr(rec.CreatedAt.Time()),
			"updatedAt":     llx.TimeDataPtr(rec.UpdatedAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		mqlGroup := group.(*mqlVercelAccessGroup)
		mqlGroup.teamID = c.Id.Data
		res = append(res, mqlGroup)
	}
	return res, nil
}

func (c *mqlVercelAccessGroup) id() (string, error) {
	return c.Id.Data, c.Id.Error
}
