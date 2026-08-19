// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/hcp/connection"
	"go.mondoo.com/mql/v13/types"
)

// ---------------------------------------------------------------------------
// decoding helpers
// ---------------------------------------------------------------------------

// tfeTime decodes a JSON:API timestamp. A null, an empty string, or a value the
// API formats unexpectedly yields no time at all rather than the zero instant,
// which would otherwise be reported as 1 January year 1 as though it were a
// real date. A malformed timestamp is treated the same way instead of failing
// the whole record, so one odd value cannot blind an entire collection.
type tfeTime struct {
	Time *time.Time
}

func (t *tfeTime) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		t.Time = nil
		return nil
	}
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Time = nil
		return nil
	}
	if raw == "" {
		t.Time = nil
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Time = nil
		return nil
	}
	t.Time = &parsed
	return nil
}

// derefStr dereferences an optional string, yielding the empty string for a
// null value.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ---------------------------------------------------------------------------
// API record shapes
// ---------------------------------------------------------------------------

type tfeOrganizationAttrs struct {
	Name                       string  `json:"name"`
	ExternalID                 string  `json:"external-id"`
	Email                      string  `json:"email"`
	CreatedAt                  tfeTime `json:"created-at"`
	CollaboratorAuthPolicy     string  `json:"collaborator-auth-policy"`
	TwoFactorConformant        bool    `json:"two-factor-conformant"`
	SamlEnabled                bool    `json:"saml-enabled"`
	OwnersTeamSamlRoleID       *string `json:"owners-team-saml-role-id"`
	SessionTimeout             *int64  `json:"session-timeout"`
	SessionRemember            *int64  `json:"session-remember"`
	CostEstimationEnabled      bool    `json:"cost-estimation-enabled"`
	AssessmentsEnforced        bool    `json:"assessments-enforced"`
	AllowForceDeleteWorkspaces bool    `json:"allow-force-delete-workspaces"`
	DefaultExecutionMode       string  `json:"default-execution-mode"`
	PlanExpired                bool    `json:"plan-expired"`
}

type tfeVCSRepo struct {
	Identifier        string `json:"identifier"`
	DisplayIdentifier string `json:"display-identifier"`
	Branch            string `json:"branch"`
	ServiceProvider   string `json:"service-provider"`
	IngressSubmodules bool   `json:"ingress-submodules"`
}

type tfeWorkspaceAttrs struct {
	Name                       string      `json:"name"`
	Description                *string     `json:"description"`
	ExecutionMode              string      `json:"execution-mode"`
	AutoApply                  bool        `json:"auto-apply"`
	AutoApplyRunTrigger        bool        `json:"auto-apply-run-trigger"`
	TerraformVersion           string      `json:"terraform-version"`
	WorkingDirectory           *string     `json:"working-directory"`
	Locked                     bool        `json:"locked"`
	VCSRepo                    *tfeVCSRepo `json:"vcs-repo"`
	SpeculativeEnabled         bool        `json:"speculative-enabled"`
	GlobalRemoteState          bool        `json:"global-remote-state"`
	AllowDestroyPlan           bool        `json:"allow-destroy-plan"`
	FileTriggersEnabled        bool        `json:"file-triggers-enabled"`
	QueueAllRuns               bool        `json:"queue-all-runs"`
	StructuredRunOutputEnabled bool        `json:"structured-run-output-enabled"`
	AssessmentsEnabled         bool        `json:"assessments-enabled"`
	ResourceCount              int64       `json:"resource-count"`
	TagNames                   []string    `json:"tag-names"`
	CreatedAt                  tfeTime     `json:"created-at"`
	UpdatedAt                  tfeTime     `json:"updated-at"`
}

type tfeTeamAccessAttrs struct {
	Access           string `json:"access"`
	Runs             string `json:"runs"`
	Variables        string `json:"variables"`
	StateVersions    string `json:"state-versions"`
	SentinelMocks    string `json:"sentinel-mocks"`
	WorkspaceLocking bool   `json:"workspace-locking"`
	RunTasks         bool   `json:"run-tasks"`
}

type tfeVariableAttrs struct {
	Key         string  `json:"key"`
	Category    string  `json:"category"`
	Sensitive   bool    `json:"sensitive"`
	HCL         bool    `json:"hcl"`
	Description *string `json:"description"`
}

type tfeTeamOrgAccess struct {
	ManagePolicies           bool `json:"manage-policies"`
	ManagePolicyOverrides    bool `json:"manage-policy-overrides"`
	ManageWorkspaces         bool `json:"manage-workspaces"`
	ManageVCSSettings        bool `json:"manage-vcs-settings"`
	ManageMembership         bool `json:"manage-membership"`
	ManageTeams              bool `json:"manage-teams"`
	ManageOrganizationAccess bool `json:"manage-organization-access"`
	ManageProjects           bool `json:"manage-projects"`
	ManageRunTasks           bool `json:"manage-run-tasks"`
	ManageAgentPools         bool `json:"manage-agent-pools"`
	ManageProviders          bool `json:"manage-providers"`
	ManageModules            bool `json:"manage-modules"`
	ReadWorkspaces           bool `json:"read-workspaces"`
	ReadProjects             bool `json:"read-projects"`
	AccessSecretTeams        bool `json:"access-secret-teams"`
}

