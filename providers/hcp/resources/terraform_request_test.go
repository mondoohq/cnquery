// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"net/http"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// recordedRequest is one request the resource code issued.
type recordedRequest struct {
	path  string
	query map[string][]string
}

// recordingHandler serves the given bodies by path suffix and records every
// request, so a test can assert on the request the resource code made rather
// than only on the answer it got back.
type recordingHandler struct {
	mu       sync.Mutex
	requests []recordedRequest
	bodies   map[string]string
}

func newRecordingHandler(bodies map[string]string) *recordingHandler {
	return &recordingHandler{bodies: bodies}
}

func (h *recordingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	h.requests = append(h.requests, recordedRequest{path: r.URL.Path, query: r.URL.Query()})
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/vnd.api+json")
	// Longest suffix wins, so map iteration order cannot decide which body a
	// path gets and make a test flaky.
	suffixes := make([]string, 0, len(h.bodies))
	for suffix := range h.bodies {
		suffixes = append(suffixes, suffix)
	}
	sort.Slice(suffixes, func(i, j int) bool { return len(suffixes[i]) > len(suffixes[j]) })
	for _, suffix := range suffixes {
		if strings.HasSuffix(r.URL.Path, suffix) {
			_, _ = w.Write([]byte(h.bodies[suffix]))
			return
		}
	}
	_, _ = w.Write([]byte(`{"data":[],"meta":{"pagination":{"current-page":1,"total-pages":1}}}`))
}

func (h *recordingHandler) find(t *testing.T, suffix string) recordedRequest {
	t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, req := range h.requests {
		if strings.HasSuffix(req.path, suffix) {
			return req
		}
	}
	t.Fatalf("no request was issued to a path ending %q; got %v", suffix, h.requests)
	return recordedRequest{}
}

func (h *recordingHandler) paths() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, 0, len(h.requests))
	for _, req := range h.requests {
		out = append(out, req.path)
	}
	return out
}

// emptyPage is a well-formed empty JSON:API collection.
const emptyPage = `{"data":[],"meta":{"pagination":{"current-page":1,"total-pages":1}}}`

// orgForRequests is a minimal organization record.
const orgForRequests = `{"data":{"id":"acme","type":"organizations",` +
	`"attributes":{"name":"acme","collaborator-auth-policy":"two_factor_mandatory"}}}`

// workspaceForRequests is a minimal workspace record.
const workspaceForRequests = `{"data":{"id":"ws-1","type":"workspaces",` +
	`"attributes":{"name":"prod","execution-mode":"remote"},` +
	`"relationships":{"organization":{"data":{"id":"acme","type":"organizations"}}}}}`

// TestTeamAccessIsScopedToTheWorkspace pins the filter that decides whether a
// workspace reports its own team grants or the whole organization's.
//
// If the parameter name were wrong the API would not error, it would ignore the
// filter and return every team-workspace grant in the organization, so this
// workspace would report other workspaces' canApply grants as its own. That is
// a wrong answer rather than an empty one, which is why it is pinned here.
// go-tfe issues the identical filter: its TeamAccessListOptions.WorkspaceID
// carries `url:"filter[workspace][id]"`.
func TestTeamAccessIsScopedToTheWorkspace(t *testing.T) {
	handler := newRecordingHandler(map[string]string{
		"/workspaces/ws-1": workspaceForRequests,
		"/team-workspaces": emptyPage,
	})
	runtime := terraformTestRuntime(t, handler)

	ws, err := initWorkspaceForTest(runtime, "ws-1")
	require.NoError(t, err)

	_, err = ws.teamAccess()
	require.NoError(t, err)

	req := handler.find(t, "/team-workspaces")
	assert.Equal(t, []string{"ws-1"}, req.query["filter[workspace][id]"],
		"team access must be filtered to this workspace, or a workspace reports "+
			"grants belonging to other workspaces")
}

// TestRemoteStateConsumersAsksOnlyForConfiguredOnes pins show_only_configured.
//
// Without it the endpoint returns every workspace in the organization once
// global-remote-state is enabled, which would report an entire estate as
// deliberate consumers of one workspace's state.
func TestRemoteStateConsumersAsksOnlyForConfiguredOnes(t *testing.T) {
	handler := newRecordingHandler(map[string]string{
		"/workspaces/ws-1":                      workspaceForRequests,
		"/relationships/remote-state-consumers": emptyPage,
	})
	runtime := terraformTestRuntime(t, handler)

	ws, err := initWorkspaceForTest(runtime, "ws-1")
	require.NoError(t, err)

	_, err = ws.remoteStateConsumers()
	require.NoError(t, err)

	req := handler.find(t, "/relationships/remote-state-consumers")
	assert.Equal(t, []string{"true"}, req.query["show_only_configured"],
		"remote state consumers must be limited to explicitly configured ones")
}

