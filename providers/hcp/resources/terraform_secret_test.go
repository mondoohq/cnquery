// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
	"go.mondoo.com/mql/v13/providers/hcp/connection"
	"go.mondoo.com/mql/v13/utils/syncx"
)

// Canaries planted in every credential-shaped position the API can return.
// Each is distinct so a failure names which field leaked.
const (
	canaryVariableValue    = "MONDOO-CANARY-VARIABLE-VALUE-0001"
	canarySensitiveValue   = "MONDOO-CANARY-SENSITIVE-VALUE-0002"
	canaryTeamToken        = "MONDOO-CANARY-TEAM-TOKEN-0003"
	canaryOAuthTokenID     = "MONDOO-CANARY-OAUTH-TOKEN-0004"
	canaryWebhookURL       = "MONDOO-CANARY-WEBHOOK-URL-0005"
	canaryUndeclaredAttr   = "MONDOO-CANARY-UNDECLARED-0006"
	canaryConnectingSecret = "MONDOO-CANARY-CONNECTING-TOKEN-0007"
)

// allCanaries is every secret that must not survive into a resource.
var allCanaries = []string{
	canaryVariableValue,
	canarySensitiveValue,
	canaryTeamToken,
	canaryOAuthTokenID,
	canaryWebhookURL,
	canaryUndeclaredAttr,
	canaryConnectingSecret,
}

// terraformTestRuntime wires a real HcpConnection, and therefore a real
// TfeClient, at an httptest server serving the payloads below. The resource
// code under test is the production code: nothing here reimplements a field
// mapping that could drift away from what actually ships.
func terraformTestRuntime(t *testing.T, handler http.Handler) *plugin.Runtime {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	conf := &inventory.Config{
		Options: map[string]string{
			connection.OptionTfeAddress:      srv.URL,
			connection.OptionTfeOrganization: "acme",
			// HCP Terraform is a separate control plane and needs none of
			// this, but the connection refuses to build without an HCP service
			// principal (risk R9 in TESTING-TODO-terraform.md). These are
			// placeholders: nothing in this test reaches an HCP endpoint.
			connection.OptionClientID: "test-client-id",
		},
		Credentials: []*vault.Credential{
			vault.NewPasswordCredential(connection.CredentialClientSecret, "test-client-secret"),
			vault.NewPasswordCredential(connection.CredentialTfeToken, canaryConnectingSecret),
		},
	}
	conn, err := connection.NewHcpConnection(1, &inventory.Asset{}, conf)
	require.NoError(t, err)

	runtime := &plugin.Runtime{
		Connection: conn,
		Resources:  &syncx.Map[plugin.Resource]{},
	}
	runtime.CreateResource = CreateResource
	runtime.NewResource = NewResource
	runtime.GetData = GetData
	runtime.SetData = SetData
	return runtime
}

