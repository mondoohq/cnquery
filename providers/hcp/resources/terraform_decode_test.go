// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers/hcp/connection"
)

// decodeRecord unmarshals a JSON:API resource object and decodes its
// attributes, exercising exactly the path the resource code takes.
func decodeRecord(t *testing.T, raw string, out any) connection.TfeRecord {
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
	var attrs tfeOrganizationAttrs
	rec := decodeRecord(t, orgPayload, &attrs)

	assert.Equal(t, "acme", attrs.Name)
	assert.Equal(t, "org-Hu9WT3XkVQpMWkBb", attrs.ExternalID)
	assert.Equal(t, "ops@acme.example", attrs.Email)
	assert.Equal(t, "two_factor_mandatory", attrs.CollaboratorAuthPolicy)
	// Every security-relevant tag is pinned: a mistyped tag would decode to
	// the zero value and report a hardened organization as unhardened.
	assert.True(t, attrs.TwoFactorConformant)
	assert.True(t, attrs.SamlEnabled)
	assert.True(t, attrs.CostEstimationEnabled)
	assert.True(t, attrs.AssessmentsEnforced)
	assert.True(t, attrs.AllowForceDeleteWorkspaces)
	assert.True(t, attrs.PlanExpired)
	assert.Equal(t, "agent", attrs.DefaultExecutionMode)
	require.NotNil(t, attrs.OwnersTeamSamlRoleID)
	assert.Equal(t, "owners-role", *attrs.OwnersTeamSamlRoleID)
	require.NotNil(t, attrs.SessionTimeout)
	assert.Equal(t, int64(20160), *attrs.SessionTimeout)
	require.NotNil(t, attrs.SessionRemember)
	assert.Equal(t, int64(20161), *attrs.SessionRemember)
	require.NotNil(t, attrs.CreatedAt.Time)

	assert.Equal(t, "team-owners", relOneID(rec, "owners-team"))
}

func TestOrganizationOptionalsStayNull(t *testing.T) {
	// The API reports "use the installation default" as null. Reporting 0
	// instead would claim a zero-minute session lifetime, which is a real
	// setting and a wrong one.
	var attrs tfeOrganizationAttrs
	rec := decodeRecord(t, `{
      "id":"acme","type":"organizations",
      "attributes":{
        "name":"acme",
        "session-timeout":null,
        "session-remember":null,
        "owners-team-saml-role-id":null,
        "created-at":null,
        "collaborator-auth-policy":"password"
      }
    }`, &attrs)

	assert.Nil(t, attrs.SessionTimeout)
	assert.Nil(t, attrs.SessionRemember)
	assert.Nil(t, attrs.OwnersTeamSamlRoleID)
	assert.Nil(t, attrs.CreatedAt.Time)
	assert.False(t, attrs.TwoFactorConformant)
	assert.Equal(t, "", relOneID(rec, "owners-team"))
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
	var attrs tfeWorkspaceAttrs
	rec := decodeRecord(t, workspacePayload, &attrs)

	assert.Equal(t, "prod-network", attrs.Name)
	assert.Equal(t, "agent", attrs.ExecutionMode)
	assert.Equal(t, "1.9.5", attrs.TerraformVersion)
	assert.True(t, attrs.AutoApply)
	assert.True(t, attrs.AutoApplyRunTrigger)
	assert.True(t, attrs.Locked)
	assert.True(t, attrs.SpeculativeEnabled)
	assert.True(t, attrs.GlobalRemoteState)
	assert.True(t, attrs.AllowDestroyPlan)
	assert.True(t, attrs.FileTriggersEnabled)
	assert.True(t, attrs.QueueAllRuns)
	assert.True(t, attrs.StructuredRunOutputEnabled)
	assert.True(t, attrs.AssessmentsEnabled)
	assert.Equal(t, int64(42), attrs.ResourceCount)
	assert.Equal(t, []string{"prod", "network"}, attrs.TagNames)
	assert.Equal(t, "envs/prod", derefStr(attrs.WorkingDirectory))
	assert.Equal(t, "production network", derefStr(attrs.Description))
	require.NotNil(t, attrs.CreatedAt.Time)
	require.NotNil(t, attrs.UpdatedAt.Time)

	require.NotNil(t, attrs.VCSRepo)
	assert.Equal(t, "acme/infrastructure", attrs.VCSRepo.Identifier)
	assert.Equal(t, "main", attrs.VCSRepo.Branch)
	assert.Equal(t, "github", attrs.VCSRepo.ServiceProvider)
	assert.True(t, attrs.VCSRepo.IngressSubmodules)

	assert.Equal(t, "acme", relOneID(rec, "organization"))
	assert.Equal(t, "apool-1", relOneID(rec, "agent-pool"))
}