// TestTeamTokensUseTheOrganizationListing pins the endpoint that decides
// whether a team reports all of its tokens or only one.
//
// GET /teams/:id/authentication-tokens does not exist: that path accepts only
// POST, and neither go-tfe nor the vendor's OpenAPI specification gives it a
// GET. Listing tokens goes through the organization instead. Calling the
// non-existent path would 404 and silently degrade to the single-token
// endpoint, so a team holding several tokens would report one.
func TestTeamTokensUseTheOrganizationListing(t *testing.T) {
	handler := newRecordingHandler(map[string]string{
		"/organizations/acme": orgForRequests,
		"/organizations/acme/teams": `{"data":[{"id":"team-1","type":"teams",` +
			`"attributes":{"name":"platform","users-count":1,"visibility":"organization"}}],` +
			`"meta":{"pagination":{"current-page":1,"total-pages":1}}}`,
		"/organizations/acme/team-tokens": `{"data":[
			{"id":"at-1","type":"authentication-tokens",
			 "attributes":{"created-at":"2023-01-05T10:00:00Z","description":"ci"},
			 "relationships":{"team":{"data":{"id":"team-1","type":"teams"}}}},
			{"id":"at-2","type":"authentication-tokens",
			 "attributes":{"created-at":"2024-01-05T10:00:00Z","description":"release"},
			 "relationships":{"team":{"data":{"id":"team-1","type":"teams"}}}},
			{"id":"at-3","type":"authentication-tokens",
			 "attributes":{"created-at":"2024-06-05T10:00:00Z","description":"other team"},
			 "relationships":{"team":{"data":{"id":"team-2","type":"teams"}}}}],
			"meta":{"pagination":{"current-page":1,"total-pages":1}}}`,
	})
	runtime := terraformTestRuntime(t, handler)

	org, err := fetchMqlHcpTerraformOrganization(runtime, "acme")
	require.NoError(t, err)
	teams, err := org.teams()
	require.NoError(t, err)
	require.Len(t, teams, 1)

	tokens, err := teams[0].(*mqlHcpTerraformTeam).tokens()
	require.NoError(t, err)

	// Both of this team's tokens, and neither of the other team's.
	require.Len(t, tokens, 2, "every token issued to the team must be reported, not just one")
	ids := []string{
		tokens[0].(*mqlHcpTerraformTeamToken).Id.Data,
		tokens[1].(*mqlHcpTerraformTeamToken).Id.Data,
	}
	assert.ElementsMatch(t, []string{"at-1", "at-2"}, ids)

	for _, path := range handler.paths() {
		assert.NotContains(t, path, "authentication-tokens",
			"the plural team token path does not exist and must not be requested")
	}
}

// TestTeamTokensFallBackToTheSingleTokenEndpoint exercises the fallback
// deliberately. A fallback added alongside a working primary never runs in a
// normal test, so it would otherwise ship unverified.
func TestTeamTokensFallBackToTheSingleTokenEndpoint(t *testing.T) {
	handler := newRecordingHandler(map[string]string{
		"/organizations/acme": orgForRequests,
		"/organizations/acme/teams": `{"data":[{"id":"team-1","type":"teams",` +
			`"attributes":{"name":"platform","users-count":1,"visibility":"organization"}}],` +
			`"meta":{"pagination":{"current-page":1,"total-pages":1}}}`,
		"/teams/team-1/authentication-token": `{"data":{"id":"at-legacy",` +
			`"type":"authentication-tokens","attributes":{"created-at":"2020-01-05T10:00:00Z"}}}`,
	})
	// The organization-wide listing 404s, as it does on an installation that
	// predates it.
	notFound := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/team-tokens") {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errors":[{"status":"404","title":"not found"}]}`))
			return
		}
		handler.ServeHTTP(w, r)
	})
	runtime := terraformTestRuntime(t, notFound)

	org, err := fetchMqlHcpTerraformOrganization(runtime, "acme")
	require.NoError(t, err)
	teams, err := org.teams()
	require.NoError(t, err)
	require.Len(t, teams, 1)

	tokens, err := teams[0].(*mqlHcpTerraformTeam).tokens()
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, "at-legacy", tokens[0].(*mqlHcpTerraformTeamToken).Id.Data)
}

// TestOwnersTeamResolvesByName records that the owners team is found by name.
// The organization carries no owners-team relationship in either go-tfe or the
// vendor's OpenAPI specification, so this listing is the only path there is.
func TestOwnersTeamResolvesByName(t *testing.T) {
	handler := newRecordingHandler(map[string]string{
		"/organizations/acme": orgForRequests,
		"/organizations/acme/teams": `{"data":[
			{"id":"team-1","type":"teams","attributes":{"name":"platform","visibility":"organization"}},
			{"id":"team-owners","type":"teams","attributes":{"name":"owners","visibility":"organization"}}],
			"meta":{"pagination":{"current-page":1,"total-pages":1}}}`,
	})
	runtime := terraformTestRuntime(t, handler)

	org, err := fetchMqlHcpTerraformOrganization(runtime, "acme")
	require.NoError(t, err)

	owners, err := org.ownersTeam()
	require.NoError(t, err)
	require.NotNil(t, owners)
	assert.Equal(t, "team-owners", owners.Id.Data)
	assert.Equal(t, "owners", owners.Name.Data)
}

// TestOwnersTeamIsNullWhenNoTeamIsNamedOwners is the absent case: it must
// report null rather than an empty resource, so an audit written on it has
// something to fail against.
func TestOwnersTeamIsNullWhenNoTeamIsNamedOwners(t *testing.T) {
	handler := newRecordingHandler(map[string]string{
		"/organizations/acme": orgForRequests,
		"/organizations/acme/teams": `{"data":[
			{"id":"team-1","type":"teams","attributes":{"name":"platform","visibility":"organization"}}],
			"meta":{"pagination":{"current-page":1,"total-pages":1}}}`,
	})
	runtime := terraformTestRuntime(t, handler)

	org, err := fetchMqlHcpTerraformOrganization(runtime, "acme")
	require.NoError(t, err)

	owners, err := org.ownersTeam()
	require.NoError(t, err)
	assert.Nil(t, owners)
}

// initWorkspaceForTest hydrates a workspace through the production init.
func initWorkspaceForTest(runtime *plugin.Runtime, id string) (*mqlHcpTerraformWorkspace, error) {
	res, err := NewResource(runtime, "hcp.terraform.workspace", map[string]*llx.RawData{
		"id": llx.StringData(id),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHcpTerraformWorkspace), nil
}