// terraformSecretHandler serves one workspace, its variables, its team access,
// one team and its tokens, a policy set, a policy and an agent pool. Every
// credential-shaped attribute the API can return carries a canary, and each
// record also carries an undeclared attribute so a future `dict` passthrough
// would be caught too.
func terraformSecretHandler() http.Handler {
	respond := func(w http.ResponseWriter, body string) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(body))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/organizations/acme"):
			respond(w, `{"data":{"id":"acme","type":"organizations","attributes":{
				"name":"acme","email":"ops@acme.example","collaborator-auth-policy":"two_factor_mandatory",
				"session-timeout":20160,"plan-expired":false,
				"undeclared-secret":"`+canaryUndeclaredAttr+`"}}}`)

		case strings.HasSuffix(r.URL.Path, "/organizations/acme/workspaces"):
			respond(w, `{"data":[{"id":"ws-1","type":"workspaces","attributes":{
				"name":"prod","execution-mode":"remote","resource-count":3,
				"created-at":"2021-06-03T14:34:40.492Z",
				"vcs-repo":{"identifier":"acme/infra","branch":"main","service-provider":"github",
					"oauth-token-id":"`+canaryOAuthTokenID+`",
					"webhook-url":"`+canaryWebhookURL+`"},
				"undeclared-secret":"`+canaryUndeclaredAttr+`"}}],
				"meta":{"pagination":{"current-page":1,"total-pages":1}}}`)

		case strings.HasSuffix(r.URL.Path, "/workspaces/ws-1/vars"):
			respond(w, `{"data":[
				{"id":"var-1","type":"vars","attributes":{"key":"AWS_SECRET_ACCESS_KEY",
					"value":"`+canarySensitiveValue+`","sensitive":true,"category":"env","hcl":false,
					"undeclared-secret":"`+canaryUndeclaredAttr+`"}},
				{"id":"var-2","type":"vars","attributes":{"key":"region",
					"value":"`+canaryVariableValue+`","sensitive":false,"category":"terraform","hcl":false}}],
				"meta":{"pagination":{"current-page":1,"total-pages":1}}}`)

		case strings.HasSuffix(r.URL.Path, "/team-workspaces"):
			respond(w, `{"data":[{"id":"tws-1","type":"team-workspaces","attributes":{
				"access":"admin","runs":"apply","undeclared-secret":"`+canaryUndeclaredAttr+`"},
				"relationships":{"team":{"data":{"id":"team-1","type":"teams"}},
					"workspace":{"data":{"id":"ws-1","type":"workspaces"}}}}],
				"meta":{"pagination":{"current-page":1,"total-pages":1}}}`)

		case strings.HasSuffix(r.URL.Path, "/organizations/acme/teams"):
			respond(w, `{"data":[{"id":"team-1","type":"teams","attributes":{
				"name":"platform","users-count":2,"visibility":"organization",
				"organization-access":{"manage-policies":true},
				"undeclared-secret":"`+canaryUndeclaredAttr+`"}}],
				"meta":{"pagination":{"current-page":1,"total-pages":1}}}`)

		case strings.HasSuffix(r.URL.Path, "/organizations/acme/team-tokens"):
			respond(w, `{"data":[{"id":"at-1","type":"authentication-tokens","attributes":{
				"created-at":"2023-01-05T10:00:00Z","description":"ci pipeline",
				"token":"`+canaryTeamToken+`",
				"undeclared-secret":"`+canaryUndeclaredAttr+`"},
				"relationships":{"team":{"data":{"id":"team-1","type":"teams"}}}}],
				"meta":{"pagination":{"current-page":1,"total-pages":1}}}`)

		case strings.HasSuffix(r.URL.Path, "/organizations/acme/policy-sets"):
			respond(w, `{"data":[{"id":"polset-1","type":"policy-sets","attributes":{
				"name":"prod","kind":"sentinel","global":true,"policy-count":1,"versioned":true,
				"vcs-repo":{"identifier":"acme/policies",
					"oauth-token-id":"`+canaryOAuthTokenID+`",
					"webhook-url":"`+canaryWebhookURL+`"},
				"undeclared-secret":"`+canaryUndeclaredAttr+`"}}],
				"meta":{"pagination":{"current-page":1,"total-pages":1}}}`)

		case strings.HasSuffix(r.URL.Path, "/organizations/acme/policies"):
			respond(w, `{"data":[{"id":"pol-1","type":"policies","attributes":{
				"name":"no-public-buckets","kind":"sentinel","enforcement-level":"hard-mandatory",
				"undeclared-secret":"`+canaryUndeclaredAttr+`"}}],
				"meta":{"pagination":{"current-page":1,"total-pages":1}}}`)

		case strings.HasSuffix(r.URL.Path, "/organizations/acme/agent-pools"):
			respond(w, `{"data":[{"id":"apool-1","type":"agent-pools","attributes":{
				"name":"prod-agents","agent-count":1,"organization-scoped":true,
				"undeclared-secret":"`+canaryUndeclaredAttr+`"}}],
				"meta":{"pagination":{"current-page":1,"total-pages":1}}}`)

		default:
			respond(w, `{"data":[],"meta":{"pagination":{"current-page":1,"total-pages":1}}}`)
		}
	})
}

