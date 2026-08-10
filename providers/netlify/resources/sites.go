// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/netlify/connection"
	"go.mondoo.com/mql/v13/types"
)

// mqlNetlifySiteInternal caches the identifiers the site's own child lookups
// need: the account the environment-variable endpoint is keyed on, and the
// deploy key the build clones with.
type mqlNetlifySiteInternal struct {
	cacheAccountID   string
	cacheDeployKeyID string
}

type siteRecord struct {
	ID                        string             `json:"id"`
	Name                      string             `json:"name"`
	State                     string             `json:"state"`
	Plan                      string             `json:"plan"`
	URL                       string             `json:"url"`
	SslURL                    string             `json:"ssl_url"`
	AdminURL                  string             `json:"admin_url"`
	CustomDomain              string             `json:"custom_domain"`
	DomainAliases             []string           `json:"domain_aliases"`
	BranchDeployCustomDomain  string             `json:"branch_deploy_custom_domain"`
	DeployPreviewCustomDomain string             `json:"deploy_preview_custom_domain"`
	NotificationEmail         string             `json:"notification_email"`
	Password                  string             `json:"password"`
	Ssl                       bool               `json:"ssl"`
	ForceSsl                  bool               `json:"force_ssl"`
	ManagedDns                bool               `json:"managed_dns"`
	PreventNonGitProdDeploys  bool               `json:"prevent_non_git_prod_deploys"`
	BuildImage                string             `json:"build_image"`
	Prerender                 string             `json:"prerender"`
	FunctionsRegion           string             `json:"functions_region"`
	AccountID                 string             `json:"account_id"`
	AccountSlug               string             `json:"account_slug"`
	CreatedAt                 netlifyTime        `json:"created_at"`
	UpdatedAt                 netlifyTime        `json:"updated_at"`
	BuildSettings             *buildSettingsData `json:"build_settings"`
}

// passwordProtected reports whether the site is gated behind a site-wide
// basic-auth password. Netlify signals that by returning the password on the
// site rather than by a dedicated flag.
func (r *siteRecord) passwordProtected() bool {
	return r.Password != ""
}

// buildSettingsData is the repository configuration a build runs from. Netlify
// returns it inline with the site.
type buildSettingsData struct {
	Provider        string   `json:"provider"`
	RepoURL         string   `json:"repo_url"`
	RepoBranch      string   `json:"repo_branch"`
	RepoPath        string   `json:"repo_path"`
	Cmd             string   `json:"cmd"`
	Dir             string   `json:"dir"`
	FunctionsDir    string   `json:"functions_dir"`
	AllowedBranches []string `json:"allowed_branches"`
	PublicRepo      bool     `json:"public_repo"`
	PrivateLogs     bool     `json:"private_logs"`
	StopBuilds      bool     `json:"stop_builds"`
	DeployKeyID     string   `json:"deploy_key_id"`
}

// sites lists the account's sites, narrowed to the site a discovered asset is
// scoped to.
func (a *mqlNetlifyAccount) sites() ([]any, error) {
	c := netlifyConn(a.MqlRuntime)

	slug := a.accountSlug()
	if slug == "" {
		return nil, errors.New("netlify.account.sites requires the account slug")
	}

	records, err := connection.GetPaged[siteRecord](context.Background(), c,
		"/"+url.PathEscape(slug)+"/sites", nil)
	if err != nil {
		return nil, err
	}

	siteFilter := c.SiteFilter()

	var res []any
	for i := range records {
		rec := records[i]
		if siteFilter != "" && rec.ID != siteFilter {
			continue
		}
		site, err := newNetlifySite(a.MqlRuntime, &rec)
		if err != nil {
			return nil, err
		}
		res = append(res, site)
	}
	return res, nil
}

