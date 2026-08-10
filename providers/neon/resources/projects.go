// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/neon/connection"
	"go.mondoo.com/mql/v13/types"
)

// mqlNeonProjectInternal caches the organization the project belongs to, which
// is only present on the project payload.
type mqlNeonProjectInternal struct {
	cacheOrgID string
}

type projectRecord struct {
	ID                      string               `json:"id"`
	Name                    string               `json:"name"`
	PlatformID              string               `json:"platform_id"`
	RegionID                string               `json:"region_id"`
	PgVersion               int64                `json:"pg_version"`
	Provisioner             string               `json:"provisioner"`
	ProxyHost               string               `json:"proxy_host"`
	StorePasswords          bool                 `json:"store_passwords"`
	HistoryRetentionSeconds *int64               `json:"history_retention_seconds"`
	HipaaEnabledAt          neonTime             `json:"hipaa_enabled_at"`
	OrgID                   *string              `json:"org_id"`
	Owner                   *projectOwnerRecord  `json:"owner"`
	CreatedAt               neonTime             `json:"created_at"`
	UpdatedAt               neonTime             `json:"updated_at"`
	ComputeLastActiveAt     neonTime             `json:"compute_last_active_at"`
	Settings                *projectSettingsData `json:"settings"`
}

type projectOwnerRecord struct {
	Email string `json:"email"`
}

// projectSettingsData holds the project-wide controls. Neon omits a setting
// that has never been switched on, so every field is optional.
type projectSettingsData struct {
	AllowedIps               *allowedIpsData `json:"allowed_ips"`
	AuditLogLevel            *string         `json:"audit_log_level"`
	BlockPublicConnections   *bool           `json:"block_public_connections"`
	BlockVpcConnections      *bool           `json:"block_vpc_connections"`
	EnableLogicalReplication *bool           `json:"enable_logical_replication"`
	Hipaa                    *bool           `json:"hipaa"`
}

type allowedIpsData struct {
	Ips                   *[]string `json:"ips"`
	ProtectedBranchesOnly *bool     `json:"protected_branches_only"`
}

// projects lists every project the API key can reach, narrowed to the
// organization and project the connection is scoped to.
func (n *mqlNeon) projects() ([]any, error) {
	c := neonConn(n.MqlRuntime)

	query := url.Values{}
	if orgID := c.OrganizationFilter(); orgID != "" {
		query.Set("org_id", orgID)
	}

	records, err := connection.GetPagedCursor[projectRecord](context.Background(), c,
		"/projects", query, "projects")
	if err != nil {
		return nil, err
	}

	projectFilter := c.ProjectFilter()

	var res []any
	for i := range records {
		rec := records[i]
		if projectFilter != "" && rec.ID != projectFilter {
			continue
		}
		project, err := newNeonProject(n.MqlRuntime, &rec)
		if err != nil {
			return nil, err
		}
		res = append(res, project)
	}
	return res, nil
}

// projects lists the organization's projects.
func (o *mqlNeonOrganization) projects() ([]any, error) {
	c := neonConn(o.MqlRuntime)

	query := url.Values{}
	query.Set("org_id", o.Id.Data)

	records, err := connection.GetPagedCursor[projectRecord](context.Background(), c,
		"/projects", query, "projects")
	if err != nil {
		if connection.IsForbidden(err) {
			o.Projects = plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}
			return nil, nil
		}
		return nil, err
	}

	projectFilter := c.ProjectFilter()

	var res []any
	for i := range records {
		rec := records[i]
		if projectFilter != "" && rec.ID != projectFilter {
			continue
		}
		project, err := newNeonProject(o.MqlRuntime, &rec)
		if err != nil {
			return nil, err
		}
		res = append(res, project)
	}
	return res, nil
}