// renderResources dumps every field of every resource, including the internal
// cache fields the generated struct embeds, which is a wider view than any
// report could take of them. The runtime pointer is skipped because it is
// machinery rather than data, and it is not renderable.
func renderResources(t *testing.T, resources ...any) string {
	t.Helper()
	var sb strings.Builder
	var render func(v reflect.Value)
	seen := map[uintptr]bool{}

	render = func(v reflect.Value) {
		switch v.Kind() {
		case reflect.Ptr, reflect.Interface:
			if v.IsNil() {
				return
			}
			if v.Kind() == reflect.Ptr {
				if seen[v.Pointer()] {
					return
				}
				seen[v.Pointer()] = true
			}
			render(v.Elem())
		case reflect.Slice, reflect.Array:
			for i := 0; i < v.Len(); i++ {
				render(v.Index(i))
			}
		case reflect.Map:
			for _, k := range v.MapKeys() {
				fmt.Fprintf(&sb, "%v=", k.Interface())
				render(v.MapIndex(k))
			}
		case reflect.Struct:
			ty := v.Type()
			for i := 0; i < v.NumField(); i++ {
				if ty.Field(i).Name == "MqlRuntime" {
					continue
				}
				f := v.Field(i)
				if !f.CanInterface() {
					// Unexported cache fields still have to be swept: a secret
					// parked on an Internal struct is one accessor away from a
					// report. Read them through an addressable copy.
					if f.CanAddr() {
						f = reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
					} else {
						continue
					}
				}
				fmt.Fprintf(&sb, "%s:", ty.Field(i).Name)
				render(f)
			}
		default:
			if v.CanInterface() {
				fmt.Fprintf(&sb, "%v ", v.Interface())
			}
		}
	}

	for _, res := range resources {
		render(reflect.ValueOf(res))
	}
	return sb.String()
}

// TestTerraformResourcesCarryNoSecrets is the sweep: every SDK-backed record is
// built by the production code from a payload whose every credential position
// holds a canary, and no canary may survive into a resource.
//
// go-tfe's types carry credentials the hand-written record types deliberately
// omitted: Variable.Value, TeamToken.Token, and a VCS connection's
// OAuthTokenID and WebhookURL. They are cleared at the decode chokepoint; this
// proves it end to end rather than at the struct.
func TestTerraformResourcesCarryNoSecrets(t *testing.T) {
	runtime := terraformTestRuntime(t, terraformSecretHandler())

	org, err := fetchMqlHcpTerraformOrganization(runtime, "acme")
	require.NoError(t, err)

	workspaces, err := org.workspaces()
	require.NoError(t, err)
	require.Len(t, workspaces, 1)
	ws := workspaces[0].(*mqlHcpTerraformWorkspace)

	variables, err := ws.variables()
	require.NoError(t, err)
	require.Len(t, variables, 2)

	grants, err := ws.teamAccess()
	require.NoError(t, err)
	require.Len(t, grants, 1)

	teams, err := org.teams()
	require.NoError(t, err)
	require.Len(t, teams, 1)
	team := teams[0].(*mqlHcpTerraformTeam)

	tokens, err := team.tokens()
	require.NoError(t, err)
	require.Len(t, tokens, 1)

	policySets, err := org.policySets()
	require.NoError(t, err)
	require.Len(t, policySets, 1)

	policies, err := org.policies()
	require.NoError(t, err)
	require.Len(t, policies, 1)

	pools, err := org.agentPools()
	require.NoError(t, err)
	require.Len(t, pools, 1)

	rendered := renderResources(t, org, workspaces, variables, grants, teams, tokens,
		policySets, policies, pools)

	for _, canary := range allCanaries {
		assert.NotContains(t, rendered, canary,
			"a credential reached an mql resource; canary %s", canary)
	}

	// Negative control. A sweep that passes because nothing was decoded proves
	// nothing at all, so assert the non-secret values really did come through.
	for _, want := range []string{"acme", "prod", "AWS_SECRET_ACCESS_KEY", "region",
		"platform", "ci pipeline", "no-public-buckets", "prod-agents", "acme/infra"} {
		assert.Contains(t, rendered, want,
			"the sweep read nothing, so its absence assertions are vacuous")
	}
}

// TestTerraformVariableHasNoValueField pins the omission itself. The schema has
// no field for a variable value, for sensitive and non-sensitive variables
// alike; scrubbing is the second line of defence, not the first.
func TestTerraformVariableHasNoValueField(t *testing.T) {
	runtime := terraformTestRuntime(t, terraformSecretHandler())

	res, err := CreateResource(runtime, "hcp.terraform.variable", map[string]*llx.RawData{
		"__id": llx.StringData("hcp.terraform.variable/ws-1/var-1"),
		"id":   llx.StringData("var-1"),
		"key":  llx.StringData("AWS_SECRET_ACCESS_KEY"),
	})
	require.NoError(t, err)

	// Reading a field the schema does not declare must fail rather than
	// resolve. If this ever starts succeeding, a value field has been added.
	for _, forbidden := range []string{"value", "token", "secret", "password", "oauthToken"} {
		data := GetData(res.(plugin.Resource), forbidden, nil)
		require.NotNil(t, data)
		assert.NotEmpty(t, data.Error,
			"hcp.terraform.variable must not expose a %q field", forbidden)
	}
}
