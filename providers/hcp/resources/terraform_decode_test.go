// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers/hcp/connection"
)

// decodeRecord unmarshals a JSON:API resource object and decodes it into a
// go-tfe record type, exercising exactly the path the resource code takes.
//
// The payloads below are unchanged from when this provider decoded them with
// hand-written struct tags. They now pin go-tfe's tags instead, which is the
// point: the same fixtures that caught a mistyped tag here would catch a
// wire-format contradiction in the vendor's types.
func decodeRecord(t *testing.T, raw string, out any) connection.TfeRecord {
	t.Helper()
	var rec connection.TfeRecord
	require.NoError(t, json.Unmarshal([]byte(raw), &rec))
	require.NoError(t, decodeTfeRecord(rec, out))
	return rec
}

// decodeGaps decodes the attributes go-tfe cannot express (see the tfe*Gaps
// types) straight off the record.
func decodeGaps(t *testing.T, raw string, out any) connection.TfeRecord {
	t.Helper()
	var rec connection.TfeRecord
	require.NoError(t, json.Unmarshal([]byte(raw), &rec))
	require.NoError(t, rec.DecodeAttributes(out))
	return rec
}

// ---------------------------------------------------------------------------
// timestamps
// ---------------------------------------------------------------------------

func TestTfeTimeDecoding(t *testing.T) {
	want := time.Date(2017, 9, 7, 14, 34, 40, 492000000, time.UTC)

	tests := []struct {
		name string
		raw  string
		want *time.Time
	}{
		{"rfc3339 with millis", `{"t":"2017-09-07T14:34:40.492Z"}`, &want},
		{"null", `{"t":null}`, nil},
		{"empty string", `{"t":""}`, nil},
		{"absent", `{}`, nil},
		{"malformed", `{"t":"not-a-time"}`, nil},
		{"number", `{"t":1234}`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got struct {
				T tfeTime `json:"t"`
			}
			// A bad timestamp must not fail the whole record: one odd value
			// cannot be allowed to blind an entire collection.
			require.NoError(t, json.Unmarshal([]byte(tt.raw), &got))
			if tt.want == nil {
				assert.Nil(t, got.T.Time, "absent or unparseable timestamps must stay null, "+
					"never become the zero instant (1 January year 1)")
				return
			}
			require.NotNil(t, got.T.Time)
			assert.True(t, got.T.Time.Equal(*tt.want), "got %v want %v", got.T.Time, tt.want)
		})
	}
}

func TestTimePtrNullsTheZeroInstant(t *testing.T) {
	// go-tfe types every timestamp as a bare time.Time, so an absent or null
	// value arrives as the zero instant. Reporting that would put
	// 1 January year 1 in a `time` field as though it were a real date.
	assert.Nil(t, timePtr(time.Time{}))

	real := time.Date(2024, 2, 1, 9, 0, 0, 0, time.UTC)
	got := timePtr(real)
	require.NotNil(t, got)
	assert.True(t, got.Equal(real))
}