func newNeonProject(runtime *plugin.Runtime, rec *projectRecord) (*mqlNeonProject, error) {
	settings := rec.Settings
	if settings == nil {
		settings = &projectSettingsData{}
	}

	var allowedIps []string
	protectedOnly := false
	if settings.AllowedIps != nil {
		if settings.AllowedIps.Ips != nil {
			allowedIps = *settings.AllowedIps.Ips
		}
		protectedOnly = boolPtr(settings.AllowedIps.ProtectedBranchesOnly)
	}

	ownerEmail := ""
	if rec.Owner != nil {
		ownerEmail = rec.Owner.Email
	}

	historyRetention := int64(0)
	if rec.HistoryRetentionSeconds != nil {
		historyRetention = *rec.HistoryRetentionSeconds
	}

	res, err := CreateResource(runtime, "neon.project", map[string]*llx.RawData{
		"id":                              llx.StringData(rec.ID),
		"name":                            llx.StringData(rec.Name),
		"platformId":                      llx.StringData(rec.PlatformID),
		"regionId":                        llx.StringData(rec.RegionID),
		"pgVersion":                       llx.IntData(rec.PgVersion),
		"provisioner":                     llx.StringData(rec.Provisioner),
		"proxyHost":                       llx.StringData(rec.ProxyHost),
		"blockPublicConnections":          llx.BoolData(boolPtr(settings.BlockPublicConnections)),
		"blockVpcConnections":             llx.BoolData(boolPtr(settings.BlockVpcConnections)),
		"allowedIps":                      llx.ArrayData(strSliceToAny(allowedIps), types.String),
		"allowedIpsProtectedBranchesOnly": llx.BoolData(protectedOnly),
		"enableLogicalReplication":        llx.BoolData(boolPtr(settings.EnableLogicalReplication)),
		"storePasswords":                  llx.BoolData(rec.StorePasswords),
		"hipaa":                           llx.BoolData(boolPtr(settings.Hipaa)),
		"hipaaEnabledAt":                  llx.TimeDataPtr(rec.HipaaEnabledAt.Time()),
		"auditLogLevel":                   llx.StringData(strPtr(settings.AuditLogLevel)),
		"historyRetentionSeconds":         llx.IntData(historyRetention),
		"createdAt":                       llx.TimeDataPtr(rec.CreatedAt.Time()),
		"updatedAt":                       llx.TimeDataPtr(rec.UpdatedAt.Time()),
		"computeLastActiveAt":             llx.TimeDataPtr(rec.ComputeLastActiveAt.Time()),
		"ownerEmail":                      llx.StringData(ownerEmail),
	})
	if err != nil {
		return nil, err
	}

	project := res.(*mqlNeonProject)
	project.cacheOrgID = strPtr(rec.OrgID)
	return project, nil
}

// initNeonProject resolves the project a query targets: an explicit id argument
// or the project a discovered asset is scoped to.
func initNeonProject(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	c := neonConn(runtime)

	projectID := ""
	if data, ok := args["id"]; ok {
		if s, ok := data.Value.(string); ok {
			projectID = s
		}
	}
	if projectID == "" && c.Asset() != nil {
		for _, pid := range c.Asset().PlatformIds {
			if t := strings.TrimPrefix(pid, connection.PlatformIdNeonProject); t != pid {
				projectID = t
				break
			}
		}
	}
	if projectID == "" {
		projectID = c.ProjectFilter()
	}
	if projectID == "" {
		return nil, nil, errors.New("neon.project requires a project id")
	}

	var resp struct {
		Project projectRecord `json:"project"`
	}
	if err := c.Get(context.Background(), "/projects/"+url.PathEscape(projectID), nil, &resp); err != nil {
		return nil, nil, err
	}
	if resp.Project.ID == "" {
		resp.Project.ID = projectID
	}

	project, err := newNeonProject(runtime, &resp.Project)
	if err != nil {
		return nil, nil, err
	}
	return args, project, nil
}

func (p *mqlNeonProject) id() (string, error) {
	return p.Id.Data, p.Id.Error
}

