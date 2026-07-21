// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/vercel/connection"
	"go.mondoo.com/mql/v13/types"
)

// mqlVercelProjectInternal caches the team a project belongs to so project-scoped
// API calls (environment variables, deployments, firewall) can pass teamId.
type mqlVercelProjectInternal struct {
	teamID string
}

type deploymentTypeHolder struct {
	DeploymentType string `json:"deploymentType"`
}

type trustedIpsRecord struct {
	DeploymentType string           `json:"deploymentType"`
	ProtectionMode string           `json:"protectionMode"`
	Addresses      []map[string]any `json:"addresses"`
}

type projectLink struct {
	Type             string  `json:"type"`
	Org              string  `json:"org"`
	Repo             string  `json:"repo"`
	ProductionBranch *string `json:"productionBranch"`
}

type cronsRecord struct {
	Definitions []map[string]any `json:"definitions"`
}

type projectRecord struct {
	ID                   string                `json:"id"`
	Name                 string                `json:"name"`
	Framework            *string               `json:"framework"`
	NodeVersion          string                `json:"nodeVersion"`
	RootDirectory        *string               `json:"rootDirectory"`
	BuildCommand         *string               `json:"buildCommand"`
	DevCommand           *string               `json:"devCommand"`
	InstallCommand       *string               `json:"installCommand"`
	OutputDirectory      *string               `json:"outputDirectory"`
	PublicSource         *bool                 `json:"publicSource"`
	AutoExposeSystemEnvs *bool                 `json:"autoExposeSystemEnvs"`
	GitForkProtection    *bool                 `json:"gitForkProtection"`
	GitLFS               *bool                 `json:"gitLFS"`
	Live                 *bool                 `json:"live"`
	CreatedAt            flexTime              `json:"createdAt"`
	UpdatedAt            flexTime              `json:"updatedAt"`
	SsoProtection        *deploymentTypeHolder `json:"ssoProtection"`
	PasswordProtection   *deploymentTypeHolder `json:"passwordProtection"`
	TrustedIps           *trustedIpsRecord     `json:"trustedIps"`
	Link                 *projectLink          `json:"link"`
	Crons                *cronsRecord          `json:"crons"`
}

func holderType(h *deploymentTypeHolder) *string {
	if h == nil {
		return nil
	}
	return &h.DeploymentType
}