type tfeTeamAttrs struct {
	Name                       string            `json:"name"`
	UsersCount                 int64             `json:"users-count"`
	Visibility                 string            `json:"visibility"`
	SSOTeamID                  *string           `json:"sso-team-id"`
	AllowMemberTokenManagement bool              `json:"allow-member-token-management"`
	OrganizationAccess         *tfeTeamOrgAccess `json:"organization-access"`
}

type tfeTeamTokenAttrs struct {
	Description *string `json:"description"`
	CreatedAt   tfeTime `json:"created-at"`
	LastUsedAt  tfeTime `json:"last-used-at"`
	ExpiredAt   tfeTime `json:"expired-at"`
}

type tfePolicySetAttrs struct {
	Name           string  `json:"name"`
	Description    *string `json:"description"`
	Kind           string  `json:"kind"`
	Global         bool    `json:"global"`
	PolicyCount    int64   `json:"policy-count"`
	WorkspaceCount int64   `json:"workspace-count"`
	Versioned      bool    `json:"versioned"`
	PoliciesPath   *string `json:"policies-path"`
	AgentEnabled   bool    `json:"agent-enabled"`
	Overridable    *bool   `json:"overridable"`
	CreatedAt      tfeTime `json:"created-at"`
	UpdatedAt      tfeTime `json:"updated-at"`
}

// tfePolicyEnforcement is one entry of the legacy per-file enforcement list a
// Sentinel policy carries when the API does not report a top-level
// enforcement-level.
type tfePolicyEnforcement struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
}

type tfePolicyAttrs struct {
	Name             string                 `json:"name"`
	Description      *string                `json:"description"`
	Kind             string                 `json:"kind"`
	EnforcementLevel string                 `json:"enforcement-level"`
	Enforce          []tfePolicyEnforcement `json:"enforce"`
	PolicySetCount   int64                  `json:"policy-set-count"`
	UpdatedAt        tfeTime                `json:"updated-at"`
}

type tfeAgentPoolAttrs struct {
	Name               string  `json:"name"`
	AgentCount         int64   `json:"agent-count"`
	OrganizationScoped bool    `json:"organization-scoped"`
	CreatedAt          tfeTime `json:"created-at"`
}

// ---------------------------------------------------------------------------
// derived predicates
// ---------------------------------------------------------------------------

// terraformTwoFactorRequired reports whether an organization mandates
// two-factor authentication for its members. HCP Terraform expresses this as
// the collaborator authentication policy rather than as a boolean.
func terraformTwoFactorRequired(collaboratorAuthPolicy string) bool {
	return collaboratorAuthPolicy == "two_factor_mandatory"
}

// terraformVCSDriven reports whether runs enter a workspace from a connected
// VCS repository. A workspace with no repository is driven by the CLI or the
// API instead.
func terraformVCSDriven(repo *tfeVCSRepo) bool {
	if repo == nil {
		return false
	}
	return repo.Identifier != "" || repo.DisplayIdentifier != ""
}

// terraformVCSIdentifier returns the repository backing a workspace, preferring
// the canonical identifier and falling back to the display identifier some
// VCS providers report instead.
func terraformVCSIdentifier(repo *tfeVCSRepo) string {
	if repo == nil {
		return ""
	}
	if repo.Identifier != "" {
		return repo.Identifier
	}
	return repo.DisplayIdentifier
}

// terraformCanApply reports whether a team access grant lets the team confirm
// and apply a plan, turning a proposed change into a real one. The fixed write
// and admin roles carry it; a custom role carries it only when its run
// permission is apply.
func terraformCanApply(access, runs string) bool {
	switch access {
	case "write", "admin":
		return true
	case "custom":
		return runs == "apply"
	}
	return false
}

// terraformEnforcementLevel resolves a policy's enforcement level. Newer API
// versions report it directly; older Sentinel policies report it only as the
// mode of the first entry in the per-file enforce list.
func terraformEnforcementLevel(level string, enforce []tfePolicyEnforcement) string {
	if level != "" {
		return level
	}
	for _, e := range enforce {
		if e.Mode != "" {
			return e.Mode
		}
	}
	return ""
}

// terraformPolicyBlocking reports whether a failure of the policy stops the
// apply with no override path: hard-mandatory for Sentinel, mandatory for OPA.
// Advisory only warns, and soft-mandatory can be overridden.
func terraformPolicyBlocking(enforcementLevel string) bool {
	return enforcementLevel == "hard-mandatory" || enforcementLevel == "mandatory"
}

// ---------------------------------------------------------------------------
// internal caches
// ---------------------------------------------------------------------------

type mqlHcpTerraformWorkspaceInternal struct {
	cacheOrgName     string
	cacheAgentPoolID string
}

type mqlHcpTerraformTeamAccessInternal struct {
	cacheTeamID      string
	cacheWorkspaceID string
}

type mqlHcpTerraformVariableInternal struct {
	cacheWorkspaceID string
}

type mqlHcpTerraformTeamInternal struct {
	cacheOrgName string
}

type mqlHcpTerraformTeamTokenInternal struct {
	cacheTeamID string
}

type mqlHcpTerraformPolicySetInternal struct {
	cacheOrgName      string
	cachePolicyIDs    []string
	cacheWorkspaceIDs []string
}

type mqlHcpTerraformPolicyInternal struct {
	cacheOrgName string
}

type mqlHcpTerraformAgentPoolInternal struct {
	cacheOrgName             string
	cacheAllowedWorkspaceIDs []string
}

type mqlHcpTerraformOrganizationInternal struct {
	cacheOwnersTeamID string
}

// ---------------------------------------------------------------------------
// shared plumbing
// ---------------------------------------------------------------------------

