// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers/newrelic/connection"
)

// -----------------------------------------------------------------------------
// cursor walking
// -----------------------------------------------------------------------------

func TestWalkPagesHappyPath(t *testing.T) {
	cursors := []string{}
	pages := []string{"c1", "c2", ""}

	err := walkPages("things", func(cursor string) (string, error) {
		cursors = append(cursors, cursor)
		return pages[len(cursors)-1], nil
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"", "c1", "c2"}, cursors, "the first page asks for no cursor")
}

func TestWalkPagesSinglePage(t *testing.T) {
	calls := 0
	err := walkPages("things", func(cursor string) (string, error) {
		calls++
		return "", nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

// A server that ignores the cursor answers with the same page forever. Stopping
// early would under-report the collection and continuing would multiply every
// record, and neither is distinguishable from a correct answer by looking at
// the result, so the walk refuses to produce one.
func TestWalkPagesStuckCursor(t *testing.T) {
	calls := 0
	err := walkPages("things", func(cursor string) (string, error) {
		calls++
		return "same-cursor", nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "same page cursor")
	assert.Contains(t, err.Error(), "things")
	assert.Equal(t, 2, calls, "the repeat is caught on the second request, not after a thousand")
}

func TestWalkPagesCycle(t *testing.T) {
	sequence := []string{"c1", "c2", "c3", "c1"}
	calls := 0

	err := walkPages("things", func(cursor string) (string, error) {
		next := sequence[calls]
		calls++
		return next, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "same page cursor")
}

func TestWalkPagesPropagatesError(t *testing.T) {
	sentinel := errors.New("boom")
	err := walkPages("things", func(cursor string) (string, error) {
		return "", sentinel
	})
	assert.ErrorIs(t, err, sentinel)
}

func TestWalkPagesRespectsPageCap(t *testing.T) {
	calls := 0
	err := walkPages("things", func(cursor string) (string, error) {
		calls++
		return fmt.Sprintf("cursor-%d", calls), nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "did not finish within")
	assert.Equal(t, maxPages, calls)
}

func TestCursorVars(t *testing.T) {
	assert.Equal(t, map[string]any{}, cursorVars("", nil))
	assert.Equal(t, map[string]any{"cursor": "c1"}, cursorVars("c1", nil))
	assert.Equal(t, map[string]any{"accountId": 42}, cursorVars("", map[string]any{"accountId": 42}))
	assert.Equal(t, map[string]any{"accountId": 42, "cursor": "c1"}, cursorVars("c1", map[string]any{"accountId": 42}))

	// The caller's map must not be modified, or the second page of a walk would
	// carry the first page's cursor.
	extra := map[string]any{"accountId": 42}
	cursorVars("c1", extra)
	assert.Equal(t, map[string]any{"accountId": 42}, extra)
}

// -----------------------------------------------------------------------------
// test server
// -----------------------------------------------------------------------------

// scriptedServer answers each request with the next canned response and records
// the variables it was called with, so a pagination walk can be asserted from
// both ends.
type scriptedServer struct {
	responses []string
	requests  []map[string]any
}

func newScriptedClient(t *testing.T, responses ...string) (*connection.Client, *scriptedServer) {
	t.Helper()

	script := &scriptedServer{responses: responses}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body struct {
			Variables map[string]any `json:"variables"`
		}
		_ = json.Unmarshal(raw, &body)
		script.requests = append(script.requests, body.Variables)

		idx := len(script.requests) - 1
		if idx >= len(script.responses) {
			http.Error(w, `{"errors":[{"message":"unexpected extra request"}]}`, http.StatusInternalServerError)
			return
		}
		_, _ = io.WriteString(w, script.responses[idx])
	}))
	t.Cleanup(srv.Close)

	return connection.NewClient(srv.Client(), srv.URL, "NRAK-TEST"), script
}

// -----------------------------------------------------------------------------
// API keys
// -----------------------------------------------------------------------------

func keySearchPage(nextCursor string, ids ...string) string {
	keys := ""
	for i, id := range ids {
		if i > 0 {
			keys += ","
		}
		keys += fmt.Sprintf(`{"id":%q,"name":%q,"type":"INGEST","ingestType":"LICENSE","createdAt":1709288430,"accountId":1234567}`, id, "key-"+id)
	}
	return fmt.Sprintf(`{"data":{"actor":{"apiAccess":{"keySearch":{"nextCursor":%q,"count":9,"keys":[%s]}}}}}`, nextCursor, keys)
}

func TestFetchAPIKeysWalksPages(t *testing.T) {
	client, script := newScriptedClient(t,
		keySearchPage("c1", "k1", "k2"),
		keySearchPage("c2", "k3"),
		keySearchPage("", "k4"),
	)

	keys, err := fetchAPIKeys(context.Background(), client, 1234567)
	require.NoError(t, err)
	require.Len(t, keys, 4)
	assert.Equal(t, []string{"k1", "k2", "k3", "k4"}, []string{keys[0].ID, keys[1].ID, keys[2].ID, keys[3].ID})

	require.Len(t, script.requests, 3)
	assert.Nil(t, script.requests[0]["cursor"], "the first page asks for no cursor")
	assert.Equal(t, float64(1234567), script.requests[0]["accountId"])
	assert.Equal(t, "c1", script.requests[1]["cursor"])
	assert.Equal(t, "c2", script.requests[2]["cursor"])
	assert.Equal(t, float64(1234567), script.requests[2]["accountId"], "the account scope is carried on every page")
}

func TestFetchAPIKeysStuckCursorFails(t *testing.T) {
	client, _ := newScriptedClient(t,
		keySearchPage("stuck", "k1"),
		keySearchPage("stuck", "k1"),
	)

	_, err := fetchAPIKeys(context.Background(), client, 1234567)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "same page cursor")
}

func TestFetchAPIKeysForbiddenSurfacesAsError(t *testing.T) {
	client, _ := newScriptedClient(t,
		`{"data":{"actor":{"apiAccess":null}},"errors":[{"message":"no","extensions":{"errorClass":"FORBIDDEN"}}]}`,
	)

	keys, err := fetchAPIKeys(context.Background(), client, 1234567)
	require.Error(t, err)
	assert.Nil(t, keys, "a refusal must not read as an account with no keys")
	assert.True(t, connection.IsForbidden(err))
}

// -----------------------------------------------------------------------------
// authentication domains and users
// -----------------------------------------------------------------------------

func TestFetchAuthDomainsWithNestedUserPagination(t *testing.T) {
	client, script := newScriptedClient(t,
		`{"data":{"actor":{"organization":{"userManagement":{"authenticationDomains":{
			"nextCursor":"",
			"totalCount":1,
			"authenticationDomains":[{
				"id":"d1","name":"Corp","provisioningType":"SCIM",
				"users":{"nextCursor":"u1","totalCount":3,"users":[{"id":"1001","email":"a@example.com"}]}
			}]
		}}}}}}`,
		`{"data":{"actor":{"organization":{"userManagement":{"authenticationDomains":{
			"authenticationDomains":[{
				"id":"d1",
				"users":{"nextCursor":"u2","users":[{"id":"1002","email":"b@example.com"}]}
			}]
		}}}}}}`,
		`{"data":{"actor":{"organization":{"userManagement":{"authenticationDomains":{
			"authenticationDomains":[{
				"id":"d1",
				"users":{"nextCursor":"","users":[{"id":"1003","email":"c@example.com"}]}
			}]
		}}}}}}`,
	)

	domains, err := fetchAuthDomainsWithUsers(context.Background(), client)
	require.NoError(t, err)
	require.Len(t, domains, 1)
	require.Len(t, domains[0].Users.Users, 3, "every user page has to be collected")

	for _, user := range domains[0].Users.Users {
		assert.Equal(t, "d1", user.domainID, "each user carries the domain it was listed under")
	}

	require.Len(t, script.requests, 3)
	assert.Equal(t, "u1", script.requests[1]["cursor"])
	assert.Equal(t, []any{"d1"}, script.requests[1]["domainIds"])
}

// A follow-up page that no longer returns the domain is a filter that stopped
// matching, not the end of the list. Reading it as the end would drop every
// remaining user in the domain.
func TestFetchAuthDomainsFailsWhenDomainDisappears(t *testing.T) {
	client, _ := newScriptedClient(t,
		`{"data":{"actor":{"organization":{"userManagement":{"authenticationDomains":{
			"nextCursor":"",
			"authenticationDomains":[{"id":"d1","users":{"nextCursor":"u1","users":[{"id":"1001"}]}}]
		}}}}}}`,
		`{"data":{"actor":{"organization":{"userManagement":{"authenticationDomains":{
			"authenticationDomains":[]
		}}}}}}`,
	)

	_, err := fetchAuthDomainsWithUsers(context.Background(), client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stopped returning authentication domain")
}

func TestFetchAuthDomainsStuckUserCursorFails(t *testing.T) {
	page := `{"data":{"actor":{"organization":{"userManagement":{"authenticationDomains":{
		"authenticationDomains":[{"id":"d1","users":{"nextCursor":"u1","users":[{"id":"1002"}]}}]
	}}}}}}`

	client, _ := newScriptedClient(t,
		`{"data":{"actor":{"organization":{"userManagement":{"authenticationDomains":{
			"nextCursor":"",
			"authenticationDomains":[{"id":"d1","users":{"nextCursor":"u1","users":[{"id":"1001"}]}}]
		}}}}}}`,
		page,
	)

	_, err := fetchAuthDomainsWithUsers(context.Background(), client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "same page cursor")
}

// A user's group list that spills over one page cannot be completed through a
// documented follow-up query, so it is reported rather than truncated. A
// truncated membership list makes an over-privileged account look narrower than
// it is.
func TestFetchAuthDomainsRefusesUnpageableUserGroups(t *testing.T) {
	client, _ := newScriptedClient(t,
		`{"data":{"actor":{"organization":{"userManagement":{"authenticationDomains":{
			"nextCursor":"",
			"authenticationDomains":[{
				"id":"d1",
				"users":{"nextCursor":"","users":[
					{"id":"1001","groups":{"nextCursor":"more-groups","groups":[{"id":"g1"}]}}
				]}
			}]
		}}}}}}`,
	)

	_, err := fetchAuthDomainsWithUsers(context.Background(), client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "more groups than one page returns")
}

func TestFetchAdminAuthDomains(t *testing.T) {
	client, _ := newScriptedClient(t,
		`{"data":{"customerAdministration":{"authenticationDomains":{
			"nextCursor":"a1",
			"items":[{"id":"d1","name":"Corp","authenticationType":"SAML_SSO","provisioningType":"SCIM","organizationId":"org-1"}]
		}}}}`,
		`{"data":{"customerAdministration":{"authenticationDomains":{
			"nextCursor":"",
			"items":[{"id":"d2","name":"Contractors","authenticationType":"PASSWORD","provisioningType":"MANUAL","organizationId":"org-1"}]
		}}}}`,
	)

	items, err := fetchAdminAuthDomains(context.Background(), client)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "SAML_SSO", items[0].AuthenticationType)
	assert.Equal(t, "PASSWORD", items[1].AuthenticationType)
}

// -----------------------------------------------------------------------------
// groups and access grants
// -----------------------------------------------------------------------------

func TestFetchGroupsWithGrants(t *testing.T) {
	client, script := newScriptedClient(t,
		`{"data":{"actor":{"organization":{"authorizationManagement":{"authenticationDomains":{
			"nextCursor":"",
			"authenticationDomains":[{
				"id":"d1","name":"Corp",
				"groups":{"nextCursor":"g-next","totalCount":2,"groups":[
					{"id":"gA","displayName":"Admins","roles":{"nextCursor":"","roles":[
						{"id":"1","name":"admin","displayName":"Administrator","type":"STANDARD","roleId":99999,"accountId":1234567}
					]}}
				]}
			}]
		}}}}}}`,
		`{"data":{"actor":{"organization":{"authorizationManagement":{"authenticationDomains":{
			"authenticationDomains":[{
				"id":"d1",
				"groups":{"nextCursor":"","groups":[
					{"id":"gB","displayName":"Readers","roles":{"nextCursor":"","roles":[]}}
				]}
			}]
		}}}}}}`,
	)

	groups, err := fetchGroupsWithGrants(context.Background(), client)
	require.NoError(t, err)
	require.Len(t, groups, 2)

	assert.Equal(t, "gA", groups[0].ID)
	assert.Equal(t, "d1", groups[0].domainID, "each group carries the domain it was listed under")
	require.Len(t, groups[0].Roles.Roles, 1)
	assert.Equal(t, 99999, groups[0].Roles.Roles[0].RoleID)

	assert.Equal(t, "gB", groups[1].ID)
	assert.Equal(t, "d1", groups[1].domainID)

	require.Len(t, script.requests, 2)
	assert.Equal(t, "g-next", script.requests[1]["cursor"])
}

// An access grant list that spills over one page cannot be completed through a
// documented follow-up query. Truncating it would drop grants and make an
// over-privileged group look narrower than it is, so it is reported instead.
func TestFetchGroupsRefusesUnpageableGrants(t *testing.T) {
	client, _ := newScriptedClient(t,
		`{"data":{"actor":{"organization":{"authorizationManagement":{"authenticationDomains":{
			"nextCursor":"",
			"authenticationDomains":[{
				"id":"d1",
				"groups":{"nextCursor":"","groups":[
					{"id":"gA","displayName":"Admins","roles":{"nextCursor":"more-roles","roles":[{"id":"1","roleId":1}]}}
				]}
			}]
		}}}}}}`,
	)

	_, err := fetchGroupsWithGrants(context.Background(), client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "more access grants than one page returns")
}

// -----------------------------------------------------------------------------
// in-band errors
// -----------------------------------------------------------------------------

// The drop rule list answers with an error object rather than a GraphQL error.
// A page carrying one holds no rules, and reporting that as "nothing is being
// discarded" is the opposite of "unknown".
func TestFetchDropRulesInBandErrorIsNotAnEmptyList(t *testing.T) {
	client, _ := newScriptedClient(t,
		`{"data":{"actor":{"account":{"nrqlDropRules":{"list":{
			"error":{"description":"account not enabled","reason":"FEATURE_FLAG_DISABLED"},
			"rules":[]
		}}}}}}`,
	)

	rules, err := fetchDropRules(context.Background(), client, 1234567)
	require.Error(t, err)
	assert.Nil(t, rules)
	assert.Contains(t, err.Error(), "FEATURE_FLAG_DISABLED")
	assert.Contains(t, err.Error(), "account not enabled")
}

func TestFetchDropRulesSuccess(t *testing.T) {
	client, _ := newScriptedClient(t,
		`{"data":{"actor":{"account":{"nrqlDropRules":{"list":{
			"error":null,
			"rules":[{"id":"r1","action":"DROP_DATA","nrql":"SELECT * FROM Log","accountId":1234567,"createdBy":1001}]
		}}}}}}`,
	)

	rules, err := fetchDropRules(context.Background(), client, 1234567)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	assert.Equal(t, "DROP_DATA", rules[0].Action)
}

func TestFetchNotificationDestinationsInBandErrorIsNotAnEmptyList(t *testing.T) {
	client, _ := newScriptedClient(t,
		`{"data":{"actor":{"account":{"aiNotifications":{"destinations":{
			"nextCursor":"",
			"error":{"description":"no access","details":"missing capability","type":"EXTERNAL_SERVER_ERROR"},
			"entities":[]
		}}}}}}`,
	)

	destinations, err := fetchNotificationDestinations(context.Background(), client, 1234567)
	require.Error(t, err)
	assert.Nil(t, destinations)
	assert.Contains(t, err.Error(), "EXTERNAL_SERVER_ERROR")
}

func TestFetchNotificationChannelsInBandErrorIsNotAnEmptyList(t *testing.T) {
	client, _ := newScriptedClient(t,
		`{"data":{"actor":{"account":{"aiNotifications":{"channels":{
			"nextCursor":"",
			"error":{"description":"no access","type":"EXTERNAL_SERVER_ERROR"},
			"entities":[]
		}}}}}}`,
	)

	_, err := fetchNotificationChannels(context.Background(), client, 1234567)
	require.Error(t, err)
}

// A null error object is the normal case and must not be mistaken for a
// failure, or every healthy account would report an error.
func TestNotificationsErrorIsSet(t *testing.T) {
	var absent *apiNotificationsError
	assert.False(t, absent.isSet())
	assert.NoError(t, absent.asError("things"))

	assert.False(t, (&apiNotificationsError{}).isSet())
	assert.True(t, (&apiNotificationsError{Description: "x"}).isSet())
	assert.True(t, (&apiNotificationsError{Details: "x"}).isSet())
	assert.True(t, (&apiNotificationsError{Type: "x"}).isSet())
}

func TestDropRulesErrorIsSet(t *testing.T) {
	var absent *apiDropRulesError
	assert.False(t, absent.isSet())
	assert.False(t, (&apiDropRulesError{}).isSet())
	assert.True(t, (&apiDropRulesError{Reason: "FEATURE_FLAG_DISABLED"}).isSet())
	assert.True(t, (&apiDropRulesError{Description: "nope"}).isSet())
}

// -----------------------------------------------------------------------------
// other collections
// -----------------------------------------------------------------------------

func TestFetchAlertPoliciesAndConditions(t *testing.T) {
	client, _ := newScriptedClient(t,
		`{"data":{"actor":{"account":{"alerts":{"policiesSearch":{
			"nextCursor":"","totalCount":1,
			"policies":[{"id":"p1","name":"Prod","incidentPreference":"PER_CONDITION","accountId":1234567}]
		}}}}}}`,
	)
	policies, err := fetchAlertPolicies(context.Background(), client, 1234567)
	require.NoError(t, err)
	require.Len(t, policies, 1)
	assert.Equal(t, "PER_CONDITION", policies[0].IncidentPreference)

	client2, _ := newScriptedClient(t,
		`{"data":{"actor":{"account":{"alerts":{"nrqlConditionsSearch":{
			"nextCursor":"","totalCount":1,
			"nrqlConditions":[{"id":"c1","name":"Errors","enabled":true,"type":"STATIC","policyId":"p1","nrql":{"query":"SELECT 1"}}]
		}}}}}}`,
	)
	conditions, err := fetchAlertConditions(context.Background(), client2, 1234567)
	require.NoError(t, err)
	require.Len(t, conditions, 1)
	assert.Equal(t, "p1", conditions[0].PolicyID)
}

func TestFetchRoles(t *testing.T) {
	client, _ := newScriptedClient(t,
		`{"data":{"actor":{"organization":{"authorizationManagement":{"roles":{
			"nextCursor":"","totalCount":2,
			"roles":[
				{"id":"1","name":"admin","displayName":"Administrator","type":"STANDARD","scope":"ACCOUNT"},
				{"id":"2","name":"org_owner","displayName":"Organization owner","type":"STANDARD","scope":"ORGANIZATION"}
			]
		}}}}}}`,
	)

	roles, err := fetchRoles(context.Background(), client)
	require.NoError(t, err)
	require.Len(t, roles, 2)
	assert.Equal(t, "ORGANIZATION", roles[1].Scope)
}

func TestFetchAccountsAndOrganization(t *testing.T) {
	client, _ := newScriptedClient(t, `{"data":{"actor":{"accounts":[{"id":1234567,"name":"Prod"}]}}}`)
	accounts, err := fetchAccounts(context.Background(), client)
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	assert.Equal(t, 1234567, accounts[0].ID)

	client2, _ := newScriptedClient(t, `{"data":{"actor":{"organization":{"id":"org-1","name":"Example"}}}}`)
	org, err := fetchOrganization(context.Background(), client2)
	require.NoError(t, err)
	assert.Equal(t, "org-1", org.ID)
}

// A key that cannot see an organization gets a null back rather than an error.
// Reporting an organization with an empty ID would let every organization-wide
// check run against nothing.
func TestFetchOrganizationEmptyIsAnError(t *testing.T) {
	client, _ := newScriptedClient(t, `{"data":{"actor":{"organization":null}}}`)
	_, err := fetchOrganization(context.Background(), client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no organization")
}

func TestFetchRetentionRules(t *testing.T) {
	client, _ := newScriptedClient(t,
		`{"data":{"actor":{"account":{"dataManagement":{"eventRetentionRules":[
			{"id":"1","namespace":"Log","retentionInDays":30,"createdAt":"2024-01-02T03:04:05Z","createdById":"1001","deletedAt":null}
		]}}}}}`,
	)

	rules, err := fetchRetentionRules(context.Background(), client, 1234567)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	assert.True(t, isRetentionRuleActive(rules[0]))
}
