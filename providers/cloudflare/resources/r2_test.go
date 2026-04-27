// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// createTestR2 creates a cloudflare.r2 resource pre-wired with the test
// account ID so buckets() doesn't need a real zone navigation.
func createTestR2(t *testing.T, env *testEnv) *mqlCloudflareR2 {
	t.Helper()
	r, err := CreateResource(env.Runtime, "cloudflare.r2", map[string]*llx.RawData{
		"__id": llx.StringData("cloudflare.r2@" + testAccountID),
	})
	require.NoError(t, err)
	r2 := r.(*mqlCloudflareR2)
	r2.AccountID = testAccountID
	return r2
}

func TestR2Buckets(t *testing.T) {
	env := setupTestEnv(t)
	r2 := createTestR2(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/r2/buckets", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("r2_buckets"))
	})

	result, err := r2.buckets()
	require.NoError(t, err)
	require.Len(t, result, 3)

	b1 := result[0].(*mqlCloudflareR2Bucket)
	assert.Equal(t, "logs-archive", b1.Name.Data)
	assert.Equal(t, "ENAM", b1.Location.Data)
	require.NotNil(t, b1.CreatedOn.Data)

	b2 := result[1].(*mqlCloudflareR2Bucket)
	assert.Equal(t, "public-assets", b2.Name.Data)
	assert.Equal(t, "WNAM", b2.Location.Data)

	b3 := result[2].(*mqlCloudflareR2Bucket)
	assert.Equal(t, "backups", b3.Name.Data)
	assert.Equal(t, "WEUR", b3.Location.Data)
}

func TestR2BucketPublicAccess_enabled(t *testing.T) {
	env := setupTestEnv(t)
	r2 := createTestR2(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/r2/buckets", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, loadFixture("r2_buckets"))
	})
	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/r2/buckets/public-assets/domains/managed", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, loadFixture("r2_managed_domain_enabled"))
	})

	buckets, err := r2.buckets()
	require.NoError(t, err)
	require.Len(t, buckets, 3)

	bucket := buckets[1].(*mqlCloudflareR2Bucket)
	assert.Equal(t, "public-assets", bucket.Name.Data)

	enabled, err := bucket.publicAccessEnabled()
	require.NoError(t, err)
	assert.True(t, enabled)

	domain, err := bucket.publicAccessDomain()
	require.NoError(t, err)
	assert.Equal(t, "pub-deadbeef1234.r2.dev", domain)
}

func TestR2BucketPublicAccess_disabled(t *testing.T) {
	env := setupTestEnv(t)
	r2 := createTestR2(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/r2/buckets", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, loadFixture("r2_buckets"))
	})
	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/r2/buckets/logs-archive/domains/managed", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, loadFixture("r2_managed_domain_disabled"))
	})

	buckets, err := r2.buckets()
	require.NoError(t, err)
	bucket := buckets[0].(*mqlCloudflareR2Bucket)

	enabled, err := bucket.publicAccessEnabled()
	require.NoError(t, err)
	assert.False(t, enabled)

	domain, err := bucket.publicAccessDomain()
	require.NoError(t, err)
	assert.Equal(t, "pub-cafef00d5678.r2.dev", domain)
}

func TestR2BucketPublicAccess_forbidden(t *testing.T) {
	env := setupTestEnv(t)
	r2 := createTestR2(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/r2/buckets", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, loadFixture("r2_buckets"))
	})
	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/r2/buckets/backups/domains/managed", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		jsonResponse(w, `{"success":false,"errors":[{"code":10000,"message":"Forbidden"}]}`)
	})

	buckets, err := r2.buckets()
	require.NoError(t, err)
	bucket := buckets[2].(*mqlCloudflareR2Bucket)

	// Caller can't read the managed-domain config — fields should be marked
	// unset/null without bubbling an error.
	enabled, err := bucket.publicAccessEnabled()
	require.NoError(t, err)
	assert.False(t, enabled)
	assert.Equal(t, plugin.StateIsNull|plugin.StateIsSet, bucket.PublicAccessEnabled.State)

	domain, err := bucket.publicAccessDomain()
	require.NoError(t, err)
	assert.Equal(t, "", domain)
	assert.Equal(t, plugin.StateIsNull|plugin.StateIsSet, bucket.PublicAccessDomain.State)
}
