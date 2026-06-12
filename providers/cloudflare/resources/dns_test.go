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

func TestDNSRecords(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/dns_records", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		if page := r.URL.Query().Get("page"); page != "" && page != "1" {
			jsonResponse(w, `{"result":[],"success":true,"errors":[],"messages":[]}`)
			return
		}
		jsonResponse(w, loadFixture("dns_records"))
	})

	dns, err := zone.dns()
	require.NoError(t, err)
	require.NotNil(t, dns)

	records, err := dns.records()
	require.NoError(t, err)
	require.Len(t, records, 4)

	a := records[0].(*mqlCloudflareDnsRecord)
	assert.Equal(t, "rec-a-001", a.Id.Data)
	assert.Equal(t, "A", a.Type.Data)
	assert.Equal(t, "example.com", a.Name.Data)
	assert.Equal(t, "192.0.2.10", a.Content.Data)
	assert.True(t, a.Proxied.Data)
	assert.True(t, a.Proxiable.Data)
	assert.Equal(t, "primary apex A", a.Comment.Data)
	assert.Equal(t, []any{"env:prod"}, a.Tags.Data)
	assert.Equal(t, int64(0), a.Priority.Data, "non-MX records should default to priority 0")

	mx := records[2].(*mqlCloudflareDnsRecord)
	assert.Equal(t, "MX", mx.Type.Data)
	assert.Equal(t, int64(10), mx.Priority.Data, "MX record should preserve its priority")
	assert.False(t, mx.Proxied.Data)

	cname := records[3].(*mqlCloudflareDnsRecord)
	assert.Equal(t, "CNAME", cname.Type.Data)
	assert.Equal(t, "www.example.com", cname.Name.Data)
}
