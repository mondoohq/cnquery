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

func TestEmailRouting(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/email/routing", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("email_routing_settings"))
	})

	er, err := zone.emailRouting()
	require.NoError(t, err)
	require.NotNil(t, er)

	assert.True(t, er.Enabled.Data)
	assert.Equal(t, "ready", er.Status.Data)
	assert.Equal(t, "example.com", er.Name.Data)
	require.NotNil(t, er.CreatedAt.Data)
	require.NotNil(t, er.ModifiedAt.Data)
}

func TestEmailRouting_unavailable(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/email/routing", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		jsonResponse(w, `{"success":false,"errors":[{"code":10000,"message":"Forbidden"}]}`)
	})

	er, err := zone.emailRouting()
	require.NoError(t, err, "Forbidden should be treated as 'not configured', not an error")
	assert.Nil(t, er)
}

func TestEmailRouting_mxAndSpfConfigured(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/email/routing", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, loadFixture("email_routing_settings"))
	})
	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/email/routing/dns", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, loadFixture("email_routing_dns"))
	})

	er, err := zone.emailRouting()
	require.NoError(t, err)
	require.NotNil(t, er)

	mx, err := er.mxConfigured()
	require.NoError(t, err)
	assert.True(t, mx, "fixture has 3 MX records, mxConfigured should be true")

	spf, err := er.spfConfigured()
	require.NoError(t, err)
	assert.True(t, spf, "fixture has TXT v=spf1 ..., spfConfigured should be true")
}

func TestEmailRouting_dmarcConfigured(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/email/routing", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, loadFixture("email_routing_settings"))
	})
	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/dns_records", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		// Verify the DNS lookup is scoped to the dmarc record and TXT type
		assert.Equal(t, "_dmarc.example.com", r.URL.Query().Get("name"))
		assert.Equal(t, "TXT", r.URL.Query().Get("type"))
		jsonResponse(w, loadFixture("dmarc_records"))
	})

	er, err := zone.emailRouting()
	require.NoError(t, err)

	dmarc, err := er.dmarcConfigured()
	require.NoError(t, err)
	assert.True(t, dmarc, "fixture has v=DMARC1 record, dmarcConfigured should be true")
}

func TestEmailRouting_dmarcNotConfigured(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/email/routing", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, loadFixture("email_routing_settings"))
	})
	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/dns_records", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, loadFixture("dmarc_records_empty"))
	})

	er, err := zone.emailRouting()
	require.NoError(t, err)

	dmarc, err := er.dmarcConfigured()
	require.NoError(t, err)
	assert.False(t, dmarc, "no _dmarc TXT record present, dmarcConfigured should be false")
}

func TestEmailRouting_dnsRecords(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/email/routing", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, loadFixture("email_routing_settings"))
	})
	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/email/routing/dns", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, loadFixture("email_routing_dns"))
	})

	er, err := zone.emailRouting()
	require.NoError(t, err)

	records, err := er.dnsRecords()
	require.NoError(t, err)
	require.Len(t, records, 4)

	mx0 := records[0].(map[string]any)
	assert.Equal(t, "MX", mx0["type"])
	assert.Equal(t, "isaac.mx.cloudflare.net", mx0["content"])
	assert.Equal(t, int64(11), mx0["priority"])

	txt := records[3].(map[string]any)
	assert.Equal(t, "TXT", txt["type"])
	assert.Contains(t, txt["content"].(string), "v=spf1")
	// SPF/MX records that aren't MX have priority = 0
	assert.Equal(t, int64(0), txt["priority"])
}