func TestWorkspaceCLIDriven(t *testing.T) {
	// A CLI-driven workspace carries no VCS repo and null strings where the
	// VCS-driven one carries values.
	var attrs tfeWorkspaceAttrs
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
    }`, &attrs)

	assert.Nil(t, attrs.VCSRepo)
	assert.Nil(t, attrs.Description)
	assert.Nil(t, attrs.WorkingDirectory)
	assert.Equal(t, "", derefStr(attrs.Description))
	assert.False(t, attrs.AutoApply)
	assert.False(t, attrs.Locked)
	assert.False(t, attrs.GlobalRemoteState)
	assert.Nil(t, attrs.UpdatedAt.Time)
	assert.Equal(t, "", relOneID(rec, "agent-pool"))
	assert.Empty(t, relManyIDs(rec, "workspaces"))
}

func TestTerraformVCSDriven(t *testing.T) {
	tests := []struct {
		name string
		repo *tfeVCSRepo
		want bool
	}{
		{"no repo", nil, false},
		{"empty repo object", &tfeVCSRepo{}, false},
		{"identifier", &tfeVCSRepo{Identifier: "acme/infra"}, true},
		{"display identifier only", &tfeVCSRepo{DisplayIdentifier: "acme/infra"}, true},
		{"branch without identifier", &tfeVCSRepo{Branch: "main"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, terraformVCSDriven(tt.repo))
		})
	}
}

func TestTerraformVCSIdentifier(t *testing.T) {
	assert.Equal(t, "", terraformVCSIdentifier(nil))
	assert.Equal(t, "", terraformVCSIdentifier(&tfeVCSRepo{}))
	assert.Equal(t, "acme/infra", terraformVCSIdentifier(&tfeVCSRepo{Identifier: "acme/infra"}))
	assert.Equal(t, "acme/display", terraformVCSIdentifier(&tfeVCSRepo{DisplayIdentifier: "acme/display"}))
	// The canonical identifier wins when both are present.
	assert.Equal(t, "acme/infra", terraformVCSIdentifier(&tfeVCSRepo{
		Identifier: "acme/infra", DisplayIdentifier: "acme/display",
	}))
}

// ---------------------------------------------------------------------------
// team access
// ---------------------------------------------------------------------------

func TestTeamAccessAttributeTags(t *testing.T) {
	var attrs tfeTeamAccessAttrs
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
    }`, &attrs)

	assert.Equal(t, "custom", attrs.Access)
	assert.Equal(t, "apply", attrs.Runs)
	assert.Equal(t, "write", attrs.Variables)
	assert.Equal(t, "read-outputs", attrs.StateVersions)
	assert.Equal(t, "read", attrs.SentinelMocks)
	assert.True(t, attrs.WorkspaceLocking)
	assert.True(t, attrs.RunTasks)

	assert.Equal(t, "team-1", relOneID(rec, "team"))
	assert.Equal(t, "ws-1", relOneID(rec, "workspace"))
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
	var sensitive tfeVariableAttrs
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
	assert.Equal(t, "env", sensitive.Category)
	assert.False(t, sensitive.HCL)
	assert.Equal(t, "deploy credential", derefStr(sensitive.Description))

	var plain tfeVariableAttrs
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
	assert.Equal(t, "terraform", plain.Category)
	assert.True(t, plain.HCL)
	assert.Nil(t, plain.Description)
	assert.Equal(t, "", derefStr(plain.Description))
}

func TestVariableNeverCarriesItsValue(t *testing.T) {
	// The record shape deliberately has no field for the variable value, so a
	// value the API does return cannot reach a report. Re-marshaling what was
	// decoded is the sweep: the secret must not survive the round trip.
	const secret = "super-secret-token-value"
	var attrs tfeVariableAttrs
	decodeRecord(t, `{
      "id":"var-3","type":"vars",
      "attributes":{"key":"TOKEN","value":"`+secret+`","sensitive":false,"category":"env","hcl":false}
    }`, &attrs)

	out, err := json.Marshal(attrs)
	require.NoError(t, err)
	assert.NotContains(t, string(out), secret,
		"the variable value must never be retained by the resource")
}

// ---------------------------------------------------------------------------
// teams and team tokens
// ---------------------------------------------------------------------------