func TestSanitizeTimestampsKeepsOneBadValueFromBlindingTheRecord(t *testing.T) {
	// go-tfe's decoder rejects the entire record when a single timestamp is
	// malformed, empty, or a number, taking every other field down with it.
	// The offending value is nulled instead, so the record stays readable and
	// the timestamp alone reports null.
	for _, tc := range []struct{ name, raw string }{
		{"malformed", `{"id":"ws-1","type":"workspaces","attributes":{"name":"prod","updated-at":"not-a-date"}}`},
		{"empty string", `{"id":"ws-1","type":"workspaces","attributes":{"name":"prod","updated-at":""}}`},
		{"number", `{"id":"ws-1","type":"workspaces","attributes":{"name":"prod","updated-at":1767322845}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var ws tfe.Workspace
			decodeRecord(t, tc.raw, &ws)
			assert.Equal(t, "prod", ws.Name, "the rest of the record must survive one bad timestamp")
			assert.Nil(t, timePtr(ws.UpdatedAt), "an unreadable timestamp must report null")
		})
	}

	// A good timestamp is left alone.
	var ws tfe.Workspace
	decodeRecord(t, `{"id":"ws-1","type":"workspaces","attributes":{"name":"prod","updated-at":"2024-02-01T09:00:00Z"}}`, &ws)
	require.NotNil(t, timePtr(ws.UpdatedAt))
	assert.Equal(t, 2024, ws.UpdatedAt.Year())
}

func TestDerefStr(t *testing.T) {
	assert.Equal(t, "", derefStr(nil))
	s := "value"
	assert.Equal(t, "value", derefStr(&s))
	empty := ""
	assert.Equal(t, "", derefStr(&empty))
}

// ---------------------------------------------------------------------------
// organization
// ---------------------------------------------------------------------------

const orgPayload = `{
  "id": "acme",
  "type": "organizations",
  "attributes": {
    "external-id": "org-Hu9WT3XkVQpMWkBb",
    "created-at": "2017-09-07T14:34:40.492Z",
    "email": "ops@acme.example",
    "session-timeout": 20160,
    "session-remember": 20161,
    "collaborator-auth-policy": "two_factor_mandatory",
    "plan-expired": true,
    "cost-estimation-enabled": true,
    "name": "acme",
    "saml-enabled": true,
    "owners-team-saml-role-id": "owners-role",
    "two-factor-conformant": true,
    "assessments-enforced": true,
    "allow-force-delete-workspaces": true,
    "default-execution-mode": "agent"
  },
  "relationships": {
    "owners-team": {"data": {"id": "team-owners", "type": "teams"}}
  }
}`

func TestOrganizationAttributeTags(t *testing.T) {
	var org tfe.Organization
	decodeRecord(t, orgPayload, &org)

	var gaps tfeOrganizationGaps
	decodeGaps(t, orgPayload, &gaps)

	assert.Equal(t, "acme", org.Name)
	assert.Equal(t, "org-Hu9WT3XkVQpMWkBb", org.ExternalID)
	assert.Equal(t, "ops@acme.example", org.Email)
	assert.Equal(t, "two_factor_mandatory", string(org.CollaboratorAuthPolicy))
	// Every security-relevant tag is pinned: a mistyped tag would decode to
	// the zero value and report a hardened organization as unhardened.
	assert.True(t, org.TwoFactorConformant)
	assert.True(t, org.SAMLEnabled)
	assert.True(t, org.CostEstimationEnabled)
	assert.True(t, org.AssessmentsEnforced)
	assert.True(t, org.AllowForceDeleteWorkspaces)
	assert.True(t, gaps.PlanExpired)
	assert.Equal(t, "agent", org.DefaultExecutionMode)
	require.NotNil(t, gaps.OwnersTeamSamlRoleID)
	assert.Equal(t, "owners-role", *gaps.OwnersTeamSamlRoleID)
	require.NotNil(t, gaps.SessionTimeout)
	assert.Equal(t, int64(20160), *gaps.SessionTimeout)
	require.NotNil(t, gaps.SessionRemember)
	assert.Equal(t, int64(20161), *gaps.SessionRemember)
	require.NotNil(t, timePtr(org.CreatedAt))
}

func TestOrganizationHasNoOwnersTeamRelationship(t *testing.T) {
	// The organization record carries no owners-team relationship: it appears
	// neither in go-tfe's Organization type nor in the vendor's OpenAPI
	// specification. ownersTeam therefore resolves the team by name, and this
	// records why the relationship read was removed rather than kept as a
	// preferred path that never ran.
	var org tfe.Organization
	decodeRecord(t, orgPayload, &org)

	assert.Nil(t, org.DefaultAgentPool)
	assert.Nil(t, org.DefaultProject)
}

func TestOrganizationOptionalsStayNull(t *testing.T) {
	// The API reports "use the installation default" as null. Reporting 0
	// instead would claim a zero-minute session lifetime, which is a real
	// setting and a wrong one.
	//
	// go-tfe types these as plain int and string, so null and zero are
	// indistinguishable in its struct. That is exactly why they are read from
	// tfeOrganizationGaps rather than from the vendor type.
	const raw = `{
      "id":"acme","type":"organizations",
      "attributes":{
        "name":"acme",
        "session-timeout":null,
        "session-remember":null,
        "owners-team-saml-role-id":null,
        "created-at":null,
        "collaborator-auth-policy":"password"
      }
    }`

	var org tfe.Organization
	decodeRecord(t, raw, &org)

	var gaps tfeOrganizationGaps
	decodeGaps(t, raw, &gaps)

	assert.Nil(t, gaps.SessionTimeout)
	assert.Nil(t, gaps.SessionRemember)
	assert.Nil(t, gaps.OwnersTeamSamlRoleID)
	assert.Nil(t, timePtr(org.CreatedAt))
	assert.False(t, org.TwoFactorConformant)
	assert.False(t, gaps.PlanExpired)
}

func TestTerraformTwoFactorRequired(t *testing.T) {
	tests := []struct {
		policy string
		want   bool
	}{
		{"two_factor_mandatory", true},
		{"password", false},
		{"", false},
		{"TWO_FACTOR_MANDATORY", false},
		{"two-factor-mandatory", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, terraformTwoFactorRequired(tt.policy), "policy %q", tt.policy)
	}
}

// ---------------------------------------------------------------------------
// workspace
// ---------------------------------------------------------------------------

const workspacePayload = `{
  "id": "ws-Hu9WT3XkVQpMWkBb",
  "type": "workspaces",
  "attributes": {
    "name": "prod-network",
    "description": "production network",
    "execution-mode": "agent",
    "auto-apply": true,
    "auto-apply-run-trigger": true,
    "terraform-version": "1.9.5",
    "working-directory": "envs/prod",
    "locked": true,
    "speculative-enabled": true,
    "global-remote-state": true,
    "allow-destroy-plan": true,
    "file-triggers-enabled": true,
    "queue-all-runs": true,
    "structured-run-output-enabled": true,
    "assessments-enabled": true,
    "resource-count": 42,
    "tag-names": ["prod", "network"],
    "created-at": "2021-06-03T14:34:40.492Z",
    "updated-at": "2024-02-01T09:00:00Z",
    "vcs-repo": {
      "branch": "main",
      "ingress-submodules": true,
      "identifier": "acme/infrastructure",
      "display-identifier": "acme/infrastructure",
      "service-provider": "github"
    }
  },
  "relationships": {
    "organization": {"data": {"id": "acme", "type": "organizations"}},
    "agent-pool": {"data": {"id": "apool-1", "type": "agent-pools"}}
  }
}`

func TestWorkspaceAttributeTags(t *testing.T) {
	var ws tfe.Workspace
	rec := decodeRecord(t, workspacePayload, &ws)

	assert.Equal(t, "prod-network", ws.Name)
	assert.Equal(t, "agent", ws.ExecutionMode)
	assert.Equal(t, "1.9.5", ws.TerraformVersion)
	assert.True(t, ws.AutoApply)
	assert.True(t, ws.AutoApplyRunTrigger)
	assert.True(t, ws.Locked)
	assert.True(t, ws.SpeculativeEnabled)
	assert.True(t, ws.GlobalRemoteState)
	assert.True(t, ws.AllowDestroyPlan)
	assert.True(t, ws.FileTriggersEnabled)
	assert.True(t, ws.QueueAllRuns)
	assert.True(t, ws.StructuredRunOutputEnabled)
	assert.True(t, ws.AssessmentsEnabled)
	assert.Equal(t, int64(42), int64(ws.ResourceCount))
	assert.Equal(t, []string{"prod", "network"}, ws.TagNames)
	assert.Equal(t, "envs/prod", ws.WorkingDirectory)
	assert.Equal(t, "production network", ws.Description)
	require.NotNil(t, timePtr(ws.CreatedAt))
	require.NotNil(t, timePtr(ws.UpdatedAt))

	require.NotNil(t, ws.VCSRepo)
	assert.Equal(t, "acme/infrastructure", ws.VCSRepo.Identifier)
	assert.Equal(t, "main", ws.VCSRepo.Branch)
	assert.Equal(t, "github", ws.VCSRepo.ServiceProvider)
	assert.True(t, ws.VCSRepo.IngressSubmodules)

	assert.Equal(t, "acme", relOneID(rec, "organization"))
	assert.Equal(t, "apool-1", relOneID(rec, "agent-pool"))
}

func TestWorkspaceCLIDriven(t *testing.T) {
	// A CLI-driven workspace carries no VCS repo and null strings where the
	// VCS-driven one carries values.
	var ws tfe.Workspace
	rec := decodeRecord(t, `{
      "id":"ws-cli","type":"workspaces",
      "attributes":{
        "name":"cli-driven",
        "description":null,
        "working-directory":null,
        "execution-mode":"remote",
        "auto-apply":false,
        "locked":false,
        "global-remote-state":false,
        "vcs-repo":null,
        "tag-names":[],
        "created-at":"2021-06-03T14:34:40.492Z"
      },
      "relationships":{"organization":{"data":{"id":"acme","type":"organizations"}}}
    }`, &ws)

	assert.Nil(t, ws.VCSRepo)
	// go-tfe types these as plain strings, so a null arrives as "". That is
	// the value the schema reported before this change too, because the
	// resource passed derefStr over an optional string.
	assert.Equal(t, "", ws.Description)
	assert.Equal(t, "", ws.WorkingDirectory)
	assert.False(t, ws.AutoApply)
	assert.False(t, ws.Locked)
	assert.False(t, ws.GlobalRemoteState)
	assert.Nil(t, timePtr(ws.UpdatedAt))
	assert.Equal(t, "", relOneID(rec, "agent-pool"))
	assert.Empty(t, relManyIDs(rec, "workspaces"))
}

func TestTerraformVCSDriven(t *testing.T) {
	tests := []struct {
		name string
		repo *tfe.VCSRepo
		want bool
	}{
		{"no repo", nil, false},
		{"empty repo object", &tfe.VCSRepo{}, false},
		{"identifier", &tfe.VCSRepo{Identifier: "acme/infra"}, true},
		{"display identifier only", &tfe.VCSRepo{DisplayIdentifier: "acme/infra"}, true},
		{"branch without identifier", &tfe.VCSRepo{Branch: "main"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, terraformVCSDriven(tt.repo))
		})
	}
}

func TestTerraformVCSIdentifier(t *testing.T) {
	assert.Equal(t, "", terraformVCSIdentifier(nil))
	assert.Equal(t, "", terraformVCSIdentifier(&tfe.VCSRepo{}))
	assert.Equal(t, "acme/infra", terraformVCSIdentifier(&tfe.VCSRepo{Identifier: "acme/infra"}))
	assert.Equal(t, "acme/display", terraformVCSIdentifier(&tfe.VCSRepo{DisplayIdentifier: "acme/display"}))
	// The canonical identifier wins when both are present.
	assert.Equal(t, "acme/infra", terraformVCSIdentifier(&tfe.VCSRepo{
		Identifier: "acme/infra", DisplayIdentifier: "acme/display",
	}))
}

// ---------------------------------------------------------------------------
// team access
// ---------------------------------------------------------------------------

func TestTeamAccessAttributeTags(t *testing.T) {
	var grant tfe.TeamAccess
	rec := decodeRecord(t, `{
      "id":"tws-Hu9WT3XkVQpMWkBb","type":"team-workspaces",
      "attributes":{
        "access":"custom",
        "runs":"apply",
        "variables":"write",
        "state-versions":"read-outputs",
        "sentinel-mocks":"read",
        "workspace-locking":true,
        "run-tasks":true
      },
      "relationships":{
        "team":{"data":{"id":"team-1","type":"teams"}},
        "workspace":{"data":{"id":"ws-1","type":"workspaces"}}
      }
    }`, &grant)

	assert.Equal(t, "custom", string(grant.Access))
	assert.Equal(t, "apply", string(grant.Runs))
	assert.Equal(t, "write", string(grant.Variables))
	assert.Equal(t, "read-outputs", string(grant.StateVersions))
	assert.Equal(t, "read", string(grant.SentinelMocks))
	assert.True(t, grant.WorkspaceLocking)
	assert.True(t, grant.RunTasks)

	assert.Equal(t, "team-1", relOneID(rec, "team"))
	assert.Equal(t, "ws-1", relOneID(rec, "workspace"))
}

func TestTeamAccessKeepsUnknownRoleVerbatim(t *testing.T) {
	// go-tfe types these as named string types, so a role HashiCorp adds later
	// still arrives verbatim rather than being dropped. That matters because
	// the schema reports `access` as a plain string, and canApply has to fail
	// closed on a role it does not recognise rather than on an empty one.
	var grant tfe.TeamAccess
	decodeRecord(t, `{
      "id":"tws-2","type":"team-workspaces",
      "attributes":{"access":"some-future-role","runs":"apply"}
    }`, &grant)

	assert.Equal(t, "some-future-role", string(grant.Access))
	assert.False(t, terraformCanApply(string(grant.Access), string(grant.Runs)))
}

func TestTerraformCanApply(t *testing.T) {
	tests := []struct {
		access string
		runs   string
		want   bool
	}{
		{"admin", "", true},
		{"write", "", true},
		{"read", "", false},
		{"plan", "", false},
		{"custom", "apply", true},
		{"custom", "plan", false},
		{"custom", "read", false},
		{"custom", "", false},
		// A fixed role's runs value must not leak into the decision, and an
		// unknown role must not be assumed permissive.
		{"read", "apply", false},
		{"plan", "apply", false},
		{"", "apply", false},
		{"unknown-future-role", "apply", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, terraformCanApply(tt.access, tt.runs),
			"access=%q runs=%q", tt.access, tt.runs)
	}
}

// ---------------------------------------------------------------------------
// variables
// ---------------------------------------------------------------------------

func TestVariableAttributeTags(t *testing.T) {
	// The sensitive flag is the single most consequential tag in this
	// provider: a mistyped tag reads false on a sensitive variable and an
	// audit for unmarked secrets passes on a workspace full of them.
	var sensitive tfe.Variable
	decodeRecord(t, `{
      "id":"var-1","type":"vars",
      "attributes":{
        "key":"AWS_SECRET_ACCESS_KEY",
        "value":null,
        "sensitive":true,
        "category":"env",
        "hcl":false,
        "description":"deploy credential",
        "version-id":"abc"
      }
    }`, &sensitive)

	assert.Equal(t, "AWS_SECRET_ACCESS_KEY", sensitive.Key)
	assert.True(t, sensitive.Sensitive)
	assert.Equal(t, "env", string(sensitive.Category))
	assert.False(t, sensitive.HCL)
	assert.Equal(t, "deploy credential", sensitive.Description)

	var plain tfe.Variable
	decodeRecord(t, `{
      "id":"var-2","type":"vars",
      "attributes":{
        "key":"region",
        "value":"us-east-1",
        "sensitive":false,
        "category":"terraform",
        "hcl":true,
        "description":null
      }
    }`, &plain)

	assert.Equal(t, "region", plain.Key)
	assert.False(t, plain.Sensitive)
	assert.Equal(t, "terraform", string(plain.Category))
	assert.True(t, plain.HCL)
	assert.Equal(t, "", plain.Description)
}

func TestVariableNeverCarriesItsValue(t *testing.T) {
	// go-tfe's Variable type does have a Value field, unlike the hand-written
	// record type it replaced. scrubSecrets clears it at the decode
	// chokepoint, so there is no populated value for a later field addition to
	// reach. Re-marshaling what was decoded is the sweep: the secret must not
	// survive the round trip.
	const secret = "super-secret-token-value"
	var variable tfe.Variable
	decodeRecord(t, `{
      "id":"var-3","type":"vars",
      "attributes":{"key":"TOKEN","value":"`+secret+`","sensitive":false,"category":"env","hcl":false}
    }`, &variable)

	assert.Equal(t, "", variable.Value, "the decode chokepoint must clear the variable value")

	out, err := json.Marshal(variable)
	require.NoError(t, err)
	assert.NotContains(t, string(out), secret,
		"the variable value must never be retained by the resource")
	// Negative control: a sweep that passes because nothing was decoded proves
	// nothing at all.
	assert.Contains(t, string(out), "TOKEN", "the record must actually have been read")
}

// ---------------------------------------------------------------------------
// teams and team tokens
// ---------------------------------------------------------------------------

const teamPayload = `{
      "id":"team-Hu9WT3XkVQpMWkBb","type":"teams",
      "attributes":{
        "name":"platform",
        "users-count":7,
        "visibility":"secret",
        "sso-team-id":"sso-platform",
        "allow-member-token-management":true,
        "organization-access":{
          "manage-policies":true,
          "manage-policy-overrides":true,
          "manage-workspaces":true,
          "manage-vcs-settings":true,
          "manage-membership":true,
          "manage-teams":true,
          "manage-organization-access":true,
          "manage-projects":true,
          "manage-run-tasks":true,
          "manage-agent-pools":true,
          "manage-providers":true,
          "manage-modules":true,
          "read-workspaces":true,
          "read-projects":true,
          "access-secret-teams":true
        }
      },
      "relationships":{"organization":{"data":{"id":"acme","type":"organizations"}}}
    }`

func TestTeamAttributeTags(t *testing.T) {
	var team tfe.Team
	rec := decodeRecord(t, teamPayload, &team)

	var gaps tfeTeamGaps
	decodeGaps(t, teamPayload, &gaps)

	assert.Equal(t, "platform", team.Name)
	assert.Equal(t, int64(7), int64(team.UserCount))
	assert.Equal(t, "secret", team.Visibility)
	assert.Equal(t, "sso-platform", derefStr(gaps.SSOTeamID))
	assert.True(t, team.AllowMemberTokenManagement)

	require.NotNil(t, team.OrganizationAccess)
	access := *team.OrganizationAccess
	// Every permission tag is pinned individually: a single mistyped tag
	// would report a fully privileged team as holding no privilege at all.
	assert.True(t, access.ManagePolicies)
	assert.True(t, access.ManagePolicyOverrides)
	assert.True(t, access.ManageWorkspaces)
	assert.True(t, access.ManageVCSSettings)
	assert.True(t, access.ManageMembership)
	assert.True(t, access.ManageTeams)
	assert.True(t, access.ManageOrganizationAccess)
	assert.True(t, access.ManageProjects)
	assert.True(t, access.ManageRunTasks)
	assert.True(t, access.ManageAgentPools)
	assert.True(t, access.ManageProviders)
	assert.True(t, access.ManageModules)
	assert.True(t, access.ReadWorkspaces)
	assert.True(t, access.ReadProjects)
	assert.True(t, access.AccessSecretTeams)

	assert.Equal(t, "acme", relOneID(rec, "organization"))
}

func TestTeamWithoutOrganizationAccess(t *testing.T) {
	const raw = `{
      "id":"team-2","type":"teams",
      "attributes":{"name":"readers","users-count":0,"visibility":"organization","sso-team-id":null}
    }`

	var team tfe.Team
	decodeRecord(t, raw, &team)

	var gaps tfeTeamGaps
	decodeGaps(t, raw, &gaps)

	assert.Nil(t, team.OrganizationAccess)
	assert.Nil(t, gaps.SSOTeamID)
	assert.Equal(t, "", derefStr(gaps.SSOTeamID))
	assert.Equal(t, int64(0), int64(team.UserCount))
}

func TestTeamTokenAttributeTags(t *testing.T) {
	var token tfe.TeamToken
	decodeRecord(t, `{
      "id":"at-Hu9WT3XkVQpMWkBb","type":"authentication-tokens",
      "attributes":{
        "created-at":"2023-01-05T10:00:00Z",
        "last-used-at":"2024-11-02T08:15:00Z",
        "expired-at":"2026-01-05T10:00:00Z",
        "description":"ci pipeline",
        "token":null
      }
    }`, &token)

	require.NotNil(t, timePtr(token.CreatedAt))
	assert.Equal(t, 2023, token.CreatedAt.Year())
	require.NotNil(t, timePtr(token.LastUsedAt))
	assert.Equal(t, 2024, token.LastUsedAt.Year())
	require.NotNil(t, timePtr(token.ExpiredAt))
	assert.Equal(t, "ci pipeline", derefStr(token.Description))
}

func TestTeamTokenNeverUsedOrExpiring(t *testing.T) {
	// A never-used, never-expiring token reports null for both times. Reading
	// them as the zero instant would date a fresh token to year 1 and make a
	// dormancy audit fire on every token in the estate.
	var token tfe.TeamToken
	decodeRecord(t, `{
      "id":"at-2","type":"authentication-tokens",
      "attributes":{"created-at":"2023-01-05T10:00:00Z","last-used-at":null,"expired-at":null,"description":null}
    }`, &token)

	require.NotNil(t, timePtr(token.CreatedAt))
	assert.Nil(t, timePtr(token.LastUsedAt))
	assert.Nil(t, timePtr(token.ExpiredAt))
	assert.Equal(t, "", derefStr(token.Description))
}

func TestTeamTokenNeverCarriesItsSecret(t *testing.T) {
	// go-tfe's TeamToken type does have a Token field, unlike the hand-written
	// record type it replaced.
	const secret = "atlasv1.SUPERSECRETTOKEN"
	var token tfe.TeamToken
	decodeRecord(t, `{
      "id":"at-3","type":"authentication-tokens",
      "attributes":{"created-at":"2023-01-05T10:00:00Z","token":"`+secret+`","description":"ci"}
    }`, &token)

	assert.Equal(t, "", token.Token, "the decode chokepoint must clear the token secret")

	out, err := json.Marshal(token)
	require.NoError(t, err)
	assert.NotContains(t, string(out), secret,
		"a team token secret must never be retained by the resource")
	// Negative control, as above.
	assert.Contains(t, string(out), "ci", "the record must actually have been read")
}

// ---------------------------------------------------------------------------
// policy sets and policies
// ---------------------------------------------------------------------------

const policySetPayload = `{
      "id":"polset-Hu9WT3XkVQpMWkBb","type":"policy-sets",
      "attributes":{
        "name":"production",
        "description":"prod guardrails",
        "kind":"sentinel",
        "global":true,
        "policy-count":3,
        "workspace-count":0,
        "versioned":true,
        "policies-path":"policies/prod",
        "agent-enabled":true,
        "overridable":true,
        "created-at":"2020-01-01T00:00:00Z",
        "updated-at":"2024-01-01T00:00:00Z"
      },
      "relationships":{
        "policies":{"data":[{"id":"pol-1","type":"policies"},{"id":"pol-2","type":"policies"}]},
        "workspaces":{"data":[]}
      }
    }`

func TestPolicySetAttributeTags(t *testing.T) {
	var set tfe.PolicySet
	rec := decodeRecord(t, policySetPayload, &set)

	var gaps tfePolicySetGaps
	decodeGaps(t, policySetPayload, &gaps)

	assert.Equal(t, "production", set.Name)
	assert.Equal(t, "sentinel", string(set.Kind))
	assert.True(t, set.Global)
	assert.Equal(t, int64(3), int64(set.PolicyCount))
	assert.Equal(t, int64(0), int64(set.WorkspaceCount))
	assert.True(t, gaps.Versioned)
	assert.Equal(t, "policies/prod", set.PoliciesPath)
	assert.True(t, set.AgentEnabled)
	require.NotNil(t, set.Overridable)
	assert.True(t, *set.Overridable)
	require.NotNil(t, timePtr(set.CreatedAt))
	require.NotNil(t, timePtr(set.UpdatedAt))

	assert.Equal(t, []string{"pol-1", "pol-2"}, relManyIDs(rec, "policies"))
	assert.Empty(t, relManyIDs(rec, "workspaces"))
}

func TestPolicySetScopedToWorkspaces(t *testing.T) {
	const raw = `{
      "id":"polset-2","type":"policy-sets",
      "attributes":{
        "name":"staging","description":null,"kind":"opa","global":false,
        "policy-count":1,"workspace-count":2,"versioned":false,
        "policies-path":null,"overridable":null
      },
      "relationships":{
        "workspaces":{"data":[{"id":"ws-1","type":"workspaces"},{"id":"ws-2","type":"workspaces"}]}
      }
    }`

	var set tfe.PolicySet
	rec := decodeRecord(t, raw, &set)

	var gaps tfePolicySetGaps
	decodeGaps(t, raw, &gaps)

	assert.False(t, set.Global)
	assert.Equal(t, "opa", string(set.Kind))
	assert.Nil(t, set.Overridable)
	assert.Equal(t, "", set.PoliciesPath)
	assert.False(t, gaps.Versioned)
	assert.Nil(t, timePtr(set.CreatedAt))
	assert.Equal(t, []string{"ws-1", "ws-2"}, relManyIDs(rec, "workspaces"))
	assert.Empty(t, relManyIDs(rec, "policies"))
}

func TestPolicyAttributeTags(t *testing.T) {
	var policy tfe.Policy
	rec := decodeRecord(t, `{
      "id":"pol-Hu9WT3XkVQpMWkBb","type":"policies",
      "attributes":{
        "name":"no-public-buckets",
        "description":"blocks public buckets",
        "kind":"sentinel",
        "enforcement-level":"hard-mandatory",
        "enforce":[{"path":"no-public-buckets.sentinel","mode":"hard-mandatory"}],
        "policy-set-count":2,
        "updated-at":"2024-06-01T00:00:00Z"
      },
      "relationships":{"organization":{"data":{"id":"acme","type":"organizations"}}}
    }`, &policy)

	assert.Equal(t, "no-public-buckets", policy.Name)
	assert.Equal(t, "blocks public buckets", policy.Description)
	assert.Equal(t, "sentinel", string(policy.Kind))
	assert.Equal(t, "hard-mandatory", string(policy.EnforcementLevel))
	require.Len(t, policy.Enforce, 1)
	assert.Equal(t, "hard-mandatory", string(policy.Enforce[0].Mode))
	assert.Equal(t, "no-public-buckets.sentinel", policy.Enforce[0].Path)
	assert.Equal(t, int64(2), int64(policy.PolicySetCount))
	require.NotNil(t, timePtr(policy.UpdatedAt))
	assert.Equal(t, "acme", relOneID(rec, "organization"))
}

func TestTerraformEnforcementLevel(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		enforce []*tfe.Enforcement
		want    string
	}{
		{"top level wins", "advisory", []*tfe.Enforcement{{Mode: "hard-mandatory"}}, "advisory"},
		{"legacy fallback", "", []*tfe.Enforcement{{Path: "p.sentinel", Mode: "soft-mandatory"}}, "soft-mandatory"},
		{"legacy first non-empty", "", []*tfe.Enforcement{{Mode: ""}, {Mode: "hard-mandatory"}}, "hard-mandatory"},
		{"nothing reported", "", nil, ""},
		{"empty enforce list", "", []*tfe.Enforcement{}, ""},
		{"enforce with no modes", "", []*tfe.Enforcement{{Path: "p.sentinel"}}, ""},
		{"opa mandatory", "mandatory", nil, "mandatory"},
		// go-tfe's Enforce is a slice of pointers, so a null entry is
		// representable where the previous value type made it impossible.
		{"nil entry skipped", "", []*tfe.Enforcement{nil, {Mode: "advisory"}}, "advisory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, terraformEnforcementLevel(tt.level, tt.enforce))
		})
	}
}

func TestPolicyLegacyEnforceOnly(t *testing.T) {
	// An older Sentinel policy reports no top-level enforcement-level at all;
	// falling back to the enforce list is what keeps such a policy from
	// reading as unenforced. go-tfe marks Enforce deprecated in favour of
	// EnforcementLevel, which is exactly this preference order.
	var policy tfe.Policy
	decodeRecord(t, `{
      "id":"pol-legacy","type":"policies",
      "attributes":{
        "name":"legacy","kind":"sentinel",
        "enforce":[{"path":"legacy.sentinel","mode":"hard-mandatory"}],
        "policy-set-count":1
      }
    }`, &policy)

	assert.Equal(t, "", string(policy.EnforcementLevel))
	level := terraformEnforcementLevel(string(policy.EnforcementLevel), policy.Enforce)
	assert.Equal(t, "hard-mandatory", level)
	assert.True(t, terraformPolicyBlocking(level))
}

func TestTerraformPolicyBlocking(t *testing.T) {
	tests := []struct {
		level string
		want  bool
	}{
		{"hard-mandatory", true},
		{"mandatory", true},
		{"soft-mandatory", false},
		{"advisory", false},
		{"", false},
		{"Hard-Mandatory", false},
		{"hard_mandatory", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, terraformPolicyBlocking(tt.level), "level %q", tt.level)
	}
}

// ---------------------------------------------------------------------------
// agent pools
// ---------------------------------------------------------------------------

func TestAgentPoolAttributeTags(t *testing.T) {
	var pool tfe.AgentPool
	rec := decodeRecord(t, `{
      "id":"apool-Hu9WT3XkVQpMWkBb","type":"agent-pools",
      "attributes":{
        "name":"prod-agents",
        "agent-count":4,
        "organization-scoped":false,
        "created-at":"2022-03-01T00:00:00Z"
      },
      "relationships":{
        "organization":{"data":{"id":"acme","type":"organizations"}},
        "allowed-workspaces":{"data":[{"id":"ws-1","type":"workspaces"}]}
      }
    }`, &pool)

	assert.Equal(t, "prod-agents", pool.Name)
	assert.Equal(t, int64(4), int64(pool.AgentCount))
	assert.False(t, pool.OrganizationScoped)
	require.NotNil(t, timePtr(pool.CreatedAt))
	assert.Equal(t, "acme", relOneID(rec, "organization"))
	assert.Equal(t, []string{"ws-1"}, relManyIDs(rec, "allowed-workspaces"))
}

func TestAgentPoolOrganizationScoped(t *testing.T) {
	var pool tfe.AgentPool
	rec := decodeRecord(t, `{
      "id":"apool-2","type":"agent-pools",
      "attributes":{"name":"shared","agent-count":0,"organization-scoped":true},
      "relationships":{"allowed-workspaces":{"data":[]}}
    }`, &pool)

	assert.True(t, pool.OrganizationScoped)
	assert.Equal(t, int64(0), int64(pool.AgentCount))
	assert.Nil(t, timePtr(pool.CreatedAt))
	assert.Empty(t, relManyIDs(rec, "allowed-workspaces"))
}

// ---------------------------------------------------------------------------
// relationship helpers
// ---------------------------------------------------------------------------

func TestRelHelpersOnRecordWithoutRelationships(t *testing.T) {
	rec := connection.TfeRecord{ID: "ws-1", Type: "workspaces"}
	assert.Equal(t, "", relOneID(rec, "organization"))
	got := relManyIDs(rec, "workspaces")
	require.NotNil(t, got, "relManyIDs must never return nil")
	assert.Empty(t, got)
}