func newNetlifySite(runtime *plugin.Runtime, rec *siteRecord) (*mqlNetlifySite, error) {
	build := rec.BuildSettings
	if build == nil {
		build = &buildSettingsData{}
	}

	res, err := CreateResource(runtime, "netlify.site", map[string]*llx.RawData{
		"id":                        llx.StringData(rec.ID),
		"name":                      llx.StringData(rec.Name),
		"state":                     llx.StringData(rec.State),
		"plan":                      llx.StringData(rec.Plan),
		"url":                       llx.StringData(rec.URL),
		"sslUrl":                    llx.StringData(rec.SslURL),
		"adminUrl":                  llx.StringData(rec.AdminURL),
		"customDomain":              llx.StringData(rec.CustomDomain),
		"domainAliases":             llx.ArrayData(strSliceToAny(rec.DomainAliases), types.String),
		"branchDeployCustomDomain":  llx.StringData(rec.BranchDeployCustomDomain),
		"deployPreviewCustomDomain": llx.StringData(rec.DeployPreviewCustomDomain),
		"notificationEmail":         llx.StringData(rec.NotificationEmail),
		"ssl":                       llx.BoolData(rec.Ssl),
		"forceSsl":                  llx.BoolData(rec.ForceSsl),
		"managedDns":                llx.BoolData(rec.ManagedDns),
		"passwordProtected":         llx.BoolData(rec.passwordProtected()),
		"preventNonGitProdDeploys":  llx.BoolData(rec.PreventNonGitProdDeploys),
		"buildImage":                llx.StringData(rec.BuildImage),
		"prerender":                 llx.StringData(rec.Prerender),
		"functionsRegion":           llx.StringData(rec.FunctionsRegion),
		"createdAt":                 llx.TimeDataPtr(rec.CreatedAt.Time()),
		"updatedAt":                 llx.TimeDataPtr(rec.UpdatedAt.Time()),
		"repoProvider":              llx.StringData(build.Provider),
		"repoUrl":                   llx.StringData(build.RepoURL),
		"repoBranch":                llx.StringData(build.RepoBranch),
		"repoPath":                  llx.StringData(build.RepoPath),
		"buildCommand":              llx.StringData(build.Cmd),
		"publishDirectory":          llx.StringData(build.Dir),
		"functionsDirectory":        llx.StringData(build.FunctionsDir),
		"allowedBranches":           llx.ArrayData(strSliceToAny(build.AllowedBranches), types.String),
		"publicRepo":                llx.BoolData(build.PublicRepo),
		"privateLogs":               llx.BoolData(build.PrivateLogs),
		"stopBuilds":                llx.BoolData(build.StopBuilds),
	})
	if err != nil {
		return nil, err
	}

	site := res.(*mqlNetlifySite)
	site.cacheAccountID = rec.AccountID
	site.cacheDeployKeyID = build.DeployKeyID
	return site, nil
}

// initNetlifySite resolves the site a query targets: an explicit id argument or
// the site a discovered asset is scoped to.
func initNetlifySite(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	c := netlifyConn(runtime)

	siteID := ""
	if data, ok := args["id"]; ok {
		if s, ok := data.Value.(string); ok {
			siteID = s
		}
	}
	if siteID == "" && c.Asset() != nil {
		for _, pid := range c.Asset().PlatformIds {
			if t := strings.TrimPrefix(pid, connection.PlatformIdNetlifySite); t != pid {
				siteID = t
				break
			}
		}
	}
	if siteID == "" {
		siteID = c.SiteFilter()
	}
	if siteID == "" {
		return nil, nil, errors.New("netlify.site requires a site id")
	}

	var rec siteRecord
	if err := c.Get(context.Background(), "/sites/"+url.PathEscape(siteID), nil, &rec); err != nil {
		return nil, nil, err
	}
	if rec.ID == "" {
		rec.ID = siteID
	}

	site, err := newNetlifySite(runtime, &rec)
	if err != nil {
		return nil, nil, err
	}
	return args, site, nil
}

func (s *mqlNetlifySite) id() (string, error) {
	return s.Id.Data, s.Id.Error
}