func TestTeamAttributeTags(t *testing.T) {
	var attrs tfeTeamAttrs
	rec := decodeRecord(t, `{
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
    }`, &attrs)

	assert.Equal(t, "platform", attrs.Name)
	assert.Equal(t, int64(7), attrs.UsersCount)
	assert.Equal(t, "secret", attrs.Visibility)
	assert.Equal(t, "sso-platform", derefStr(attrs.SSOTeamID))
	assert.True(t, attrs.AllowMemberTokenManagement)

	require.NotNil(t, attrs.OrganizationAccess)
	access := *attrs.OrganizationAccess
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
	var attrs tfeTeamAttrs
	decodeRecord(t, `{
      "id":"team-2","type":"teams",
      "attributes":{"name":"readers","users-count":0,"visibility":"organization","sso-team-id":null}
    }`, &attrs)

	assert.Nil(t, attrs.OrganizationAccess)
	assert.Nil(t, attrs.SSOTeamID)
	assert.Equal(t, "", derefStr(attrs.SSOTeamID))
	assert.Equal(t, int64(0), attrs.UsersCount)
}

func TestTeamTokenAttributeTags(t *testing.T) {
	var attrs tfeTeamTokenAttrs
	decodeRecord(t, `{
      "id":"at-Hu9WT3XkVQpMWkBb","type":"authentication-tokens",
      "attributes":{
        "created-at":"2023-01-05T10:00:00Z",
        "last-used-at":"2024-11-02T08:15:00Z",
        "expired-at":"2026-01-05T10:00:00Z",
        "description":"ci pipeline",
        "token":null
      }
    }`, &attrs)

	require.NotNil(t, attrs.CreatedAt.Time)
	assert.Equal(t, 2023, attrs.CreatedAt.Time.Year())
	require.NotNil(t, attrs.LastUsedAt.Time)
	assert.Equal(t, 2024, attrs.LastUsedAt.Time.Year())
	require.NotNil(t, attrs.ExpiredAt.Time)
	assert.Equal(t, "ci pipeline", derefStr(attrs.Description))
}

func TestTeamTokenNeverUsedOrExpiring(t *testing.T) {
	// A never-used, never-expiring token reports null for both times. Reading
	// them as the zero instant would date a fresh token to year 1 and make a
	// dormancy audit fire on every token in the estate.
	var attrs tfeTeamTokenAttrs
	decodeRecord(t, `{
      "id":"at-2","type":"authentication-tokens",
      "attributes":{"created-at":"2023-01-05T10:00:00Z","last-used-at":null,"expired-at":null,"description":null}
    }`, &attrs)

	require.NotNil(t, attrs.CreatedAt.Time)
	assert.Nil(t, attrs.LastUsedAt.Time)
	assert.Nil(t, attrs.ExpiredAt.Time)
	assert.Equal(t, "", derefStr(attrs.Description))
}

func TestTeamTokenNeverCarriesItsSecret(t *testing.T) {
	const secret = "atlasv1.SUPERSECRETTOKEN"
	var attrs tfeTeamTokenAttrs
	decodeRecord(t, `{
      "id":"at-3","type":"authentication-tokens",
      "attributes":{"created-at":"2023-01-05T10:00:00Z","token":"`+secret+`"}
    }`, &attrs)

	out, err := json.Marshal(attrs)
	require.NoError(t, err)
	assert.NotContains(t, string(out), secret,
		"a team token secret must never be retained by the resource")
}

// ---------------------------------------------------------------------------
// policy sets and policies
// ---------------------------------------------------------------------------

func TestPolicySetAttributeTags(t *testing.T) {
	var attrs tfePolicySetAttrs
	rec := decodeRecord(t, `{
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
    }`, &attrs)

	assert.Equal(t, "production", attrs.Name)
	assert.Equal(t, "sentinel", attrs.Kind)
	assert.True(t, attrs.Global)
	assert.Equal(t, int64(3), attrs.PolicyCount)
	assert.Equal(t, int64(0), attrs.WorkspaceCount)
	assert.True(t, attrs.Versioned)
	assert.Equal(t, "policies/prod", derefStr(attrs.PoliciesPath))
	assert.True(t, attrs.AgentEnabled)
	require.NotNil(t, attrs.Overridable)
	assert.True(t, *attrs.Overridable)
	require.NotNil(t, attrs.CreatedAt.Time)
	require.NotNil(t, attrs.UpdatedAt.Time)

	assert.Equal(t, []string{"pol-1", "pol-2"}, relManyIDs(rec, "policies"))
	assert.Empty(t, relManyIDs(rec, "workspaces"))
}