func newVercelProject(runtime *plugin.Runtime, teamID string, rec *projectRecord) (*mqlVercelProject, error) {
	var trustedMode, trustedType *string
	trustedAddresses := []any{}
	if rec.TrustedIps != nil {
		trustedMode = &rec.TrustedIps.ProtectionMode
		trustedType = &rec.TrustedIps.DeploymentType
		trustedAddresses = dictSliceToAny(rec.TrustedIps.Addresses)
	}

	var repoType, repoOwner, repoName, productionBranch *string
	if rec.Link != nil {
		repoType = &rec.Link.Type
		repoOwner = &rec.Link.Org
		repoName = &rec.Link.Repo
		productionBranch = rec.Link.ProductionBranch
	}

	cronJobs := []any{}
	if rec.Crons != nil {
		cronJobs = dictSliceToAny(rec.Crons.Definitions)
	}

	res, err := CreateResource(runtime, "vercel.project", map[string]*llx.RawData{
		"id":                               llx.StringData(rec.ID),
		"name":                             llx.StringData(rec.Name),
		"framework":                        llx.StringDataPtr(rec.Framework),
		"nodeVersion":                      llx.StringData(rec.NodeVersion),
		"rootDirectory":                    llx.StringDataPtr(rec.RootDirectory),
		"buildCommand":                     llx.StringDataPtr(rec.BuildCommand),
		"devCommand":                       llx.StringDataPtr(rec.DevCommand),
		"installCommand":                   llx.StringDataPtr(rec.InstallCommand),
		"outputDirectory":                  llx.StringDataPtr(rec.OutputDirectory),
		"publicSource":                     llx.BoolData(rec.PublicSource != nil && *rec.PublicSource),
		"autoExposeSystemEnvs":             llx.BoolData(rec.AutoExposeSystemEnvs != nil && *rec.AutoExposeSystemEnvs),
		"gitForkProtection":                llx.BoolData(rec.GitForkProtection != nil && *rec.GitForkProtection),
		"gitLFS":                           llx.BoolData(rec.GitLFS != nil && *rec.GitLFS),
		"live":                             llx.BoolData(rec.Live != nil && *rec.Live),
		"createdAt":                        llx.TimeDataPtr(rec.CreatedAt.Time()),
		"updatedAt":                        llx.TimeDataPtr(rec.UpdatedAt.Time()),
		"ssoProtectionDeploymentType":      llx.StringDataPtr(holderType(rec.SsoProtection)),
		"passwordProtectionDeploymentType": llx.StringDataPtr(holderType(rec.PasswordProtection)),
		"trustedIpsProtectionMode":         llx.StringDataPtr(trustedMode),
		"trustedIpsDeploymentType":         llx.StringDataPtr(trustedType),
		"trustedIpsAddresses":              llx.ArrayData(trustedAddresses, types.Dict),
		"repositoryType":                   llx.StringDataPtr(repoType),
		"repositoryOwner":                  llx.StringDataPtr(repoOwner),
		"repositoryName":                   llx.StringDataPtr(repoName),
		"productionBranch":                 llx.StringDataPtr(productionBranch),
		"cronJobs":                         llx.ArrayData(cronJobs, types.Dict),
	})
	if err != nil {
		return nil, err
	}
	project := res.(*mqlVercelProject)
	project.teamID = teamID
	return project, nil
}

func (c *mqlVercelTeam) projects() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.VercelConnection)
	records, err := connection.GetPaged[projectRecord](context.Background(), conn, "/v9/projects", connection.TeamQuery(c.Id.Data), "projects")
	if err != nil {
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]
		project, err := newVercelProject(c.MqlRuntime, c.Id.Data, &rec)
		if err != nil {
			return nil, err
		}
		res = append(res, project)
	}
	return res, nil
}

// initVercelProject resolves the project a query targets from an explicit id, the
// project a discovered asset is scoped to, or the connection options.
func initVercelProject(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.VercelConnection)

	projectID := ""
	if idData, ok := args["id"]; ok {
		if s, ok := idData.Value.(string); ok {
			projectID = s
		}
	}
	if projectID == "" && conn.Asset() != nil {
		for _, pid := range conn.Asset().PlatformIds {
			if p := strings.TrimPrefix(pid, connection.PlatformIdVercelProject); p != pid {
				projectID = p
				break
			}
		}
	}
	if projectID == "" {
		projectID = conn.ProjectID()
	}
	if projectID == "" {
		return nil, nil, errors.New("vercel.project requires a project id")
	}

	teamID := conn.TeamID()
	var rec projectRecord
	if err := conn.Get(context.Background(), "/v9/projects/"+projectID, connection.TeamQuery(teamID), &rec); err != nil {
		return nil, nil, err
	}
	if rec.ID == "" {
		rec.ID = projectID
	}

	project, err := newVercelProject(runtime, teamID, &rec)
	if err != nil {
		return nil, nil, err
	}
	return args, project, nil
}

func (c *mqlVercelProject) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