// organization resolves the organization the project belongs to. A personal
// project has none.
func (p *mqlNeonProject) organization() (*mqlNeonOrganization, error) {
	if p.cacheOrgID == "" {
		p.Organization.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	res, err := NewResource(p.MqlRuntime, "neon.organization", map[string]*llx.RawData{
		"id": llx.StringData(p.cacheOrgID),
	})
	if err != nil {
		// A project can belong to an organization the key is not a member of.
		if connection.IsForbidden(err) || connection.IsNotFound(err) {
			p.Organization.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	return res.(*mqlNeonOrganization), nil
}

// --- project access -------------------------------------------------------

type permissionRecord struct {
	ID             string   `json:"id"`
	GrantedToEmail string   `json:"granted_to_email"`
	GrantedAt      neonTime `json:"granted_at"`
	RevokedAt      neonTime `json:"revoked_at"`
}

func (p *mqlNeonProject) permissions() ([]any, error) {
	c := neonConn(p.MqlRuntime)

	records, err := connection.GetList[permissionRecord](context.Background(), c,
		"/projects/"+url.PathEscape(p.Id.Data)+"/permissions", nil, "project_permissions")
	if err != nil {
		if connection.IsForbidden(err) {
			p.Permissions = plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}
			return nil, nil
		}
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]
		permission, err := CreateResource(p.MqlRuntime, "neon.project.permission", map[string]*llx.RawData{
			"id":             llx.StringData(rec.ID),
			"grantedToEmail": llx.StringData(rec.GrantedToEmail),
			"grantedAt":      llx.TimeDataPtr(rec.GrantedAt.Time()),
			"revokedAt":      llx.TimeDataPtr(rec.RevokedAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, permission)
	}
	return res, nil
}

func (p *mqlNeonProjectPermission) id() (string, error) {
	return p.Id.Data, p.Id.Error
}

// --- trusted key sets -----------------------------------------------------

// mqlNeonProjectJwksEndpointInternal caches the project and branch the key set
// applies to so the branch can be resolved from the project's branch list.
type mqlNeonProjectJwksEndpointInternal struct {
	cacheProjectID string
	cacheBranchID  string
}

type jwksRecord struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	BranchID     *string   `json:"branch_id"`
	JwksURL      string    `json:"jwks_url"`
	ProviderName string    `json:"provider_name"`
	JwtAudience  *string   `json:"jwt_audience"`
	RoleNames    *[]string `json:"role_names"`
	CreatedAt    neonTime  `json:"created_at"`
	UpdatedAt    neonTime  `json:"updated_at"`
}

func (p *mqlNeonProject) jwksEndpoints() ([]any, error) {
	c := neonConn(p.MqlRuntime)

	records, err := connection.GetList[jwksRecord](context.Background(), c,
		"/projects/"+url.PathEscape(p.Id.Data)+"/jwks", nil, "jwks")
	if err != nil {
		if connection.IsForbidden(err) {
			p.JwksEndpoints = plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}
			return nil, nil
		}
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]

		var roleNames []string
		if rec.RoleNames != nil {
			roleNames = *rec.RoleNames
		}

		endpoint, err := CreateResource(p.MqlRuntime, "neon.project.jwksEndpoint", map[string]*llx.RawData{
			"id":           llx.StringData(rec.ID),
			"jwksUrl":      llx.StringData(rec.JwksURL),
			"providerName": llx.StringData(rec.ProviderName),
			"jwtAudience":  llx.StringData(strPtr(rec.JwtAudience)),
			"roleNames":    llx.ArrayData(strSliceToAny(roleNames), types.String),
			"createdAt":    llx.TimeDataPtr(rec.CreatedAt.Time()),
			"updatedAt":    llx.TimeDataPtr(rec.UpdatedAt.Time()),
		})
		if err != nil {
			return nil, err
		}

		jwks := endpoint.(*mqlNeonProjectJwksEndpoint)
		jwks.cacheProjectID = p.Id.Data
		jwks.cacheBranchID = strPtr(rec.BranchID)
		res = append(res, jwks)
	}
	return res, nil
}

func (j *mqlNeonProjectJwksEndpoint) id() (string, error) {
	return j.Id.Data, j.Id.Error
}

// branch resolves the branch the key set applies to. A key set registered for
// the whole project has none.
func (j *mqlNeonProjectJwksEndpoint) branch() (*mqlNeonBranch, error) {
	if j.cacheBranchID == "" {
		j.Branch.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	branch, err := branchByID(j.MqlRuntime, j.cacheProjectID, j.cacheBranchID)
	if err != nil {
		return nil, err
	}
	if branch == nil {
		j.Branch.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return branch, nil
}

// --- private connectivity -------------------------------------------------

type vpcEndpointRecord struct {
	VpcEndpointID string `json:"vpc_endpoint_id"`
	Label         string `json:"label"`
}

func (p *mqlNeonProject) vpcEndpoints() ([]any, error) {
	c := neonConn(p.MqlRuntime)

	records, err := connection.GetList[vpcEndpointRecord](context.Background(), c,
		"/projects/"+url.PathEscape(p.Id.Data)+"/vpc_endpoints", nil, "endpoints")
	if err != nil {
		// Private connectivity is a plan-gated feature.
		if connection.IsForbidden(err) || connection.IsNotFound(err) {
			p.VpcEndpoints = plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}
			return nil, nil
		}
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]
		endpoint, err := CreateResource(p.MqlRuntime, "neon.vpcEndpoint", map[string]*llx.RawData{
			"__id":          llx.StringData(p.Id.Data + "/" + rec.VpcEndpointID),
			"vpcEndpointId": llx.StringData(rec.VpcEndpointID),
			"label":         llx.StringData(rec.Label),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, endpoint)
	}
	return res, nil
}