func TestPolicySetScopedToWorkspaces(t *testing.T) {
	var attrs tfePolicySetAttrs
	rec := decodeRecord(t, `{
      "id":"polset-2","type":"policy-sets",
      "attributes":{
        "name":"staging","description":null,"kind":"opa","global":false,
        "policy-count":1,"workspace-count":2,"versioned":false,
        "policies-path":null,"overridable":null
      },
      "relationships":{
        "workspaces":{"data":[{"id":"ws-1","type":"workspaces"},{"id":"ws-2","type":"workspaces"}]}
      }
    }`, &attrs)

	assert.False(t, attrs.Global)
	assert.Equal(t, "opa", attrs.Kind)
	assert.Nil(t, attrs.Overridable)
	assert.Nil(t, attrs.PoliciesPath)
	assert.Nil(t, attrs.CreatedAt.Time)
	assert.Equal(t, []string{"ws-1", "ws-2"}, relManyIDs(rec, "workspaces"))
	assert.Empty(t, relManyIDs(rec, "policies"))
}

func TestPolicyAttributeTags(t *testing.T) {
	var attrs tfePolicyAttrs
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
    }`, &attrs)

	assert.Equal(t, "no-public-buckets", attrs.Name)
	assert.Equal(t, "blocks public buckets", derefStr(attrs.Description))
	assert.Equal(t, "sentinel", attrs.Kind)
	assert.Equal(t, "hard-mandatory", attrs.EnforcementLevel)
	require.Len(t, attrs.Enforce, 1)
	assert.Equal(t, "hard-mandatory", attrs.Enforce[0].Mode)
	assert.Equal(t, "no-public-buckets.sentinel", attrs.Enforce[0].Path)
	assert.Equal(t, int64(2), attrs.PolicySetCount)
	require.NotNil(t, attrs.UpdatedAt.Time)
	assert.Equal(t, "acme", relOneID(rec, "organization"))
}

func TestTerraformEnforcementLevel(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		enforce []tfePolicyEnforcement
		want    string
	}{
		{"top level wins", "advisory", []tfePolicyEnforcement{{Mode: "hard-mandatory"}}, "advisory"},
		{"legacy fallback", "", []tfePolicyEnforcement{{Path: "p.sentinel", Mode: "soft-mandatory"}}, "soft-mandatory"},
		{"legacy first non-empty", "", []tfePolicyEnforcement{{Mode: ""}, {Mode: "hard-mandatory"}}, "hard-mandatory"},
		{"nothing reported", "", nil, ""},
		{"empty enforce list", "", []tfePolicyEnforcement{}, ""},
		{"enforce with no modes", "", []tfePolicyEnforcement{{Path: "p.sentinel"}}, ""},
		{"opa mandatory", "mandatory", nil, "mandatory"},
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
	// reading as unenforced.
	var attrs tfePolicyAttrs
	decodeRecord(t, `{
      "id":"pol-legacy","type":"policies",
      "attributes":{
        "name":"legacy","kind":"sentinel",
        "enforce":[{"path":"legacy.sentinel","mode":"hard-mandatory"}],
        "policy-set-count":1
      }
    }`, &attrs)

	assert.Equal(t, "", attrs.EnforcementLevel)
	level := terraformEnforcementLevel(attrs.EnforcementLevel, attrs.Enforce)
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
	var attrs tfeAgentPoolAttrs
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
    }`, &attrs)

	assert.Equal(t, "prod-agents", attrs.Name)
	assert.Equal(t, int64(4), attrs.AgentCount)
	assert.False(t, attrs.OrganizationScoped)
	require.NotNil(t, attrs.CreatedAt.Time)
	assert.Equal(t, "acme", relOneID(rec, "organization"))
	assert.Equal(t, []string{"ws-1"}, relManyIDs(rec, "allowed-workspaces"))
}

func TestAgentPoolOrganizationScoped(t *testing.T) {
	var attrs tfeAgentPoolAttrs
	rec := decodeRecord(t, `{
      "id":"apool-2","type":"agent-pools",
      "attributes":{"name":"shared","agent-count":0,"organization-scoped":true},
      "relationships":{"allowed-workspaces":{"data":[]}}
    }`, &attrs)

	assert.True(t, attrs.OrganizationScoped)
	assert.Equal(t, int64(0), attrs.AgentCount)
	assert.Nil(t, attrs.CreatedAt.Time)
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