// account resolves the account the site is billed to and administered from.
func (s *mqlNetlifySite) account() (*mqlNetlifyAccount, error) {
	if s.cacheAccountID == "" {
		s.Account.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	res, err := NewResource(s.MqlRuntime, "netlify.account", map[string]*llx.RawData{
		"id": llx.StringData(s.cacheAccountID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlNetlifyAccount), nil
}

// deployKey resolves the key the build clones the repository with. A site
// building from a public repository or from manual uploads has none.
func (s *mqlNetlifySite) deployKey() (*mqlNetlifyDeployKey, error) {
	if s.cacheDeployKeyID == "" {
		s.DeployKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	res, err := NewResource(s.MqlRuntime, "netlify.deployKey", map[string]*llx.RawData{
		"id": llx.StringData(s.cacheDeployKeyID),
	})
	if err != nil {
		// A key registered by another account member is not readable with this
		// token, which is not a reason to fail the whole site.
		if connection.IsForbidden(err) || connection.IsNotFound(err) {
			s.DeployKey.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	return res.(*mqlNetlifyDeployKey), nil
}

func (s *mqlNetlifySite) environmentVariables() ([]any, error) {
	if s.cacheAccountID == "" {
		s.EnvironmentVariables = plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}
		return nil, nil
	}

	query := url.Values{}
	query.Set("site_id", s.Id.Data)
	return fetchEnvVars(s.MqlRuntime, s.cacheAccountID, s.Id.Data, query)
}

// --- build hooks ----------------------------------------------------------

type buildHookRecord struct {
	ID        string      `json:"id"`
	Title     string      `json:"title"`
	Branch    string      `json:"branch"`
	CreatedAt netlifyTime `json:"created_at"`
}

func (s *mqlNetlifySite) buildHooks() ([]any, error) {
	c := netlifyConn(s.MqlRuntime)

	records, err := connection.GetList[buildHookRecord](context.Background(), c,
		"/sites/"+url.PathEscape(s.Id.Data)+"/build_hooks", nil)
	if err != nil {
		if connection.IsForbidden(err) {
			s.BuildHooks = plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}
			return nil, nil
		}
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]
		// The hook URL is an unauthenticated deploy trigger, so it is left out
		// of the resource rather than carried into scan results.
		hook, err := CreateResource(s.MqlRuntime, "netlify.site.buildHook", map[string]*llx.RawData{
			"id":        llx.StringData(rec.ID),
			"title":     llx.StringData(rec.Title),
			"branch":    llx.StringData(rec.Branch),
			"createdAt": llx.TimeDataPtr(rec.CreatedAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, hook)
	}
	return res, nil
}

func (h *mqlNetlifySiteBuildHook) id() (string, error) {
	return h.Id.Data, h.Id.Error
}

// --- notification hooks ---------------------------------------------------

type notificationHookRecord struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Event     string      `json:"event"`
	Disabled  bool        `json:"disabled"`
	CreatedAt netlifyTime `json:"created_at"`
	UpdatedAt netlifyTime `json:"updated_at"`
}

func (s *mqlNetlifySite) notificationHooks() ([]any, error) {
	c := netlifyConn(s.MqlRuntime)

	query := url.Values{}
	query.Set("site_id", s.Id.Data)

	records, err := connection.GetList[notificationHookRecord](context.Background(), c, "/hooks", query)
	if err != nil {
		if connection.IsForbidden(err) {
			s.NotificationHooks = plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}
			return nil, nil
		}
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]
		// The hook's data carries the delivery target, which for a webhook or
		// a chat integration is a bearer secret, so it is left out.
		hook, err := CreateResource(s.MqlRuntime, "netlify.site.notificationHook", map[string]*llx.RawData{
			"id":        llx.StringData(rec.ID),
			"type":      llx.StringData(rec.Type),
			"event":     llx.StringData(rec.Event),
			"disabled":  llx.BoolData(rec.Disabled),
			"createdAt": llx.TimeDataPtr(rec.CreatedAt.Time()),
			"updatedAt": llx.TimeDataPtr(rec.UpdatedAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, hook)
	}
	return res, nil
}

func (h *mqlNetlifySiteNotificationHook) id() (string, error) {
	return h.Id.Data, h.Id.Error
}

// --- snippets -------------------------------------------------------------

type snippetRecord struct {
	ID              int64  `json:"id"`
	Title           string `json:"title"`
	General         string `json:"general"`
	GeneralPosition string `json:"general_position"`
	Goal            string `json:"goal"`
	GoalPosition    string `json:"goal_position"`
}

func (s *mqlNetlifySite) snippets() ([]any, error) {
	c := netlifyConn(s.MqlRuntime)

	records, err := connection.GetList[snippetRecord](context.Background(), c,
		"/sites/"+url.PathEscape(s.Id.Data)+"/snippets", nil)
	if err != nil {
		if connection.IsForbidden(err) {
			s.Snippets = plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}
			return nil, nil
		}
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]
		snippet, err := CreateResource(s.MqlRuntime, "netlify.site.snippet", map[string]*llx.RawData{
			"id":              llx.StringData(strconv.FormatInt(rec.ID, 10)),
			"title":           llx.StringData(rec.Title),
			"general":         llx.StringData(rec.General),
			"generalPosition": llx.StringData(rec.GeneralPosition),
			"goal":            llx.StringData(rec.Goal),
			"goalPosition":    llx.StringData(rec.GoalPosition),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, snippet)
	}
	return res, nil
}

func (s *mqlNetlifySiteSnippet) id() (string, error) {
	return s.Id.Data, s.Id.Error
}
