// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v85/github"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/github/connection"
	"go.mondoo.com/mql/v13/types"
)

// ---------- helpers ----------

// isAccessDeniedOrNotFound returns true if the error indicates the resource is
// inaccessible (404 or 403). Many GitHub Enterprise Cloud-only endpoints return
// 404 for non-enterprise orgs, or 403 if the token lacks scope.
func isAccessDeniedOrNotFound(err error) bool {
	if err == nil {
		return false
	}
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) && ghErr.Response != nil {
		switch ghErr.Response.StatusCode {
		case http.StatusNotFound, http.StatusForbidden:
			return true
		}
	}
	// Fallback for non-typed errors (e.g. body-decoded GraphQL errors that
	// surface as plain strings like "no available registrations").
	msg := err.Error()
	return strings.Contains(msg, "no available registrations")
}

// doRawJSON performs a raw GET request through the github client against the
// given relative URL and decodes the JSON body into v. Useful for endpoints not
// yet covered by go-github helpers.
func doRawJSON(ctx context.Context, client *github.Client, urlStr string, v any) (*github.Response, error) {
	req, err := client.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	return client.Do(ctx, req, v)
}

// doRawGraphQL performs a POST against the GraphQL endpoint with the given
// query and variables, decoding the response into v.
func doRawGraphQL(ctx context.Context, client *github.Client, query string, vars map[string]any, v any) (*github.Response, error) {
	body := map[string]any{"query": query}
	if vars != nil {
		body["variables"] = vars
	}
	req, err := client.NewRequest(http.MethodPost, "graphql", body)
	if err != nil {
		return nil, err
	}
	return client.Do(ctx, req, v)
}

// ---------- SAML config (GraphQL) ----------

func (g *mqlGithubOrganizationSamlConfig) id() (string, error) {
	return g.__id, nil
}

type ghSamlIdentityProvider struct {
	SsoURL          *string `json:"ssoUrl"`
	Issuer          *string `json:"issuer"`
	DigestMethod    *string `json:"digestMethod"`
	SignatureMethod *string `json:"signatureMethod"`
	IDPCertificate  *string `json:"idpCertificate"`
}

