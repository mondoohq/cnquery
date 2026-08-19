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

	tfe "github.com/hashicorp/go-tfe"
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

// timePtr turns a go-tfe timestamp into an optional one. go-tfe types every
// timestamp as a bare time.Time, so an absent or null value arrives as the zero
// instant. Reporting that would put 1 January year 1 in a `time` field as
// though it were a real date, so it is reported as no time at all instead.
//
// A genuine year-1 timestamp is not a value any of these APIs produce, so
// nothing real is lost, and the sweep for `0001-01-01` in the verification
// checklist becomes impossible to fail rather than merely expected to pass.
func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// timestampAttrSuffix marks the attribute names carrying an ISO 8601 timestamp.
// Every one of them in this API ends in "-at": created-at, updated-at,
// last-used-at, expired-at, trial-expires-at.
const timestampAttrSuffix = "-at"

// sanitizeTimestamps replaces any timestamp attribute that is not a parseable
// RFC 3339 string with null, and returns the rewritten attributes.
//
// This exists because go-tfe's decoder rejects the whole record when a single
// timestamp is malformed, an empty string, or a number: the error is
// "Only strings can be parsed as dates, ISO8601 timestamps", and it takes every
// other field of that record down with it. One odd timestamp would blind an
// entire collection, and a shortened list satisfies every assertion made about
// it. Nulling the offending value keeps the record readable and reports the
// timestamp itself as null, which is what this provider did before go-tfe.
//
// tfeTime is the parser, so the tolerated shapes are exactly the ones its own
// tests pin.
func sanitizeTimestamps(attrs json.RawMessage) json.RawMessage {
	if len(attrs) == 0 {
		return attrs
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(attrs, &raw); err != nil {
		// Not an object; leave it for the typed decoder to report.
		return attrs
	}

	changed := false
	for key, val := range raw {
		if !strings.HasSuffix(key, timestampAttrSuffix) {
			continue
		}
		if string(val) == "null" {
			continue
		}
		var parsed tfeTime
		// tfeTime never errors; it yields a nil Time for anything it cannot
		// read, which is precisely the set we need to null out.
		_ = parsed.UnmarshalJSON(val)
		if parsed.Time == nil {
			raw[key] = json.RawMessage("null")
			changed = true
		}
	}
	if !changed {
		return attrs
	}

	rewritten, err := json.Marshal(raw)
	if err != nil {
		return attrs
	}
	return rewritten
}

// decodeTfeRecord decodes a record into a go-tfe struct, tolerating an
// unreadable timestamp, and clears the credential fields go-tfe carries but
// this provider deliberately does not model (see the scrub functions below).
//
// Every record in this file goes through here, so it is the single place where
// a secret could enter the process and the single place it is removed.
func decodeTfeRecord(rec connection.TfeRecord, out any) error {
	rec.Attributes = sanitizeTimestamps(rec.Attributes)
	if err := rec.DecodeTyped(out); err != nil {
		return err
	}
	scrubSecrets(out)
	return nil
}

// ---------------------------------------------------------------------------
// secret scrubbing
// ---------------------------------------------------------------------------

// scrubSecrets clears the credential-bearing fields go-tfe's types carry and
// this provider does not model: a workspace variable's value, a team token's
// secret, and the OAuth token id and webhook URL of a VCS connection.
//
// The API returns these whether or not anybody asked for them, so the exposure
// is not new. What is new is that adopting go-tfe's structs puts a populated
// Value one `llx.StringData(v.Value)` away from a future contributor, where
// before there was no such field to reach for. Clearing them at the one decode
// chokepoint restores that: a field added later reads "" and fails its own
// test, rather than silently shipping a workspace credential to a scan report.
func scrubSecrets(out any) {
	switch v := out.(type) {
	case *tfe.Variable:
		v.Value = ""
	case *tfe.TeamToken:
		v.Token = ""
	case *tfe.Workspace:
		scrubVCSRepo(v.VCSRepo)
	case *tfe.PolicySet:
		scrubVCSRepo(v.VCSRepo)
	}
}

func scrubVCSRepo(repo *tfe.VCSRepo) {
	if repo == nil {
		return
	}
	repo.OAuthTokenID = ""
	repo.WebhookURL = ""
}

// ---------------------------------------------------------------------------
// API record shapes
//
// The record types come from github.com/hashicorp/go-tfe: tfe.Organization,
// tfe.Workspace, tfe.VCSRepo, tfe.TeamAccess, tfe.Variable, tfe.Team,
// tfe.OrganizationAccess, tfe.TeamToken, tfe.PolicySet, tfe.Policy,
// tfe.Enforcement and tfe.AgentPool. Their `jsonapi:"attr,…"` tags are
// maintained by the vendor and exercised continuously by
// terraform-provider-tfe, which makes them a far better source of truth than
// tags written here from API documentation.
//
// The structs below cover only what go-tfe cannot express. There are two
// distinct reasons a field lands here, and they are not interchangeable:
//
//  1. go-tfe does not model the attribute at all. `plan-expired` and
//     `versioned` appear in neither its Organization nor its PolicySet, though
//     both are real attributes: the vendor's own OpenAPI specification (the one
//     go-tfe/v2 is generated from) carries `planExpired` and `versioned`.
//
//  2. go-tfe models the attribute with a non-pointer type, so a null from the
//     API is indistinguishable from a zero value. Reporting 0 for a session
//     lifetime the organization never set, or "" for a SAML role that does not
//     exist, would be inventing a value the API did not give.
//
// Everything else is read from the vendor types.
// ---------------------------------------------------------------------------

// tfeOrganizationGaps carries the organization attributes go-tfe cannot supply.
type tfeOrganizationGaps struct {
	// go-tfe has no field for this attribute at all.
	PlanExpired bool `json:"plan-expired"`
	// go-tfe types these as int/int/string, collapsing null onto 0 and "".
	SessionTimeout       *int64  `json:"session-timeout"`
	SessionRemember      *int64  `json:"session-remember"`
	OwnersTeamSamlRoleID *string `json:"owners-team-saml-role-id"`
}

// tfeTeamGaps carries the team attributes go-tfe cannot supply.
type tfeTeamGaps struct {
	// go-tfe types this as string, so an unmapped team reports "" rather than
	// null and cannot be told apart from one mapped to an empty group.
	SSOTeamID *string `json:"sso-team-id"`
}

// tfePolicySetGaps carries the policy set attributes go-tfe cannot supply.
type tfePolicySetGaps struct {
	// go-tfe has no field for this attribute at all.
	Versioned bool `json:"versioned"`
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
func terraformVCSDriven(repo *tfe.VCSRepo) bool {
	if repo == nil {
		return false
	}
	return repo.Identifier != "" || repo.DisplayIdentifier != ""
}

// terraformVCSIdentifier returns the repository backing a workspace, preferring
// the canonical identifier and falling back to the display identifier some
// VCS providers report instead.
func terraformVCSIdentifier(repo *tfe.VCSRepo) string {
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
//
// go-tfe marks Policy.Enforce "Deprecated: Use EnforcementLevel instead", which
// is exactly the preference order below. It is still read, deliberately, so a
// policy on an older Terraform Enterprise that reports only the legacy list
// does not read as unenforced.
func terraformEnforcementLevel(level string, enforce []*tfe.Enforcement) string {
	if level != "" {
		return level
	}
	for _, e := range enforce {
		if e == nil {
			continue
		}
		if e.Mode != "" {
			return string(e.Mode)
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

// The organization has no internal cache struct: the owners team is resolved by
// name rather than from a relationship (see ownersTeam), so there is nothing
// left to carry from the creation context.

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
	var org tfe.Organization
	if err := decodeTfeRecord(rec, &org); err != nil {
		return nil, err
	}
	var gaps tfeOrganizationGaps
	if err := rec.DecodeAttributes(&gaps); err != nil {
		return nil, err
	}
	// The record id is the organization name; go-tfe maps it onto Name.
	name := org.Name
	if name == "" {
		name = rec.ID
	}
	authPolicy := string(org.CollaboratorAuthPolicy)

	res, err := CreateResource(runtime, "hcp.terraform.organization", map[string]*llx.RawData{
		"__id":                       llx.StringData("hcp.terraform.organization/" + name),
		"name":                       llx.StringData(name),
		"externalId":                 llx.StringData(org.ExternalID),
		"email":                      llx.StringData(org.Email),
		"createdAt":                  llx.TimeDataPtr(timePtr(org.CreatedAt)),
		"collaboratorAuthPolicy":     llx.StringData(authPolicy),
		"twoFactorRequired":          llx.BoolData(terraformTwoFactorRequired(authPolicy)),
		"twoFactorConformant":        llx.BoolData(org.TwoFactorConformant),
		"samlEnabled":                llx.BoolData(org.SAMLEnabled),
		"ownersTeamSamlRoleId":       llx.StringDataPtr(gaps.OwnersTeamSamlRoleID),
		"sessionTimeoutMinutes":      llx.IntDataPtr(gaps.SessionTimeout),
		"sessionRememberMinutes":     llx.IntDataPtr(gaps.SessionRemember),
		"costEstimationEnabled":      llx.BoolData(org.CostEstimationEnabled),
		"assessmentsEnforced":        llx.BoolData(org.AssessmentsEnforced),
		"allowForceDeleteWorkspaces": llx.BoolData(org.AllowForceDeleteWorkspaces),
		"defaultExecutionMode":       llx.StringData(org.DefaultExecutionMode),
		"planExpired":                llx.BoolData(gaps.PlanExpired),
	})
	if err != nil {
		return nil, err
	}
	mqlOrg := res.(*mqlHcpTerraformOrganization)
	return mqlOrg, nil
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
//
// This reads the team named "owners", which HCP Terraform creates with every
// organization and does not allow renaming. An earlier version preferred an
// "owners-team" relationship on the organization record and used the name only
// as a fallback, but that relationship does not exist: it appears neither in
// go-tfe's Organization type nor in the vendor's OpenAPI specification, whose
// organization relationships are default-project, default-agent-pool,
// entitlement-set, subscription, data-retention-policy, the two token links,
// the two producer links and primary-hyok-configuration. The relationship read
// was therefore dead code and the name lookup ran on every call regardless, so
// only the name lookup remains.
//
// It costs one team listing per read. That is worth recording rather than
// hiding: if an installation ever permits renaming the owners team, this
// returns null and an audit asserting "the owners team enforces 2FA" finds
// nothing to assert against.
func (r *mqlHcpTerraformOrganization) ownersTeam() (*mqlHcpTerraformTeam, error) {
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
	var ws tfe.Workspace
	if err := decodeTfeRecord(rec, &ws); err != nil {
		return nil, err
	}

	vcsBranch, vcsProvider, vcsSubmodules := "", "", false
	if ws.VCSRepo != nil {
		vcsBranch = ws.VCSRepo.Branch
		vcsProvider = ws.VCSRepo.ServiceProvider
		vcsSubmodules = ws.VCSRepo.IngressSubmodules
	}

	if orgName == "" {
		orgName = relOneID(rec, "organization")
	}

	res, err := CreateResource(runtime, "hcp.terraform.workspace", map[string]*llx.RawData{
		"__id":                       llx.StringData("hcp.terraform.workspace/" + rec.ID),
		"id":                         llx.StringData(rec.ID),
		"name":                       llx.StringData(ws.Name),
		"description":                llx.StringData(ws.Description),
		"executionMode":              llx.StringData(ws.ExecutionMode),
		"autoApply":                  llx.BoolData(ws.AutoApply),
		"autoApplyRunTrigger":        llx.BoolData(ws.AutoApplyRunTrigger),
		"terraformVersion":           llx.StringData(ws.TerraformVersion),
		"workingDirectory":           llx.StringData(ws.WorkingDirectory),
		"locked":                     llx.BoolData(ws.Locked),
		"vcsDriven":                  llx.BoolData(terraformVCSDriven(ws.VCSRepo)),
		"vcsRepoIdentifier":          llx.StringData(terraformVCSIdentifier(ws.VCSRepo)),
		"vcsRepoBranch":              llx.StringData(vcsBranch),
		"vcsRepoServiceProvider":     llx.StringData(vcsProvider),
		"vcsRepoIngressSubmodules":   llx.BoolData(vcsSubmodules),
		"speculativeEnabled":         llx.BoolData(ws.SpeculativeEnabled),
		"globalRemoteState":          llx.BoolData(ws.GlobalRemoteState),
		"allowDestroyPlan":           llx.BoolData(ws.AllowDestroyPlan),
		"fileTriggersEnabled":        llx.BoolData(ws.FileTriggersEnabled),
		"queueAllRuns":               llx.BoolData(ws.QueueAllRuns),
		"structuredRunOutputEnabled": llx.BoolData(ws.StructuredRunOutputEnabled),
		"assessmentsEnabled":         llx.BoolData(ws.AssessmentsEnabled),
		"resourceCount":              llx.IntData(int64(ws.ResourceCount)),
		"tagNames":                   llx.ArrayData(strSlice(ws.TagNames), types.String),
		"createdAt":                  llx.TimeDataPtr(timePtr(ws.CreatedAt)),
		"updatedAt":                  llx.TimeDataPtr(timePtr(ws.UpdatedAt)),
	})
	if err != nil {
		return nil, err
	}
	mqlWs := res.(*mqlHcpTerraformWorkspace)
	mqlWs.cacheOrgName = orgName
	mqlWs.cacheAgentPoolID = relOneID(rec, "agent-pool")
	return mqlWs, nil
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

// remoteStateConsumers lists the workspaces explicitly allowed to read this
// workspace's state outputs.
//
// show_only_configured is what makes that true. Without it the endpoint returns
// every workspace in the organization once global-remote-state is enabled,
// which is not a list of grants anybody made: it would report an organization's
// entire workspace inventory as deliberate consumers of one workspace's state,
// and grow as the square of the estate. The parameter is documented in the
// vendor's OpenAPI specification as "return only explicitly configured remote
// state consumers even if global-remote-state is enabled", which is exactly the
// field's stated meaning.
func (r *mqlHcpTerraformWorkspace) remoteStateConsumers() ([]any, error) {
	client, err := terraformClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("show_only_configured", "true")

	records, err := client.List(terraformCtx(),
		"workspaces/"+url.PathEscape(r.Id.Data)+"/relationships/remote-state-consumers", query)
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
	// go-tfe issues the same filter for this endpoint: its
	// TeamAccessListOptions.WorkspaceID carries `url:"filter[workspace][id]"`,
	// and the vendor's OpenAPI specification documents the same parameter as
	// "the workspace ID to list team access for". A wrong name here would not
	// error, it would return every grant in the organization and report other
	// workspaces' permissions against this one, so it is pinned by a test.
	query := url.Values{}
	query.Set("filter[workspace][id]", r.Id.Data)

	records, err := client.List(terraformCtx(), "team-workspaces", query)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, rec := range records {
		var grant tfe.TeamAccess
		if err := decodeTfeRecord(rec, &grant); err != nil {
			return nil, err
		}
		access := string(grant.Access)
		runs := string(grant.Runs)
		teamID := relOneID(rec, "team")
		workspaceID := relOneID(rec, "workspace")
		if workspaceID == "" {
			workspaceID = r.Id.Data
		}

		res, err := CreateResource(r.MqlRuntime, "hcp.terraform.teamAccess", map[string]*llx.RawData{
			"__id":             llx.StringData("hcp.terraform.teamAccess/" + rec.ID),
			"id":               llx.StringData(rec.ID),
			"access":           llx.StringData(access),
			"canApply":         llx.BoolData(terraformCanApply(access, runs)),
			"runs":             llx.StringData(runs),
			"variables":        llx.StringData(string(grant.Variables)),
			"stateVersions":    llx.StringData(string(grant.StateVersions)),
			"sentinelMocks":    llx.StringData(string(grant.SentinelMocks)),
			"workspaceLocking": llx.BoolData(grant.WorkspaceLocking),
			"runTasks":         llx.BoolData(grant.RunTasks),
		})
		if err != nil {
			return nil, err
		}
		mqlAccess := res.(*mqlHcpTerraformTeamAccess)
		mqlAccess.cacheTeamID = teamID
		mqlAccess.cacheWorkspaceID = workspaceID
		out = append(out, mqlAccess)
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

// variables lists the variables defined on the workspace.
//
// Variable values are deliberately not modelled, for sensitive and
// non-sensitive variables alike, so a scan never carries a workspace credential
// out of HCP Terraform. go-tfe's Variable type does carry a Value; it is
// cleared by scrubSecrets at the decode chokepoint before this function sees
// it, so there is no populated value here to expose by accident.
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
		var variable tfe.Variable
		if err := decodeTfeRecord(rec, &variable); err != nil {
			return nil, err
		}
		res, err := CreateResource(r.MqlRuntime, "hcp.terraform.variable", map[string]*llx.RawData{
			"__id":        llx.StringData("hcp.terraform.variable/" + r.Id.Data + "/" + rec.ID),
			"id":          llx.StringData(rec.ID),
			"key":         llx.StringData(variable.Key),
			"category":    llx.StringData(string(variable.Category)),
			"sensitive":   llx.BoolData(variable.Sensitive),
			"hcl":         llx.BoolData(variable.HCL),
			"description": llx.StringData(variable.Description),
		})
		if err != nil {
			return nil, err
		}
		mqlVariable := res.(*mqlHcpTerraformVariable)
		mqlVariable.cacheWorkspaceID = r.Id.Data
		out = append(out, mqlVariable)
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
	var team tfe.Team
	if err := decodeTfeRecord(rec, &team); err != nil {
		return nil, err
	}
	var gaps tfeTeamGaps
	if err := rec.DecodeAttributes(&gaps); err != nil {
		return nil, err
	}
	// A team with no organization-access object holds none of the
	// organization-level permissions; the zero value reports exactly that.
	access := team.OrganizationAccess
	if access == nil {
		access = &tfe.OrganizationAccess{}
	}
	if orgName == "" {
		orgName = relOneID(rec, "organization")
	}

	res, err := CreateResource(runtime, "hcp.terraform.team", map[string]*llx.RawData{
		"__id":                        llx.StringData("hcp.terraform.team/" + rec.ID),
		"id":                          llx.StringData(rec.ID),
		"name":                        llx.StringData(team.Name),
		"usersCount":                  llx.IntData(int64(team.UserCount)),
		"visibility":                  llx.StringData(team.Visibility),
		"ssoTeamId":                   llx.StringDataPtr(gaps.SSOTeamID),
		"allowMemberTokenManagement":  llx.BoolData(team.AllowMemberTokenManagement),
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
	mqlTeam := res.(*mqlHcpTerraformTeam)
	mqlTeam.cacheOrgName = orgName
	return mqlTeam, nil
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

// tokens lists the API tokens issued to the team.
//
// The listing comes from the organization: `GET /organizations/:org/team-tokens`
// returns every team token in the organization, each carrying a `team`
// relationship, and this filters that to the team in hand.
//
// An earlier version listed `GET /teams/:id/authentication-tokens`, which does
// not exist. That path accepts only POST, to create a token with a description;
// go-tfe builds it for exactly that and no more, and the vendor's OpenAPI
// specification gives it no GET at all. So the list always 404ed and the code
// always fell through to the singular `GET /teams/:id/authentication-token` -
// which returns at most one token. A team holding three descriptive tokens
// reported one, and an audit asserting "this organization issues no long-lived
// team tokens" could pass on an organization that had several. The singular
// endpoint is kept as a fallback for installations that predate the
// organization-wide listing, where it is the only source there is.
func (r *mqlHcpTerraformTeam) tokens() ([]any, error) {
	client, err := terraformClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	records, err := r.listTeamTokenRecords(client)
	if err != nil {
		return nil, err
	}

	out := []any{}
	for _, rec := range records {
		var token tfe.TeamToken
		if err := decodeTfeRecord(rec, &token); err != nil {
			return nil, err
		}
		res, err := CreateResource(r.MqlRuntime, "hcp.terraform.teamToken", map[string]*llx.RawData{
			"__id":        llx.StringData("hcp.terraform.teamToken/" + r.Id.Data + "/" + rec.ID),
			"id":          llx.StringData(rec.ID),
			"description": llx.StringData(derefStr(token.Description)),
			"createdAt":   llx.TimeDataPtr(timePtr(token.CreatedAt)),
			"lastUsedAt":  llx.TimeDataPtr(timePtr(token.LastUsedAt)),
			"expiredAt":   llx.TimeDataPtr(timePtr(token.ExpiredAt)),
		})
		if err != nil {
			return nil, err
		}
		mqlToken := res.(*mqlHcpTerraformTeamToken)
		mqlToken.cacheTeamID = r.Id.Data
		out = append(out, mqlToken)
	}
	return out, nil
}

// listTeamTokenRecords fetches the team's tokens, preferring the
// organization-wide listing and falling back to the single-token endpoint on an
// installation that does not serve it.
func (r *mqlHcpTerraformTeam) listTeamTokenRecords(client *connection.TfeClient) ([]connection.TfeRecord, error) {
	orgName := r.cacheOrgName
	if orgName == "" {
		// Without the organization there is no listing endpoint to call; the
		// single-token endpoint is all that is reachable.
		return r.legacyTeamTokenRecords(client)
	}

	records, err := client.List(terraformCtx(),
		"organizations/"+url.PathEscape(orgName)+"/team-tokens", nil)
	if err != nil {
		if connection.IsTfeNotFound(err) {
			return r.legacyTeamTokenRecords(client)
		}
		return nil, err
	}

	// The listing spans the organization, so it has to be narrowed to this
	// team. A record whose team relationship is missing is skipped rather than
	// attributed here: guessing would report another team's token as this
	// team's, and over-reporting a credential is worse than under-reporting it.
	mine := []connection.TfeRecord{}
	for _, rec := range records {
		if relOneID(rec, "team") == r.Id.Data {
			mine = append(mine, rec)
		}
	}
	return mine, nil
}

// legacyTeamTokenRecords reads the single team token endpoint, which is the
// only one older Terraform Enterprise installations serve.
func (r *mqlHcpTerraformTeam) legacyTeamTokenRecords(client *connection.TfeClient) ([]connection.TfeRecord, error) {
	rec, err := client.GetOne(terraformCtx(),
		"teams/"+url.PathEscape(r.Id.Data)+"/authentication-token", nil)
	if err != nil {
		if connection.IsTfeUnavailable(err) {
			// No token has been issued to this team.
			return []connection.TfeRecord{}, nil
		}
		return nil, err
	}
	if rec == nil {
		return []connection.TfeRecord{}, nil
	}
	return []connection.TfeRecord{*rec}, nil
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
		var set tfe.PolicySet
		if err := decodeTfeRecord(rec, &set); err != nil {
			return nil, err
		}
		var gaps tfePolicySetGaps
		if err := rec.DecodeAttributes(&gaps); err != nil {
			return nil, err
		}
		overridable := false
		if set.Overridable != nil {
			overridable = *set.Overridable
		}
		res, err := CreateResource(r.MqlRuntime, "hcp.terraform.policySet", map[string]*llx.RawData{
			"__id":           llx.StringData("hcp.terraform.policySet/" + rec.ID),
			"id":             llx.StringData(rec.ID),
			"name":           llx.StringData(set.Name),
			"description":    llx.StringData(set.Description),
			"kind":           llx.StringData(string(set.Kind)),
			"global":         llx.BoolData(set.Global),
			"policyCount":    llx.IntData(int64(set.PolicyCount)),
			"workspaceCount": llx.IntData(int64(set.WorkspaceCount)),
			"versioned":      llx.BoolData(gaps.Versioned),
			"policiesPath":   llx.StringData(set.PoliciesPath),
			"agentEnabled":   llx.BoolData(set.AgentEnabled),
			"overridable":    llx.BoolData(overridable),
			"createdAt":      llx.TimeDataPtr(timePtr(set.CreatedAt)),
			"updatedAt":      llx.TimeDataPtr(timePtr(set.UpdatedAt)),
		})
		if err != nil {
			return nil, err
		}
		mqlSet := res.(*mqlHcpTerraformPolicySet)
		mqlSet.cacheOrgName = r.Name.Data
		mqlSet.cachePolicyIDs = relManyIDs(rec, "policies")
		mqlSet.cacheWorkspaceIDs = relManyIDs(rec, "workspaces")
		out = append(out, mqlSet)
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
	var policy tfe.Policy
	if err := decodeTfeRecord(rec, &policy); err != nil {
		return nil, err
	}
	level := terraformEnforcementLevel(string(policy.EnforcementLevel), policy.Enforce)
	if orgName == "" {
		orgName = relOneID(rec, "organization")
	}

	res, err := CreateResource(runtime, "hcp.terraform.policy", map[string]*llx.RawData{
		"__id":             llx.StringData("hcp.terraform.policy/" + rec.ID),
		"id":               llx.StringData(rec.ID),
		"name":             llx.StringData(policy.Name),
		"description":      llx.StringData(policy.Description),
		"kind":             llx.StringData(string(policy.Kind)),
		"enforcementLevel": llx.StringData(level),
		"blocking":         llx.BoolData(terraformPolicyBlocking(level)),
		"policySetCount":   llx.IntData(int64(policy.PolicySetCount)),
		"updatedAt":        llx.TimeDataPtr(timePtr(policy.UpdatedAt)),
	})
	if err != nil {
		return nil, err
	}
	mqlPolicy := res.(*mqlHcpTerraformPolicy)
	mqlPolicy.cacheOrgName = orgName
	return mqlPolicy, nil
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
	var pool tfe.AgentPool
	if err := decodeTfeRecord(rec, &pool); err != nil {
		return nil, err
	}
	if orgName == "" {
		orgName = relOneID(rec, "organization")
	}

	res, err := CreateResource(runtime, "hcp.terraform.agentPool", map[string]*llx.RawData{
		"__id":               llx.StringData("hcp.terraform.agentPool/" + rec.ID),
		"id":                 llx.StringData(rec.ID),
		"name":               llx.StringData(pool.Name),
		"agentCount":         llx.IntData(int64(pool.AgentCount)),
		"organizationScoped": llx.BoolData(pool.OrganizationScoped),
		"createdAt":          llx.TimeDataPtr(timePtr(pool.CreatedAt)),
	})
	if err != nil {
		return nil, err
	}
	mqlPool := res.(*mqlHcpTerraformAgentPool)
	mqlPool.cacheOrgName = orgName
	mqlPool.cacheAllowedWorkspaceIDs = relManyIDs(rec, "allowed-workspaces")
	return mqlPool, nil
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
