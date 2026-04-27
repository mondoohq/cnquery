// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
)

func newTestRoot(t *testing.T, env *testEnv) *mqlCloudflare {
	t.Helper()
	r, err := CreateResource(env.Runtime, "cloudflare", map[string]*llx.RawData{})
	require.NoError(t, err)
	return r.(*mqlCloudflare)
}

func TestZones(t *testing.T) {
	env := setupTestEnv(t)
	root := newTestRoot(t, env)

	env.Mux.HandleFunc("/zones", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("zones"))
	})

	result, err := root.zones()
	require.NoError(t, err)
	require.Len(t, result, 2)

	z1 := result[0].(*mqlCloudflareZone)
	assert.Equal(t, "d56084adb405e0b7e32c52321bf07be6", z1.Id.Data)
	assert.Equal(t, "example.com", z1.Name.Data)
	assert.Equal(t, "active", z1.Status.Data)
	assert.False(t, z1.Paused.Data)
	assert.Equal(t, "full", z1.Type.Data)
	require.NotNil(t, z1.NameServers.Data)
	assert.Equal(t, []any{"ns1.example.com", "ns2.example.com"}, z1.NameServers.Data)
	assert.Equal(t, []any{"ns-old.registrar.example.net"}, z1.OriginalNameServers.Data)

	require.NotNil(t, z1.Account.Data)
	assert.Equal(t, "01a7362d577a6c3019a474fd6f485823", z1.Account.Data.Id.Data)
	assert.Equal(t, "Test Account", z1.Account.Data.Name.Data)

	require.NotNil(t, z1.Owner.Data)
	assert.Equal(t, "owner@example.com", z1.Owner.Data.Email.Data)

	require.NotNil(t, z1.Plan.Data)
	assert.Equal(t, "Free Website", z1.Plan.Data.Name.Data)
	assert.Equal(t, int64(0), z1.Plan.Data.Price.Data)
	assert.True(t, z1.Plan.Data.IsSubscribed.Data)

	z2 := result[1].(*mqlCloudflareZone)
	assert.Equal(t, "abcdef1234567890abcdef1234567890", z2.Id.Data)
	assert.Equal(t, "internal.test", z2.Name.Data)
	assert.Equal(t, "pending", z2.Status.Data)
	assert.Equal(t, "partial", z2.Type.Data)
	assert.Empty(t, z2.NameServers.Data)
}

func TestAccounts(t *testing.T) {
	env := setupTestEnv(t)
	root := newTestRoot(t, env)

	env.Mux.HandleFunc("/accounts", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("accounts"))
	})

	result, err := root.accounts()
	require.NoError(t, err)
	require.Len(t, result, 2)

	a1 := result[0].(*mqlCloudflareAccount)
	assert.Equal(t, "01a7362d577a6c3019a474fd6f485823", a1.Id.Data)
	assert.Equal(t, "Test Account", a1.Name.Data)
	assert.Equal(t, "standard", a1.Type.Data)

	require.NotNil(t, a1.Settings.Data)
	assert.False(t, a1.Settings.Data.EnforceTwoFactor.Data)

	a2 := result[1].(*mqlCloudflareAccount)
	assert.Equal(t, "9d4e6c8f0a2b1d3f5e7c9b1a8f6e4d2c", a2.Id.Data)
	assert.Equal(t, "enterprise", a2.Type.Data)
	assert.True(t, a2.Settings.Data.EnforceTwoFactor.Data)
}
