// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessPolicies(t *testing.T) {
	env := setupTestEnv(t)
	one := createTestOne(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/access/policies", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("access_policies"))
	})

	result, err := one.accessPolicies()
	require.NoError(t, err)
	require.Len(t, result, 1)

	policy := result[0].(*mqlCloudflareOneAccessPolicy)
	assert.Equal(t, "policy-001", policy.Id.Data)
	assert.Equal(t, "Allow Engineers", policy.Name.Data)
	assert.Equal(t, "allow", policy.Decision.Data)
	assert.Equal(t, int64(1), policy.Precedence.Data)
}

func TestAccessGroups(t *testing.T) {
	env := setupTestEnv(t)
	one := createTestOne(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/access/groups", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("access_groups"))
	})

	result, err := one.accessGroups()
	require.NoError(t, err)
	require.Len(t, result, 1)

	group := result[0].(*mqlCloudflareOneAccessGroup)
	assert.Equal(t, "group-001", group.Id.Data)
	assert.Equal(t, "Engineering Team", group.Name.Data)
}

func TestServiceTokens(t *testing.T) {
	env := setupTestEnv(t)
	one := createTestOne(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/access/service_tokens", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("service_tokens"))
	})

	result, err := one.serviceTokens()
	require.NoError(t, err)
	require.Len(t, result, 1)

	token := result[0].(*mqlCloudflareOneServiceToken)
	assert.Equal(t, "svctoken-001", token.Id.Data)
	assert.Equal(t, "CI/CD Token", token.Name.Data)
	assert.Equal(t, "client-abc123", token.ClientId.Data)
	assert.Equal(t, "8760h", token.Duration.Data)
	assert.False(t, token.ExpiresAt.Data.IsZero())
	assert.False(t, token.LastSeenAt.Data.IsZero())
}

func TestOrganization(t *testing.T) {
	env := setupTestEnv(t)
	one := createTestOne(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/access/organizations", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("access_organization"))
	})

	result, err := one.organization()
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "My Organization", result.Name.Data)
	assert.Equal(t, "myorg.cloudflareaccess.com", result.AuthDomain.Data)
	assert.Equal(t, true, result.AutoRedirectToIdentity.Data)
	assert.Equal(t, "24h", result.SessionDuration.Data)
	assert.Equal(t, true, result.AllowAuthenticateViaWarp.Data)
}