// resolveProjectRefs resolves a list of project ids into typed vercel.project
// references. Ids that no longer resolve (project deleted, or not visible to the
// token) are skipped rather than failing the whole list; other errors propagate.
//
// This makes one project-by-id call per id (N+1); the Vercel API has no
// filter-by-id-list endpoint, and webhooks / integrations are typically scoped
// to a handful of projects, so the simple per-id resolve matches the existing
// store.connectedProjects pattern.
func resolveProjectRefs(runtime *plugin.Runtime, teamID string, projectIDs []string) ([]any, error) {
	conn := runtime.Connection.(*connection.VercelConnection)

	out := []any{}
	for _, projectID := range projectIDs {
		if projectID == "" {
			continue
		}
		var rec projectRecord
		if err := conn.Get(context.Background(), "/v9/projects/"+projectID, connection.TeamQuery(teamID), &rec); err != nil {
			if connection.IsForbidden(err) || connection.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		if rec.ID == "" {
			rec.ID = projectID
		}
		project, err := newVercelProject(runtime, teamID, &rec)
		if err != nil {
			return nil, err
		}
		out = append(out, project)
	}
	return out, nil
}

// --- environment variables ------------------------------------------------

type envRecord struct {
	ID        string          `json:"id"`
	Key       string          `json:"key"`
	Type      string          `json:"type"`
	Target    json.RawMessage `json:"target"`
	GitBranch *string         `json:"gitBranch"`
	CreatedAt flexTime        `json:"createdAt"`
	UpdatedAt flexTime        `json:"updatedAt"`
}

// parseTargets normalizes the env target, which Vercel returns as either a
// string array or a single string.
func parseTargets(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "null" || len(trimmed) == 0 {
		return nil
	}
	if trimmed[0] == '[' {
		var arr []string
		if json.Unmarshal(raw, &arr) == nil {
			return arr
		}
		return nil
	}
	var single string
	if json.Unmarshal(raw, &single) == nil && single != "" {
		return []string{single}
	}
	return nil
}

func (c *mqlVercelProject) environmentVariables() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.VercelConnection)
	records, err := connection.GetPaged[envRecord](context.Background(), conn, "/v9/projects/"+c.Id.Data+"/env", connection.TeamQuery(c.teamID), "envs")
	if err != nil {
		if connection.IsForbidden(err) {
			return []any{}, nil
		}
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]
		env, err := CreateResource(c.MqlRuntime, "vercel.project.environmentVariable", map[string]*llx.RawData{
			"id":        llx.StringData(rec.ID),
			"key":       llx.StringData(rec.Key),
			"type":      llx.StringData(rec.Type),
			"target":    llx.ArrayData(strSliceToAny(parseTargets(rec.Target)), types.String),
			"gitBranch": llx.StringDataPtr(rec.GitBranch),
			"createdAt": llx.TimeDataPtr(rec.CreatedAt.Time()),
			"updatedAt": llx.TimeDataPtr(rec.UpdatedAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, env)
	}
	return res, nil
}

func (c *mqlVercelProjectEnvironmentVariable) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

// --- deployments ----------------------------------------------------------