type ghSamlConfigResponse struct {
	Data struct {
		Organization struct {
			SamlIdentityProvider *ghSamlIdentityProvider `json:"samlIdentityProvider"`
		} `json:"organization"`
	} `json:"data"`
	Errors []struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"errors"`
}

func (g *mqlGithubOrganization) samlConfig() (*mqlGithubOrganizationSamlConfig, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	orgLogin := g.Login.Data

	query := `query($login: String!) {
		organization(login: $login) {
			samlIdentityProvider {
				ssoUrl
				issuer
				digestMethod
				signatureMethod
				idpCertificate
			}
		}
	}`
	var resp ghSamlConfigResponse
	_, err := doRawGraphQL(conn.Context(), conn.Client(), query, map[string]any{"login": orgLogin}, &resp)
	if err != nil {
		if isAccessDeniedOrNotFound(err) {
			log.Debug().Err(err).Msg("SAML config not accessible (requires GitHub Enterprise Cloud and admin scope)")
			g.SamlConfig.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	if len(resp.Errors) > 0 {
		// SAML SSO is not configured / not enterprise; treat as null
		log.Debug().Msgf("SAML config GraphQL errors: %s", resp.Errors[0].Message)
		g.SamlConfig.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	idp := resp.Data.Organization.SamlIdentityProvider
	enabled := idp != nil
	args := map[string]*llx.RawData{
		"__id":            llx.StringData("github.organization.samlConfig/" + orgLogin),
		"enabled":         llx.BoolData(enabled),
		"ssoUrl":          llx.StringData(""),
		"issuer":          llx.StringData(""),
		"digestMethod":    llx.StringData(""),
		"signatureMethod": llx.StringData(""),
		"idpCertificate":  llx.StringData(""),
	}
	if idp != nil {
		if idp.SsoURL != nil {
			args["ssoUrl"] = llx.StringData(*idp.SsoURL)
		}
		if idp.Issuer != nil {
			args["issuer"] = llx.StringData(*idp.Issuer)
		}
		if idp.DigestMethod != nil {
			args["digestMethod"] = llx.StringData(*idp.DigestMethod)
		}
		if idp.SignatureMethod != nil {
			args["signatureMethod"] = llx.StringData(*idp.SignatureMethod)
		}
		if idp.IDPCertificate != nil {
			args["idpCertificate"] = llx.StringData(*idp.IDPCertificate)
		}
	}
	res, err := CreateResource(g.MqlRuntime, "github.organization.samlConfig", args)
	if err != nil {
		return nil, err
	}
	return res.(*mqlGithubOrganizationSamlConfig), nil
}

// ---------- IP allow list (GraphQL) ----------

func (g *mqlGithubOrganizationIpAllowList) id() (string, error) {
	return g.__id, nil
}

func (g *mqlGithubOrganizationIpAllowListEntry) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	return "github.organization.ipAllowList.entry/" + g.Id.Data, nil
}

type ghIpAllowListEntry struct {
	ID             string     `json:"id"`
	Name           *string    `json:"name"`
	AllowListValue string     `json:"allowListValue"`
	IsActive       bool       `json:"isActive"`
	CreatedAt      *time.Time `json:"createdAt"`
	UpdatedAt      *time.Time `json:"updatedAt"`
}

type ghIpAllowListResponse struct {
	Data struct {
		Organization struct {
			IpAllowListEnabledSetting                 *string `json:"ipAllowListEnabledSetting"`
			IpAllowListForInstalledAppsEnabledSetting *string `json:"ipAllowListForInstalledAppsEnabledSetting"`
			IpAllowListEntries                        struct {
				Nodes []ghIpAllowListEntry `json:"nodes"`
			} `json:"ipAllowListEntries"`
		} `json:"organization"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (g *mqlGithubOrganization) ipAllowList() (*mqlGithubOrganizationIpAllowList, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	orgLogin := g.Login.Data

	query := `query($login: String!) {
		organization(login: $login) {
			ipAllowListEnabledSetting
			ipAllowListForInstalledAppsEnabledSetting
			ipAllowListEntries(first: 100) {
				nodes {
					id
					name
					allowListValue
					isActive
					createdAt
					updatedAt
				}
			}
		}
	}`
	var resp ghIpAllowListResponse
	_, err := doRawGraphQL(conn.Context(), conn.Client(), query, map[string]any{"login": orgLogin}, &resp)
	if err != nil {
		if isAccessDeniedOrNotFound(err) {
			g.IpAllowList.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	if len(resp.Errors) > 0 {
		log.Debug().Msgf("ip allow list GraphQL errors: %s", resp.Errors[0].Message)
		g.IpAllowList.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	enabled := resp.Data.Organization.IpAllowListEnabledSetting != nil &&
		strings.EqualFold(*resp.Data.Organization.IpAllowListEnabledSetting, "ENABLED")
	enabledForApps := resp.Data.Organization.IpAllowListForInstalledAppsEnabledSetting != nil &&
		strings.EqualFold(*resp.Data.Organization.IpAllowListForInstalledAppsEnabledSetting, "ENABLED")

	entries := []any{}
	for _, e := range resp.Data.Organization.IpAllowListEntries.Nodes {
		var name string
		if e.Name != nil {
			name = *e.Name
		}
		entryArgs := map[string]*llx.RawData{
			"id":           llx.StringData(e.ID),
			"name":         llx.StringData(name),
			"allowedValue": llx.StringData(e.AllowListValue),
			"isActive":     llx.BoolData(e.IsActive),
			"createdAt":    llx.TimeDataPtr(e.CreatedAt),
			"updatedAt":    llx.TimeDataPtr(e.UpdatedAt),
		}
		r, err := CreateResource(g.MqlRuntime, "github.organization.ipAllowList.entry", entryArgs)
		if err != nil {
			return nil, err
		}
		entries = append(entries, r)
	}

	res, err := CreateResource(g.MqlRuntime, "github.organization.ipAllowList", map[string]*llx.RawData{
		"__id":                    llx.StringData("github.organization.ipAllowList/" + orgLogin),
		"enabled":                 llx.BoolData(enabled),
		"enabledForInstalledApps": llx.BoolData(enabledForApps),
		"entries":                 llx.ArrayData(entries, types.Resource("github.organization.ipAllowList.entry")),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGithubOrganizationIpAllowList), nil
}

// ---------- Custom org roles ----------

func (g *mqlGithubOrganizationCustomRole) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	return "github.organization.customRole/" + strconv.FormatInt(g.Id.Data, 10), nil
}

func (g *mqlGithubOrganization) customRoles() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	orgLogin := g.Login.Data

	roles, _, err := conn.Client().Organizations.ListRoles(conn.Context(), orgLogin)
	if err != nil {
		if isAccessDeniedOrNotFound(err) {
			log.Debug().Err(err).Msg("Custom org roles not accessible (requires GitHub Enterprise Cloud)")
			return nil, nil
		}
		return nil, err
	}
	if roles == nil {
		return nil, nil
	}

	res := []any{}
	for _, role := range roles.CustomRepoRoles {
		var id int64
		if role.ID != nil {
			id = *role.ID
		}
		r, err := CreateResource(g.MqlRuntime, "github.organization.customRole", map[string]*llx.RawData{
			"id":          llx.IntData(id),
			"name":        llx.StringDataPtr(role.Name),
			"description": llx.StringDataPtr(role.Description),
			"baseRole":    llx.StringDataPtr(role.BaseRole),
			"permissions": llx.ArrayData(convert.SliceAnyToInterface[string](role.Permissions), types.String),
			"source":      llx.StringData(""),
			"createdAt":   llx.TimeDataPtr(githubTimestamp(role.CreatedAt)),
			"updatedAt":   llx.TimeDataPtr(githubTimestamp(role.UpdatedAt)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

// ---------- OAuth apps / SAML SSO credential authorizations ----------

type mqlGithubOrganizationOauthAppInternal struct {
	cacheOrgLogin  string
	cacheUserLogin string
}

func (g *mqlGithubOrganizationOauthApp) id() (string, error) {
	if g.CredentialId.Error != nil {
		return "", g.CredentialId.Error
	}
	return "github.organization.oauthApp/" + strconv.FormatInt(g.CredentialId.Data, 10), nil
}

func (g *mqlGithubOrganization) oauthApps() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	orgLogin := g.Login.Data

	opts := &github.CredentialAuthorizationsListOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}
	var allCreds []*github.CredentialAuthorization
	for {
		creds, resp, err := conn.Client().Organizations.ListCredentialAuthorizations(conn.Context(), orgLogin, opts)
		if err != nil {
			if isAccessDeniedOrNotFound(err) {
				log.Debug().Err(err).Msg("Credential authorizations not accessible (requires GitHub Enterprise Cloud and SAML SSO)")
				return nil, nil
			}
			return nil, err
		}
		allCreds = append(allCreds, creds...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	res := []any{}
	for _, c := range allCreds {
		var credID int64
		if c.CredentialID != nil {
			credID = *c.CredentialID
		}
		args := map[string]*llx.RawData{
			"login":                         llx.StringDataPtr(c.Login),
			"credentialId":                  llx.IntData(credID),
			"credentialType":                llx.StringDataPtr(c.CredentialType),
			"tokenLastEight":                llx.StringDataPtr(c.TokenLastEight),
			"scopes":                        llx.ArrayData(convert.SliceAnyToInterface[string](c.Scopes), types.String),
			"authorizedAt":                  llx.TimeDataPtr(githubTimestamp(c.CredentialAuthorizedAt)),
			"lastAccessedAt":                llx.TimeDataPtr(githubTimestamp(c.CredentialAccessedAt)),
			"authorizedCredentialTitle":     llx.StringDataPtr(c.AuthorizedCredentialTitle),
			"authorizedCredentialNote":      llx.StringDataPtr(c.AuthorizedCredentialNote),
			"authorizedCredentialExpiresAt": llx.TimeDataPtr(githubTimestamp(c.AuthorizedCredentialExpiresAt)),
			"fingerprint":                   llx.StringDataPtr(c.Fingerprint),
		}
		r, err := CreateResource(g.MqlRuntime, "github.organization.oauthApp", args)
		if err != nil {
			return nil, err
		}
		oauthApp := r.(*mqlGithubOrganizationOauthApp)
		oauthApp.cacheOrgLogin = orgLogin
		if c.Login != nil {
			oauthApp.cacheUserLogin = *c.Login
		}
		res = append(res, oauthApp)
	}
	return res, nil
}

func (g *mqlGithubOrganizationOauthApp) organization() (*mqlGithubOrganization, error) {
	if g.cacheOrgLogin == "" {
		g.Organization.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	o, err := NewResource(g.MqlRuntime, "github.organization", map[string]*llx.RawData{
		"login": llx.StringData(g.cacheOrgLogin),
	})
	if err != nil {
		return nil, err
	}
	return o.(*mqlGithubOrganization), nil
}

func (g *mqlGithubOrganizationOauthApp) user() (*mqlGithubUser, error) {
	if g.cacheUserLogin == "" {
		g.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	u, err := NewResource(g.MqlRuntime, "github.user", map[string]*llx.RawData{
		"login": llx.StringData(g.cacheUserLogin),
	})
	if err != nil {
		return nil, err
	}
	return u.(*mqlGithubUser), nil
}

// ---------- Personal access tokens (fine-grained, org-level) ----------

func (g *mqlGithubOrganizationPersonalAccessToken) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	return "github.organization.personalAccessToken/" + strconv.FormatInt(g.Id.Data, 10), nil
}

func (g *mqlGithubOrganizationPersonalAccessToken) owner() (*mqlGithubUser, error) {
	if g.OwnerLogin.Error != nil {
		return nil, g.OwnerLogin.Error
	}
	if g.OwnerLogin.Data == "" {
		g.Owner.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	u, err := NewResource(g.MqlRuntime, "github.user", map[string]*llx.RawData{
		"login": llx.StringData(g.OwnerLogin.Data),
	})
	if err != nil {
		return nil, err
	}
	return u.(*mqlGithubUser), nil
}

func (g *mqlGithubOrganization) personalAccessTokens() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	orgLogin := g.Login.Data

	opts := &github.ListFineGrainedPATOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}
	var allPATs []*github.PersonalAccessToken
	for {
		pats, resp, err := conn.Client().Organizations.ListFineGrainedPersonalAccessTokens(conn.Context(), orgLogin, opts)
		if err != nil {
			if isAccessDeniedOrNotFound(err) {
				log.Debug().Err(err).Msg("Fine-grained PATs not accessible (requires GitHub Enterprise Cloud)")
				return nil, nil
			}
			return nil, err
		}
		allPATs = append(allPATs, pats...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	res := []any{}
	for _, p := range allPATs {
		permsDict, _ := convert.JsonToDict(p.Permissions)

		var ownerLogin string
		if p.Owner != nil {
			ownerLogin = p.Owner.GetLogin()
		}
		var idVal int64
		if p.ID != nil {
			idVal = *p.ID
		}
		args := map[string]*llx.RawData{
			"id":                  llx.IntData(idVal),
			"tokenId":             llx.IntDataPtr(p.TokenID),
			"tokenName":           llx.StringDataPtr(p.TokenName),
			"ownerLogin":          llx.StringData(ownerLogin),
			"repositorySelection": llx.StringDataPtr(p.RepositorySelection),
			"permissions":         llx.MapData(permsDict, types.Any),
			"accessGrantedAt":     llx.TimeDataPtr(githubTimestamp(p.AccessGrantedAt)),
			"expired":             llx.BoolData(p.GetTokenExpired()),
			"expiresAt":           llx.TimeDataPtr(githubTimestamp(p.TokenExpiresAt)),
			"lastUsedAt":          llx.TimeDataPtr(githubTimestamp(p.TokenLastUsedAt)),
		}
		r, err := CreateResource(g.MqlRuntime, "github.organization.personalAccessToken", args)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

// ---------- Audit log streaming destination ----------

func (g *mqlGithubOrganizationAuditLogStreamConfig) id() (string, error) {
	return g.__id, nil
}

// ghAuditLogStream represents the response of the (private) GitHub Enterprise
// Cloud audit log stream config endpoint.
// API docs: GET /orgs/{org}/audit-log/stream-config (Enterprise Cloud only).
type ghAuditLogStream struct {
	ID           int64      `json:"id"`
	StreamType   string     `json:"stream_type"`
	StreamPaused bool       `json:"stream_paused"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

func (g *mqlGithubOrganization) auditLogStreamConfig() (*mqlGithubOrganizationAuditLogStreamConfig, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	orgLogin := g.Login.Data

	urlStr := fmt.Sprintf("orgs/%s/audit-log/stream-config", orgLogin)
	var stream ghAuditLogStream
	resp, err := doRawJSON(conn.Context(), conn.Client(), urlStr, &stream)
	if err != nil {
		// 404 means no stream configured or org isn't enterprise; return null
		if isAccessDeniedOrNotFound(err) || (resp != nil && resp.StatusCode == http.StatusNotFound) {
			res, cerr := CreateResource(g.MqlRuntime, "github.organization.auditLogStreamConfig", map[string]*llx.RawData{
				"__id":                llx.StringData("github.organization.auditLogStreamConfig/" + orgLogin),
				"enabled":             llx.BoolData(false),
				"streamType":          llx.StringData(""),
				"streamId":            llx.IntData(0),
				"enabledStreamPaused": llx.BoolData(false),
				"createdAt":           llx.TimeDataPtr(nil),
				"updatedAt":           llx.TimeDataPtr(nil),
			})
			if cerr != nil {
				return nil, cerr
			}
			return res.(*mqlGithubOrganizationAuditLogStreamConfig), nil
		}
		return nil, err
	}

	args := map[string]*llx.RawData{
		"__id":                llx.StringData("github.organization.auditLogStreamConfig/" + orgLogin),
		"enabled":             llx.BoolData(stream.ID != 0),
		"streamType":          llx.StringData(stream.StreamType),
		"streamId":            llx.IntData(stream.ID),
		"enabledStreamPaused": llx.BoolData(stream.StreamPaused),
		"createdAt":           llx.TimeDataPtr(stream.CreatedAt),
		"updatedAt":           llx.TimeDataPtr(stream.UpdatedAt),
	}

	res, err := CreateResource(g.MqlRuntime, "github.organization.auditLogStreamConfig", args)
	if err != nil {
		return nil, err
	}
	return res.(*mqlGithubOrganizationAuditLogStreamConfig), nil
}

// ---------- Installation augmentation ----------

type mqlGithubInstallationInternal struct {
	cacheOrgLogin       string
	cacheInstallationID int64
}

func (g *mqlGithubInstallation) repositories() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	// only meaningful for "selected" repository_selection
	sel := ""
	if g.RepositorySelection.IsSet() && g.RepositorySelection.Error == nil {
		sel = g.RepositorySelection.Data
	}
	if sel != "" && sel != "selected" {
		return []any{}, nil
	}

	// We need an installation access token to call /installation/repositories.
	// That requires an app-authenticated client which the connection provides
	// when running with --app-id; otherwise this endpoint isn't accessible
	// using a personal access token.
	if g.cacheOrgLogin == "" {
		// fall back to listing accessible repos via the token's default scope;
		// without app auth we can't enumerate the installation's repos.
		return []any{}, nil
	}

	// Try /installation/repositories first (requires app authentication).
	// If the token can't, it'll 401/403 and we fall back to []any{}.
	type installationRepoResp struct {
		TotalCount   int                  `json:"total_count"`
		Repositories []*github.Repository `json:"repositories"`
	}
	var allRepos []*github.Repository
	page := 1
	for {
		var r installationRepoResp
		resp, err := doRawJSON(conn.Context(), conn.Client(), fmt.Sprintf("installation/repositories?per_page=%d&page=%d", paginationPerPage, page), &r)
		if err != nil {
			if isAccessDeniedOrNotFound(err) {
				return []any{}, nil
			}
			return nil, err
		}
		allRepos = append(allRepos, r.Repositories...)
		if resp == nil || resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	res := make([]any, 0, len(allRepos))
	for _, repo := range allRepos {
		mqlRepo, err := newMqlGithubRepository(g.MqlRuntime, repo)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRepo)
	}
	return res, nil
}

// ---------- Deploy keys (repo + user) ----------

type mqlGithubDeployKeyInternal struct {
	cacheRepoOwner    string
	cacheRepoName     string
	cacheAddedByLogin string
}

func (g *mqlGithubDeployKey) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	return "github.deployKey/" + strconv.FormatInt(g.Id.Data, 10), nil
}

// keyAge returns the age of a key in days based on a CreatedAt timestamp; -1 if
// CreatedAt is unknown.
func keyAgeInDays(createdAt *github.Timestamp) int64 {
	if createdAt == nil || createdAt.IsZero() {
		return -1
	}
	return int64(time.Since(createdAt.Time).Hours() / 24)
}

func newMqlDeployKey(runtime *plugin.Runtime, k *github.Key, repoOwner, repoName string) (*mqlGithubDeployKey, error) {
	var idVal int64
	if k.ID != nil {
		idVal = *k.ID
	}
	args := map[string]*llx.RawData{
		"id":        llx.IntData(idVal),
		"title":     llx.StringDataPtr(k.Title),
		"key":       llx.StringDataPtr(k.Key),
		"readOnly":  llx.BoolData(k.GetReadOnly()),
		"verified":  llx.BoolData(k.GetVerified()),
		"createdAt": llx.TimeDataPtr(githubTimestamp(k.CreatedAt)),
		"lastUsed":  llx.TimeDataPtr(githubTimestamp(k.LastUsed)),
		"ageInDays": llx.IntData(keyAgeInDays(k.CreatedAt)),
	}
	r, err := CreateResource(runtime, "github.deployKey", args)
	if err != nil {
		return nil, err
	}
	dk := r.(*mqlGithubDeployKey)
	dk.cacheRepoOwner = repoOwner
	dk.cacheRepoName = repoName
	if k.AddedBy != nil {
		dk.cacheAddedByLogin = *k.AddedBy
	}
	if repoOwner == "" || repoName == "" {
		dk.Repository.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return dk, nil
}

func (g *mqlGithubDeployKey) addedBy() (*mqlGithubUser, error) {
	if g.cacheAddedByLogin == "" {
		g.AddedBy.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	u, err := NewResource(g.MqlRuntime, "github.user", map[string]*llx.RawData{
		"login": llx.StringData(g.cacheAddedByLogin),
	})
	if err != nil {
		return nil, err
	}
	return u.(*mqlGithubUser), nil
}

func (g *mqlGithubDeployKey) repository() (*mqlGithubRepository, error) {
	if g.cacheRepoOwner == "" || g.cacheRepoName == "" {
		g.Repository.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	repo, _, err := conn.Client().Repositories.Get(conn.Context(), g.cacheRepoOwner, g.cacheRepoName)
	if err != nil {
		if isAccessDeniedOrNotFound(err) {
			g.Repository.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	return newMqlGithubRepository(g.MqlRuntime, repo)
}

func (g *mqlGithubRepository) deployKeys() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data
	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	owner := g.Owner.Data
	if owner.Login.Error != nil {
		return nil, owner.Login.Error
	}
	ownerLogin := owner.Login.Data

	listOpts := &github.ListOptions{PerPage: paginationPerPage}
	var allKeys []*github.Key
	for {
		keys, resp, err := conn.Client().Repositories.ListKeys(conn.Context(), ownerLogin, repoName, listOpts)
		if err != nil {
			if isAccessDeniedOrNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		allKeys = append(allKeys, keys...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	res := make([]any, 0, len(allKeys))
	for _, k := range allKeys {
		dk, err := newMqlDeployKey(g.MqlRuntime, k, ownerLogin, repoName)
		if err != nil {
			return nil, err
		}
		res = append(res, dk)
	}
	return res, nil
}

func (g *mqlGithubUser) publicKeys() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	userLogin := g.Login.Data

	listOpts := &github.ListOptions{PerPage: paginationPerPage}
	var allKeys []*github.Key
	for {
		keys, resp, err := conn.Client().Users.ListKeys(conn.Context(), userLogin, listOpts)
		if err != nil {
			if isAccessDeniedOrNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		allKeys = append(allKeys, keys...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	res := make([]any, 0, len(allKeys))
	for _, k := range allKeys {
		pk, err := newMqlPublicKey(g.MqlRuntime, k, userLogin)
		if err != nil {
			return nil, err
		}
		res = append(res, pk)
	}
	return res, nil
}

// ---------- User public keys ----------

type mqlGithubPublicKeyInternal struct {
	cacheUserLogin string
}

func (g *mqlGithubPublicKey) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	return "github.publicKey/" + strconv.FormatInt(g.Id.Data, 10), nil
}

func newMqlPublicKey(runtime *plugin.Runtime, k *github.Key, userLogin string) (*mqlGithubPublicKey, error) {
	var idVal int64
	if k.ID != nil {
		idVal = *k.ID
	}
	args := map[string]*llx.RawData{
		"id":        llx.IntData(idVal),
		"title":     llx.StringDataPtr(k.Title),
		"key":       llx.StringDataPtr(k.Key),
		"verified":  llx.BoolData(k.GetVerified()),
		"createdAt": llx.TimeDataPtr(githubTimestamp(k.CreatedAt)),
		"ageInDays": llx.IntData(keyAgeInDays(k.CreatedAt)),
	}
	r, err := CreateResource(runtime, "github.publicKey", args)
	if err != nil {
		return nil, err
	}
	pk := r.(*mqlGithubPublicKey)
	pk.cacheUserLogin = userLogin
	if userLogin == "" {
		pk.User.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return pk, nil
}

func (g *mqlGithubPublicKey) user() (*mqlGithubUser, error) {
	if g.cacheUserLogin == "" {
		g.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	u, err := NewResource(g.MqlRuntime, "github.user", map[string]*llx.RawData{
		"login": llx.StringData(g.cacheUserLogin),
	})
	if err != nil {
		return nil, err
	}
	return u.(*mqlGithubUser), nil
}

// ---------- CODEOWNERS ----------

func (g *mqlGithubRepositoryCodeowners) id() (string, error) {
	return g.__id, nil
}

func (g *mqlGithubCodeownersRule) id() (string, error) {
	if g.LineNumber.Error != nil {
		return "", g.LineNumber.Error
	}
	if g.Pattern.Error != nil {
		return "", g.Pattern.Error
	}
	return fmt.Sprintf("github.codeowners.rule/%s/%d", g.Pattern.Data, g.LineNumber.Data), nil
}

// codeownersCandidatePaths returns the locations CODEOWNERS may live at, in
// resolution priority order.
var codeownersCandidatePaths = []string{
	".github/CODEOWNERS",
	"CODEOWNERS",
	"docs/CODEOWNERS",
}

// parseCodeowners parses the contents of a CODEOWNERS file and returns the
// rules as raw struct values; empty/comment lines are skipped.
type codeownersRule struct {
	pattern    string
	owners     []string
	lineNumber int
}

func parseCodeowners(content string) []codeownersRule {
	rules := []codeownersRule{}
	for i, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 1 {
			continue
		}
		rule := codeownersRule{
			pattern:    fields[0],
			owners:     fields[1:],
			lineNumber: i + 1,
		}
		rules = append(rules, rule)
	}
	return rules
}

func (g *mqlGithubRepository) codeowners() (*mqlGithubRepositoryCodeowners, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data
	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	owner := g.Owner.Data
	if owner.Login.Error != nil {
		return nil, owner.Login.Error
	}
	ownerLogin := owner.Login.Data

	resID := llx.StringData(fmt.Sprintf("github.repository.codeowners/%s/%s", ownerLogin, repoName))

	var foundPath, content string
	for _, p := range codeownersCandidatePaths {
		fc, _, _, err := conn.Client().Repositories.GetContents(conn.Context(), ownerLogin, repoName, p, nil)
		if err != nil {
			if isAccessDeniedOrNotFound(err) {
				continue
			}
			return nil, err
		}
		if fc == nil {
			continue
		}
		c, err := fc.GetContent()
		if err != nil {
			return nil, err
		}
		foundPath = p
		content = c
		break
	}

	rules := []any{}
	for _, r := range parseCodeowners(content) {
		rr, err := CreateResource(g.MqlRuntime, "github.codeowners.rule", map[string]*llx.RawData{
			"pattern":    llx.StringData(r.pattern),
			"owners":     llx.ArrayData(convert.SliceAnyToInterface[string](r.owners), types.String),
			"lineNumber": llx.IntData(int64(r.lineNumber)),
		})
		if err != nil {
			return nil, err
		}
		rules = append(rules, rr)
	}

	res, err := CreateResource(g.MqlRuntime, "github.repository.codeowners", map[string]*llx.RawData{
		"__id":    resID,
		"path":    llx.StringData(foundPath),
		"exists":  llx.BoolData(foundPath != ""),
		"content": llx.StringData(content),
		"rules":   llx.ArrayData(rules, types.Resource("github.codeowners.rule")),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGithubRepositoryCodeowners), nil
}
