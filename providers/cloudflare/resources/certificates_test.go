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

func TestOriginCACertificates(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc("/certificates", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, testZoneID, r.URL.Query().Get("zone_id"))
		jsonResponse(w, loadFixture("origin_ca_certificates"))
	})

	result, err := zone.originCACertificates()
	require.NoError(t, err)
	require.Len(t, result, 1)

	cert := result[0].(*mqlCloudflareZoneOriginCACertificate)
	assert.Equal(t, "origincert-001", cert.Id.Data)
	assert.Equal(t, "origin-rsa", cert.RequestType.Data)
	assert.Equal(t, int64(5475), cert.RequestValidity.Data)
	assert.Len(t, cert.Hostnames.Data, 2)
	assert.False(t, cert.ExpiresAt.Data.IsZero())
}

func TestMtlsCertificates(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/mtls_certificates", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("mtls_certificates"))
	})

	result, err := zone.mtlsCertificates()
	require.NoError(t, err)
	require.Len(t, result, 1)

	cert := result[0].(*mqlCloudflareMtlsCertificate)
	assert.Equal(t, "mtls-001", cert.Id.Data)
	assert.Equal(t, "API Client Cert", cert.Name.Data)
	assert.Equal(t, "CN=My CA", cert.Issuer.Data)
	assert.Equal(t, "SHA256WithRSA", cert.Signature.Data)
	assert.Equal(t, "1234567890ABCDEF", cert.SerialNumber.Data)
	assert.True(t, cert.Ca.Data)
	assert.False(t, cert.ExpiresAt.Data.IsZero())
}

func TestCustomHostnames(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/custom_hostnames", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("custom_hostnames"))
	})

	result, err := zone.customHostnames()
	require.NoError(t, err)
	require.Len(t, result, 1)

	ch := result[0].(*mqlCloudflareZoneCustomHostname)
	assert.Equal(t, "ch-001", ch.Id.Data)
	assert.Equal(t, "app.customer.com", ch.Hostname.Data)
	assert.Equal(t, "origin.example.com", ch.CustomOriginServer.Data)
	assert.Equal(t, "active", ch.Status.Data)
	assert.Equal(t, "active", ch.SslStatus.Data)
	assert.Equal(t, "http", ch.SslMethod.Data)
	assert.Equal(t, "dv", ch.SslType.Data)
}

func TestCustomHostnames_nilSSL(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/custom_hostnames", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, `{
			"success": true,
			"result": [{
				"id": "ch-002",
				"hostname": "nossl.example.com",
				"status": "pending",
				"created_at": "2024-01-01T00:00:00Z"
			}],
			"result_info": {"page":1,"per_page":25,"total_pages":1,"count":1,"total_count":1}
		}`)
	})

	result, err := zone.customHostnames()
	require.NoError(t, err)
	require.Len(t, result, 1)

	ch := result[0].(*mqlCloudflareZoneCustomHostname)
	assert.Equal(t, "", ch.SslStatus.Data)
	assert.Equal(t, "", ch.SslMethod.Data)
	assert.Equal(t, "", ch.SslType.Data)
}