type deploymentCreator struct {
	UID      string `json:"uid"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type deploymentRecord struct {
	UID          string             `json:"uid"`
	Name         string             `json:"name"`
	URL          string             `json:"url"`
	State        string             `json:"state"`
	ReadyState   string             `json:"readyState"`
	Target       *string            `json:"target"`
	Source       string             `json:"source"`
	Type         string             `json:"type"`
	Creator      *deploymentCreator `json:"creator"`
	InspectorURL string             `json:"inspectorUrl"`
	Created      flexTime           `json:"created"`
	CreatedAt    flexTime           `json:"createdAt"`
}

func (c *mqlVercelProject) deployments() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.VercelConnection)
	query := connection.TeamQuery(c.teamID)
	query.Set("projectId", c.Id.Data)
	records, err := connection.GetPaged[deploymentRecord](context.Background(), conn, "/v6/deployments", query, "deployments")
	if err != nil {
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]
		state := rec.State
		if state == "" {
			state = rec.ReadyState
		}
		created := rec.Created.Time()
		if created == nil {
			created = rec.CreatedAt.Time()
		}
		var creatorUID, creatorUsername, creatorEmail string
		if rec.Creator != nil {
			creatorUID = rec.Creator.UID
			creatorUsername = rec.Creator.Username
			creatorEmail = rec.Creator.Email
		}
		deployment, err := CreateResource(c.MqlRuntime, "vercel.deployment", map[string]*llx.RawData{
			"id":              llx.StringData(rec.UID),
			"name":            llx.StringData(rec.Name),
			"url":             llx.StringData(rec.URL),
			"state":           llx.StringData(state),
			"target":          llx.StringDataPtr(rec.Target),
			"source":          llx.StringData(rec.Source),
			"deploymentType":  llx.StringData(rec.Type),
			"creatorUid":      llx.StringData(creatorUID),
			"creatorUsername": llx.StringData(creatorUsername),
			"creatorEmail":    llx.StringData(creatorEmail),
			"inspectorUrl":    llx.StringData(rec.InspectorURL),
			"createdAt":       llx.TimeDataPtr(created),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, deployment)
	}
	return res, nil
}

func (c *mqlVercelDeployment) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

// --- project domains ------------------------------------------------------

// mqlVercelProjectDomainInternal caches the team a project domain belongs to and
// its apex name, so apexDomain can resolve the typed team-domain reference.
type mqlVercelProjectDomainInternal struct {
	teamID   string
	apexName string
}

type projectDomainRecord struct {
	Name               string   `json:"name"`
	ApexName           string   `json:"apexName"`
	Redirect           *string  `json:"redirect"`
	RedirectStatusCode *int64   `json:"redirectStatusCode"`
	GitBranch          *string  `json:"gitBranch"`
	Verified           bool     `json:"verified"`
	CreatedAt          flexTime `json:"createdAt"`
	UpdatedAt          flexTime `json:"updatedAt"`
}

func (c *mqlVercelProject) domains() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.VercelConnection)
	records, err := connection.GetPaged[projectDomainRecord](context.Background(), conn, "/v9/projects/"+c.Id.Data+"/domains", connection.TeamQuery(c.teamID), "domains")
	if err != nil {
		if connection.IsForbidden(err) {
			return []any{}, nil
		}
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]
		domain, err := CreateResource(c.MqlRuntime, "vercel.project.domain", map[string]*llx.RawData{
			"__id":               llx.StringData(c.Id.Data + "/" + rec.Name),
			"name":               llx.StringData(rec.Name),
			"apexName":           llx.StringData(rec.ApexName),
			"redirect":           llx.StringDataPtr(rec.Redirect),
			"redirectStatusCode": llx.IntDataPtr(rec.RedirectStatusCode),
			"gitBranch":          llx.StringDataPtr(rec.GitBranch),
			"verified":           llx.BoolData(rec.Verified),
			"createdAt":          llx.TimeDataPtr(rec.CreatedAt.Time()),
			"updatedAt":          llx.TimeDataPtr(rec.UpdatedAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		mqlDomain := domain.(*mqlVercelProjectDomain)
		mqlDomain.teamID = c.teamID
		mqlDomain.apexName = rec.ApexName
		res = append(res, mqlDomain)
	}
	return res, nil
}

// apexDomain resolves the apex to the Vercel-managed team domain. Not every apex
// is Vercel-managed (external or delegated domains exist), so the accessor
// degrades to null when the domain is not found or not accessible.
func (c *mqlVercelProjectDomain) apexDomain() (*mqlVercelDomain, error) {
	if c.apexName == "" {
		c.ApexDomain.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	conn := c.MqlRuntime.Connection.(*connection.VercelConnection)
	var wrapper struct {
		Domain domainRecord `json:"domain"`
	}
	if err := conn.Get(context.Background(), "/v5/domains/"+c.apexName, connection.TeamQuery(c.teamID), &wrapper); err != nil {
		if connection.IsForbidden(err) || connection.IsNotFound(err) {
			c.ApexDomain.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	rec := wrapper.Domain
	if rec.Name == "" {
		rec.Name = c.apexName
	}
	return newVercelDomain(c.MqlRuntime, c.teamID, &rec)
}