// terraformClient returns the HCP Terraform API client for the runtime. It
// fails when no API token is configured rather than reporting an empty estate.
func terraformClient(runtime *plugin.Runtime) (*connection.TfeClient, error) {
	return hcpConn(runtime).TerraformClient()
}

// terraformCtx is the context every HCP Terraform request is issued under.
func terraformCtx() context.Context {
	return context.Background()
}

// relOneID returns the id of a record's to-one relationship, or the empty
// string when it carries none.
func relOneID(rec connection.TfeRecord, name string) string {
	if ref := rec.Rel(name).One(); ref != nil {
		return ref.ID
	}
	return ""
}

// relManyIDs returns the ids of a record's to-many relationship.
func relManyIDs(rec connection.TfeRecord, name string) []string {
	refs := rec.Rel(name).Many()
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		out = append(out, ref.ID)
	}
	return out
}

// resolveWorkspaceRefs turns a list of workspace ids into typed workspace
// resources. A workspace the token cannot read is skipped with a warning that
// names it, so the shortened list is attributable; any other failure is
// returned rather than silently shortening the list.
func resolveWorkspaceRefs(runtime *plugin.Runtime, ids []string) ([]any, error) {
	out := []any{}
	for _, id := range ids {
		if id == "" {
			continue
		}
		res, err := NewResource(runtime, "hcp.terraform.workspace", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			if connection.IsTfeUnavailable(err) {
				log.Warn().Str("workspace", id).Msg("hcp: skipping unreadable HCP Terraform workspace")
				continue
			}
			return nil, err
		}
		out = append(out, res.(*mqlHcpTerraformWorkspace))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// hcp.terraform.organization
// ---------------------------------------------------------------------------

// terraformOrganizations lists the HCP Terraform organizations the configured
// API token can reach, or just the one the connection is scoped to.
func (r *mqlHcp) terraformOrganizations() ([]any, error) {
	client, err := terraformClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	if name := hcpConn(r.MqlRuntime).TerraformOrganization(); name != "" {
		org, err := fetchMqlHcpTerraformOrganization(r.MqlRuntime, name)
		if err != nil {
			return nil, err
		}
		return []any{org}, nil
	}

	records, err := client.List(terraformCtx(), "organizations", nil)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, rec := range records {
		org, err := newMqlHcpTerraformOrganization(r.MqlRuntime, rec)
		if err != nil {
			return nil, err
		}
		out = append(out, org)
	}
	return out, nil
}

func newMqlHcpTerraformOrganization(runtime *plugin.Runtime, rec connection.TfeRecord) (*mqlHcpTerraformOrganization, error) {
	var attrs tfeOrganizationAttrs
	if err := rec.DecodeAttributes(&attrs); err != nil {
		return nil, err
	}
	// The record id is the organization name; the attributes repeat it.
	name := attrs.Name
	if name == "" {
		name = rec.ID
	}

	res, err := CreateResource(runtime, "hcp.terraform.organization", map[string]*llx.RawData{
		"__id":                       llx.StringData("hcp.terraform.organization/" + name),
		"name":                       llx.StringData(name),
		"externalId":                 llx.StringData(attrs.ExternalID),
		"email":                      llx.StringData(attrs.Email),
		"createdAt":                  llx.TimeDataPtr(attrs.CreatedAt.Time),
		"collaboratorAuthPolicy":     llx.StringData(attrs.CollaboratorAuthPolicy),
		"twoFactorRequired":          llx.BoolData(terraformTwoFactorRequired(attrs.CollaboratorAuthPolicy)),
		"twoFactorConformant":        llx.BoolData(attrs.TwoFactorConformant),
		"samlEnabled":                llx.BoolData(attrs.SamlEnabled),
		"ownersTeamSamlRoleId":       llx.StringDataPtr(attrs.OwnersTeamSamlRoleID),
		"sessionTimeoutMinutes":      llx.IntDataPtr(attrs.SessionTimeout),
		"sessionRememberMinutes":     llx.IntDataPtr(attrs.SessionRemember),
		"costEstimationEnabled":      llx.BoolData(attrs.CostEstimationEnabled),
		"assessmentsEnforced":        llx.BoolData(attrs.AssessmentsEnforced),
		"allowForceDeleteWorkspaces": llx.BoolData(attrs.AllowForceDeleteWorkspaces),
		"defaultExecutionMode":       llx.StringData(attrs.DefaultExecutionMode),
		"planExpired":                llx.BoolData(attrs.PlanExpired),
	})
	if err != nil {
		return nil, err
	}
	org := res.(*mqlHcpTerraformOrganization)
	org.cacheOwnersTeamID = relOneID(rec, "owners-team")
	return org, nil
}

// fetchMqlHcpTerraformOrganization gets a single organization by name.
func fetchMqlHcpTerraformOrganization(runtime *plugin.Runtime, name string) (*mqlHcpTerraformOrganization, error) {
	client, err := terraformClient(runtime)
	if err != nil {
		return nil, err
	}
	rec, err := client.GetOne(terraformCtx(), "organizations/"+url.PathEscape(name), nil)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("hcp.terraform.organization %q not found", name)
	}
	return newMqlHcpTerraformOrganization(runtime, *rec)
}

// initHcpTerraformOrganization resolves an organization from an explicit name
// or, when none is given, the organization the connection is scoped to.
func initHcpTerraformOrganization(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	name := ""
	if raw, ok := args["name"]; ok {
		name, _ = raw.Value.(string)
	}
	if name == "" {
		name = hcpConn(runtime).TerraformOrganization()
	}
	if name == "" {
		return nil, nil, fmt.Errorf("hcp.terraform.organization requires an organization name; pass one or set --tfe-organization")
	}
	org, err := fetchMqlHcpTerraformOrganization(runtime, name)
	if err != nil {
		return nil, nil, err
	}
	return nil, org, nil
}

// ownersTeam resolves the team whose members hold full administrative control.
func (r *mqlHcpTerraformOrganization) ownersTeam() (*mqlHcpTerraformTeam, error) {
	if r.cacheOwnersTeamID == "" {
		// The organizations list does not always carry the owners-team
		// linkage; fall back to the team named "owners", which HCP Terraform
		// creates with every organization and does not allow renaming.
		teams, err := r.teams()
		if err != nil {
			return nil, err
		}
		for _, t := range teams {
			team := t.(*mqlHcpTerraformTeam)
			if team.Name.Data == "owners" {
				return team, nil
			}
		}
		r.OwnersTeam.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "hcp.terraform.team", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheOwnersTeamID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHcpTerraformTeam), nil
}

// ---------------------------------------------------------------------------
// hcp.terraform.workspace
// ---------------------------------------------------------------------------

// workspaces lists the workspaces in the organization.
func (r *mqlHcpTerraformOrganization) workspaces() ([]any, error) {
	client, err := terraformClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	records, err := client.List(terraformCtx(),
		"organizations/"+url.PathEscape(r.Name.Data)+"/workspaces", nil)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, rec := range records {
		ws, err := newMqlHcpTerraformWorkspace(r.MqlRuntime, r.Name.Data, rec)
		if err != nil {
			return nil, err
		}
		out = append(out, ws)
	}
	return out, nil
}

func newMqlHcpTerraformWorkspace(runtime *plugin.Runtime, orgName string, rec connection.TfeRecord) (*mqlHcpTerraformWorkspace, error) {
	var attrs tfeWorkspaceAttrs
	if err := rec.DecodeAttributes(&attrs); err != nil {
		return nil, err
	}

	vcsBranch, vcsProvider, vcsSubmodules := "", "", false
	if attrs.VCSRepo != nil {
		vcsBranch = attrs.VCSRepo.Branch
		vcsProvider = attrs.VCSRepo.ServiceProvider
		vcsSubmodules = attrs.VCSRepo.IngressSubmodules
	}

	if orgName == "" {
		orgName = relOneID(rec, "organization")
	}

	res, err := CreateResource(runtime, "hcp.terraform.workspace", map[string]*llx.RawData{
		"__id":                       llx.StringData("hcp.terraform.workspace/" + rec.ID),
		"id":                         llx.StringData(rec.ID),
		"name":                       llx.StringData(attrs.Name),
		"description":                llx.StringData(derefStr(attrs.Description)),
		"executionMode":              llx.StringData(attrs.ExecutionMode),
		"autoApply":                  llx.BoolData(attrs.AutoApply),
		"autoApplyRunTrigger":        llx.BoolData(attrs.AutoApplyRunTrigger),
		"terraformVersion":           llx.StringData(attrs.TerraformVersion),
		"workingDirectory":           llx.StringData(derefStr(attrs.WorkingDirectory)),
		"locked":                     llx.BoolData(attrs.Locked),
		"vcsDriven":                  llx.BoolData(terraformVCSDriven(attrs.VCSRepo)),
		"vcsRepoIdentifier":          llx.StringData(terraformVCSIdentifier(attrs.VCSRepo)),
		"vcsRepoBranch":              llx.StringData(vcsBranch),
		"vcsRepoServiceProvider":     llx.StringData(vcsProvider),
		"vcsRepoIngressSubmodules":   llx.BoolData(vcsSubmodules),
		"speculativeEnabled":         llx.BoolData(attrs.SpeculativeEnabled),
		"globalRemoteState":          llx.BoolData(attrs.GlobalRemoteState),
		"allowDestroyPlan":           llx.BoolData(attrs.AllowDestroyPlan),
		"fileTriggersEnabled":        llx.BoolData(attrs.FileTriggersEnabled),
		"queueAllRuns":               llx.BoolData(attrs.QueueAllRuns),
		"structuredRunOutputEnabled": llx.BoolData(attrs.StructuredRunOutputEnabled),
		"assessmentsEnabled":         llx.BoolData(attrs.AssessmentsEnabled),
		"resourceCount":              llx.IntData(attrs.ResourceCount),
		"tagNames":                   llx.ArrayData(strSlice(attrs.TagNames), types.String),
		"createdAt":                  llx.TimeDataPtr(attrs.CreatedAt.Time),
		"updatedAt":                  llx.TimeDataPtr(attrs.UpdatedAt.Time),
	})
	if err != nil {
		return nil, err
	}
	ws := res.(*mqlHcpTerraformWorkspace)
	ws.cacheOrgName = orgName
	ws.cacheAgentPoolID = relOneID(rec, "agent-pool")
	return ws, nil
}

// initHcpTerraformWorkspace hydrates a single workspace by id.
func initHcpTerraformWorkspace(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	id := ""
	if raw, ok := args["id"]; ok {
		id, _ = raw.Value.(string)
	}
	if id == "" {
		return nil, nil, fmt.Errorf("hcp.terraform.workspace requires a workspace id")
	}
	client, err := terraformClient(runtime)
	if err != nil {
		return nil, nil, err
	}
	rec, err := client.GetOne(terraformCtx(), "workspaces/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, nil, err
	}
	if rec == nil {
		return nil, nil, fmt.Errorf("hcp.terraform.workspace %q not found", id)
	}
	ws, err := newMqlHcpTerraformWorkspace(runtime, "", *rec)
	if err != nil {
		return nil, nil, err
	}
	return nil, ws, nil
}

// organization resolves the organization the workspace belongs to.
func (r *mqlHcpTerraformWorkspace) organization() (*mqlHcpTerraformOrganization, error) {
	if r.cacheOrgName == "" {
		r.Organization.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "hcp.terraform.organization", map[string]*llx.RawData{
		"name": llx.StringData(r.cacheOrgName),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHcpTerraformOrganization), nil
}

// agentPool resolves the agent pool the workspace's runs execute on.
func (r *mqlHcpTerraformWorkspace) agentPool() (*mqlHcpTerraformAgentPool, error) {
	if r.cacheAgentPoolID == "" {
		r.AgentPool.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "hcp.terraform.agentPool", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheAgentPoolID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHcpTerraformAgentPool), nil
}

// remoteStateConsumers lists the workspaces allowed to read this workspace's
// state outputs. The list is empty when remote state is shared organization
// wide, because HCP Terraform does not enumerate consumers in that case.
func (r *mqlHcpTerraformWorkspace) remoteStateConsumers() ([]any, error) {
	client, err := terraformClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	records, err := client.List(terraformCtx(),
		"workspaces/"+url.PathEscape(r.Id.Data)+"/relationships/remote-state-consumers", nil)
	if err != nil {
		if connection.IsTfeUnavailable(err) {
			return []any{}, nil
		}
		return nil, err
	}
	out := []any{}
	for _, rec := range records {
		ws, err := newMqlHcpTerraformWorkspace(r.MqlRuntime, "", rec)
		if err != nil {
			return nil, err
		}
		out = append(out, ws)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// hcp.terraform.teamAccess
// ---------------------------------------------------------------------------

// teamAccess lists the team access grants on the workspace.
func (r *mqlHcpTerraformWorkspace) teamAccess() ([]any, error) {
	client, err := terraformClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("filter[workspace][id]", r.Id.Data)

	records, err := client.List(terraformCtx(), "team-workspaces", query)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, rec := range records {
		var attrs tfeTeamAccessAttrs
		if err := rec.DecodeAttributes(&attrs); err != nil {
			return nil, err
		}
		teamID := relOneID(rec, "team")
		workspaceID := relOneID(rec, "workspace")
		if workspaceID == "" {
			workspaceID = r.Id.Data
		}

		res, err := CreateResource(r.MqlRuntime, "hcp.terraform.teamAccess", map[string]*llx.RawData{
			"__id":             llx.StringData("hcp.terraform.teamAccess/" + rec.ID),
			"id":               llx.StringData(rec.ID),
			"access":           llx.StringData(attrs.Access),
			"canApply":         llx.BoolData(terraformCanApply(attrs.Access, attrs.Runs)),
			"runs":             llx.StringData(attrs.Runs),
			"variables":        llx.StringData(attrs.Variables),
			"stateVersions":    llx.StringData(attrs.StateVersions),
			"sentinelMocks":    llx.StringData(attrs.SentinelMocks),
			"workspaceLocking": llx.BoolData(attrs.WorkspaceLocking),
			"runTasks":         llx.BoolData(attrs.RunTasks),
		})
		if err != nil {
			return nil, err
		}
		access := res.(*mqlHcpTerraformTeamAccess)
		access.cacheTeamID = teamID
		access.cacheWorkspaceID = workspaceID
		out = append(out, access)
	}
	return out, nil
}

// team resolves the team the grant applies to.
func (r *mqlHcpTerraformTeamAccess) team() (*mqlHcpTerraformTeam, error) {
	if r.cacheTeamID == "" {
		r.Team.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "hcp.terraform.team", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheTeamID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHcpTerraformTeam), nil
}

// workspace resolves the workspace the grant applies to.
func (r *mqlHcpTerraformTeamAccess) workspace() (*mqlHcpTerraformWorkspace, error) {
	if r.cacheWorkspaceID == "" {
		r.Workspace.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "hcp.terraform.workspace", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheWorkspaceID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHcpTerraformWorkspace), nil
}

// ---------------------------------------------------------------------------
// hcp.terraform.variable
// ---------------------------------------------------------------------------

// variables lists the variables defined on the workspace. Variable values are
// deliberately not read, so a scan never carries a workspace credential out of
// HCP Terraform.
func (r *mqlHcpTerraformWorkspace) variables() ([]any, error) {
	client, err := terraformClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	records, err := client.List(terraformCtx(),
		"workspaces/"+url.PathEscape(r.Id.Data)+"/vars", nil)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, rec := range records {
		var attrs tfeVariableAttrs
		if err := rec.DecodeAttributes(&attrs); err != nil {
			return nil, err
		}
		res, err := CreateResource(r.MqlRuntime, "hcp.terraform.variable", map[string]*llx.RawData{
			"__id":        llx.StringData("hcp.terraform.variable/" + r.Id.Data + "/" + rec.ID),
			"id":          llx.StringData(rec.ID),
			"key":         llx.StringData(attrs.Key),
			"category":    llx.StringData(attrs.Category),
			"sensitive":   llx.BoolData(attrs.Sensitive),
			"hcl":         llx.BoolData(attrs.HCL),
			"description": llx.StringData(derefStr(attrs.Description)),
		})
		if err != nil {
			return nil, err
		}
		variable := res.(*mqlHcpTerraformVariable)
		variable.cacheWorkspaceID = r.Id.Data
		out = append(out, variable)
	}
	return out, nil
}

// workspace resolves the workspace the variable is defined on.
func (r *mqlHcpTerraformVariable) workspace() (*mqlHcpTerraformWorkspace, error) {
	if r.cacheWorkspaceID == "" {
		r.Workspace.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "hcp.terraform.workspace", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheWorkspaceID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHcpTerraformWorkspace), nil
}

// ---------------------------------------------------------------------------
// hcp.terraform.team
// ---------------------------------------------------------------------------

// teams lists the teams in the organization.
func (r *mqlHcpTerraformOrganization) teams() ([]any, error) {
	client, err := terraformClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	records, err := client.List(terraformCtx(),
		"organizations/"+url.PathEscape(r.Name.Data)+"/teams", nil)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, rec := range records {
		team, err := newMqlHcpTerraformTeam(r.MqlRuntime, r.Name.Data, rec)
		if err != nil {
			return nil, err
		}
		out = append(out, team)
	}
	return out, nil
}

func newMqlHcpTerraformTeam(runtime *plugin.Runtime, orgName string, rec connection.TfeRecord) (*mqlHcpTerraformTeam, error) {
	var attrs tfeTeamAttrs
	if err := rec.DecodeAttributes(&attrs); err != nil {
		return nil, err
	}
	// A team with no organization-access object holds none of the
	// organization-level permissions; the zero value reports exactly that.
	access := attrs.OrganizationAccess
	if access == nil {
		access = &tfeTeamOrgAccess{}
	}
	if orgName == "" {
		orgName = relOneID(rec, "organization")
	}

	res, err := CreateResource(runtime, "hcp.terraform.team", map[string]*llx.RawData{
		"__id":                        llx.StringData("hcp.terraform.team/" + rec.ID),
		"id":                          llx.StringData(rec.ID),
		"name":                        llx.StringData(attrs.Name),
		"usersCount":                  llx.IntData(attrs.UsersCount),
		"visibility":                  llx.StringData(attrs.Visibility),
		"ssoTeamId":                   llx.StringDataPtr(attrs.SSOTeamID),
		"allowMemberTokenManagement":  llx.BoolData(attrs.AllowMemberTokenManagement),
		"canManagePolicies":           llx.BoolData(access.ManagePolicies),
		"canManagePolicyOverrides":    llx.BoolData(access.ManagePolicyOverrides),
		"canManageWorkspaces":         llx.BoolData(access.ManageWorkspaces),
		"canManageVcsSettings":        llx.BoolData(access.ManageVCSSettings),
		"canManageMembership":         llx.BoolData(access.ManageMembership),
		"canManageTeams":              llx.BoolData(access.ManageTeams),
		"canManageOrganizationAccess": llx.BoolData(access.ManageOrganizationAccess),
		"canManageProjects":           llx.BoolData(access.ManageProjects),
		"canManageRunTasks":           llx.BoolData(access.ManageRunTasks),
		"canManageAgentPools":         llx.BoolData(access.ManageAgentPools),
		"canManageProviders":          llx.BoolData(access.ManageProviders),
		"canManageModules":            llx.BoolData(access.ManageModules),
		"canReadWorkspaces":           llx.BoolData(access.ReadWorkspaces),
		"canReadProjects":             llx.BoolData(access.ReadProjects),
		"canAccessSecretTeams":        llx.BoolData(access.AccessSecretTeams),
	})
	if err != nil {
		return nil, err
	}
	team := res.(*mqlHcpTerraformTeam)
	team.cacheOrgName = orgName
	return team, nil
}

// initHcpTerraformTeam hydrates a single team by id.
func initHcpTerraformTeam(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	id := ""
	if raw, ok := args["id"]; ok {
		id, _ = raw.Value.(string)
	}
	if id == "" {
		return nil, nil, fmt.Errorf("hcp.terraform.team requires a team id")
	}
	client, err := terraformClient(runtime)
	if err != nil {
		return nil, nil, err
	}
	rec, err := client.GetOne(terraformCtx(), "teams/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, nil, err
	}
	if rec == nil {
		return nil, nil, fmt.Errorf("hcp.terraform.team %q not found", id)
	}
	team, err := newMqlHcpTerraformTeam(runtime, "", *rec)
	if err != nil {
		return nil, nil, err
	}
	return nil, team, nil
}

// organization resolves the organization the team belongs to.
func (r *mqlHcpTerraformTeam) organization() (*mqlHcpTerraformOrganization, error) {
	if r.cacheOrgName == "" {
		r.Organization.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "hcp.terraform.organization", map[string]*llx.RawData{
		"name": llx.StringData(r.cacheOrgName),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHcpTerraformOrganization), nil
}

// tokens lists the API tokens issued to the team. Installations that predate
// multiple team tokens serve only the single-token endpoint, so a 404 from the
// list endpoint falls back to it rather than reporting no tokens.
func (r *mqlHcpTerraformTeam) tokens() ([]any, error) {
	client, err := terraformClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	base := "teams/" + url.PathEscape(r.Id.Data)

	records, err := client.List(terraformCtx(), base+"/authentication-tokens", nil)
	if err != nil {
		if !connection.IsTfeNotFound(err) {
			return nil, err
		}
		rec, singleErr := client.GetOne(terraformCtx(), base+"/authentication-token", nil)
		if singleErr != nil {
			if connection.IsTfeUnavailable(singleErr) {
				// No token has been issued to this team.
				return []any{}, nil
			}
			return nil, singleErr
		}
		if rec == nil {
			return []any{}, nil
		}
		records = []connection.TfeRecord{*rec}
	}

	out := []any{}
	for _, rec := range records {
		var attrs tfeTeamTokenAttrs
		if err := rec.DecodeAttributes(&attrs); err != nil {
			return nil, err
		}
		res, err := CreateResource(r.MqlRuntime, "hcp.terraform.teamToken", map[string]*llx.RawData{
			"__id":        llx.StringData("hcp.terraform.teamToken/" + r.Id.Data + "/" + rec.ID),
			"id":          llx.StringData(rec.ID),
			"description": llx.StringData(derefStr(attrs.Description)),
			"createdAt":   llx.TimeDataPtr(attrs.CreatedAt.Time),
			"lastUsedAt":  llx.TimeDataPtr(attrs.LastUsedAt.Time),
			"expiredAt":   llx.TimeDataPtr(attrs.ExpiredAt.Time),
		})
		if err != nil {
			return nil, err
		}
		token := res.(*mqlHcpTerraformTeamToken)
		token.cacheTeamID = r.Id.Data
		out = append(out, token)
	}
	return out, nil
}

// team resolves the team the token authenticates as.
func (r *mqlHcpTerraformTeamToken) team() (*mqlHcpTerraformTeam, error) {
	if r.cacheTeamID == "" {
		r.Team.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "hcp.terraform.team", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheTeamID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHcpTerraformTeam), nil
}

// ---------------------------------------------------------------------------
// hcp.terraform.policySet and hcp.terraform.policy
// ---------------------------------------------------------------------------

// policySets lists the Sentinel and OPA policy sets defined in the organization.
func (r *mqlHcpTerraformOrganization) policySets() ([]any, error) {
	client, err := terraformClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	records, err := client.List(terraformCtx(),
		"organizations/"+url.PathEscape(r.Name.Data)+"/policy-sets", nil)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, rec := range records {
		var attrs tfePolicySetAttrs
		if err := rec.DecodeAttributes(&attrs); err != nil {
			return nil, err
		}
		overridable := false
		if attrs.Overridable != nil {
			overridable = *attrs.Overridable
		}
		res, err := CreateResource(r.MqlRuntime, "hcp.terraform.policySet", map[string]*llx.RawData{
			"__id":           llx.StringData("hcp.terraform.policySet/" + rec.ID),
			"id":             llx.StringData(rec.ID),
			"name":           llx.StringData(attrs.Name),
			"description":    llx.StringData(derefStr(attrs.Description)),
			"kind":           llx.StringData(attrs.Kind),
			"global":         llx.BoolData(attrs.Global),
			"policyCount":    llx.IntData(attrs.PolicyCount),
			"workspaceCount": llx.IntData(attrs.WorkspaceCount),
			"versioned":      llx.BoolData(attrs.Versioned),
			"policiesPath":   llx.StringData(derefStr(attrs.PoliciesPath)),
			"agentEnabled":   llx.BoolData(attrs.AgentEnabled),
			"overridable":    llx.BoolData(overridable),
			"createdAt":      llx.TimeDataPtr(attrs.CreatedAt.Time),
			"updatedAt":      llx.TimeDataPtr(attrs.UpdatedAt.Time),
		})
		if err != nil {
			return nil, err
		}
		set := res.(*mqlHcpTerraformPolicySet)
		set.cacheOrgName = r.Name.Data
		set.cachePolicyIDs = relManyIDs(rec, "policies")
		set.cacheWorkspaceIDs = relManyIDs(rec, "workspaces")
		out = append(out, set)
	}
	return out, nil
}

// organization resolves the organization the policy set belongs to.
func (r *mqlHcpTerraformPolicySet) organization() (*mqlHcpTerraformOrganization, error) {
	if r.cacheOrgName == "" {
		r.Organization.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "hcp.terraform.organization", map[string]*llx.RawData{
		"name": llx.StringData(r.cacheOrgName),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHcpTerraformOrganization), nil
}

// policies lists the policies the set carries.
func (r *mqlHcpTerraformPolicySet) policies() ([]any, error) {
	client, err := terraformClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, id := range r.cachePolicyIDs {
		if id == "" {
			continue
		}
		rec, err := client.GetOne(terraformCtx(), "policies/"+url.PathEscape(id), nil)
		if err != nil {
			if connection.IsTfeUnavailable(err) {
				log.Warn().Str("policy", id).Msg("hcp: skipping unreadable HCP Terraform policy")
				continue
			}
			return nil, err
		}
		if rec == nil {
			continue
		}
		policy, err := newMqlHcpTerraformPolicy(r.MqlRuntime, r.cacheOrgName, *rec)
		if err != nil {
			return nil, err
		}
		out = append(out, policy)
	}
	return out, nil
}

// workspaces lists the workspaces the policy set is explicitly attached to.
func (r *mqlHcpTerraformPolicySet) workspaces() ([]any, error) {
	return resolveWorkspaceRefs(r.MqlRuntime, r.cacheWorkspaceIDs)
}

// policies lists the Sentinel and OPA policies defined in the organization.
func (r *mqlHcpTerraformOrganization) policies() ([]any, error) {
	client, err := terraformClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	records, err := client.List(terraformCtx(),
		"organizations/"+url.PathEscape(r.Name.Data)+"/policies", nil)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, rec := range records {
		policy, err := newMqlHcpTerraformPolicy(r.MqlRuntime, r.Name.Data, rec)
		if err != nil {
			return nil, err
		}
		out = append(out, policy)
	}
	return out, nil
}

func newMqlHcpTerraformPolicy(runtime *plugin.Runtime, orgName string, rec connection.TfeRecord) (*mqlHcpTerraformPolicy, error) {
	var attrs tfePolicyAttrs
	if err := rec.DecodeAttributes(&attrs); err != nil {
		return nil, err
	}
	level := terraformEnforcementLevel(attrs.EnforcementLevel, attrs.Enforce)
	if orgName == "" {
		orgName = relOneID(rec, "organization")
	}

	res, err := CreateResource(runtime, "hcp.terraform.policy", map[string]*llx.RawData{
		"__id":             llx.StringData("hcp.terraform.policy/" + rec.ID),
		"id":               llx.StringData(rec.ID),
		"name":             llx.StringData(attrs.Name),
		"description":      llx.StringData(derefStr(attrs.Description)),
		"kind":             llx.StringData(attrs.Kind),
		"enforcementLevel": llx.StringData(level),
		"blocking":         llx.BoolData(terraformPolicyBlocking(level)),
		"policySetCount":   llx.IntData(attrs.PolicySetCount),
		"updatedAt":        llx.TimeDataPtr(attrs.UpdatedAt.Time),
	})
	if err != nil {
		return nil, err
	}
	policy := res.(*mqlHcpTerraformPolicy)
	policy.cacheOrgName = orgName
	return policy, nil
}

// organization resolves the organization the policy belongs to.
func (r *mqlHcpTerraformPolicy) organization() (*mqlHcpTerraformOrganization, error) {
	if r.cacheOrgName == "" {
		r.Organization.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "hcp.terraform.organization", map[string]*llx.RawData{
		"name": llx.StringData(r.cacheOrgName),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHcpTerraformOrganization), nil
}

// ---------------------------------------------------------------------------
// hcp.terraform.agentPool
// ---------------------------------------------------------------------------

// agentPools lists the agent pools registered in the organization.
func (r *mqlHcpTerraformOrganization) agentPools() ([]any, error) {
	client, err := terraformClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	records, err := client.List(terraformCtx(),
		"organizations/"+url.PathEscape(r.Name.Data)+"/agent-pools", nil)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, rec := range records {
		pool, err := newMqlHcpTerraformAgentPool(r.MqlRuntime, r.Name.Data, rec)
		if err != nil {
			return nil, err
		}
		out = append(out, pool)
	}
	return out, nil
}

func newMqlHcpTerraformAgentPool(runtime *plugin.Runtime, orgName string, rec connection.TfeRecord) (*mqlHcpTerraformAgentPool, error) {
	var attrs tfeAgentPoolAttrs
	if err := rec.DecodeAttributes(&attrs); err != nil {
		return nil, err
	}
	if orgName == "" {
		orgName = relOneID(rec, "organization")
	}

	res, err := CreateResource(runtime, "hcp.terraform.agentPool", map[string]*llx.RawData{
		"__id":               llx.StringData("hcp.terraform.agentPool/" + rec.ID),
		"id":                 llx.StringData(rec.ID),
		"name":               llx.StringData(attrs.Name),
		"agentCount":         llx.IntData(attrs.AgentCount),
		"organizationScoped": llx.BoolData(attrs.OrganizationScoped),
		"createdAt":          llx.TimeDataPtr(attrs.CreatedAt.Time),
	})
	if err != nil {
		return nil, err
	}
	pool := res.(*mqlHcpTerraformAgentPool)
	pool.cacheOrgName = orgName
	pool.cacheAllowedWorkspaceIDs = relManyIDs(rec, "allowed-workspaces")
	return pool, nil
}

// initHcpTerraformAgentPool hydrates a single agent pool by id.
func initHcpTerraformAgentPool(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	id := ""
	if raw, ok := args["id"]; ok {
		id, _ = raw.Value.(string)
	}
	if id == "" {
		return nil, nil, fmt.Errorf("hcp.terraform.agentPool requires an agent pool id")
	}
	client, err := terraformClient(runtime)
	if err != nil {
		return nil, nil, err
	}
	rec, err := client.GetOne(terraformCtx(), "agent-pools/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, nil, err
	}
	if rec == nil {
		return nil, nil, fmt.Errorf("hcp.terraform.agentPool %q not found", id)
	}
	pool, err := newMqlHcpTerraformAgentPool(runtime, "", *rec)
	if err != nil {
		return nil, nil, err
	}
	return nil, pool, nil
}

// organization resolves the organization the agent pool belongs to.
func (r *mqlHcpTerraformAgentPool) organization() (*mqlHcpTerraformOrganization, error) {
	if r.cacheOrgName == "" {
		r.Organization.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "hcp.terraform.organization", map[string]*llx.RawData{
		"name": llx.StringData(r.cacheOrgName),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHcpTerraformOrganization), nil
}

// allowedWorkspaces lists the workspaces allowed to run on a scoped pool.
func (r *mqlHcpTerraformAgentPool) allowedWorkspaces() ([]any, error) {
	return resolveWorkspaceRefs(r.MqlRuntime, r.cacheAllowedWorkspaceIDs)
}
