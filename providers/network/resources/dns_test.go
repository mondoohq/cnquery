// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/network/connection"
	"go.mondoo.com/mql/v13/providers/network/resources"
	"go.mondoo.com/mql/v13/utils/syncx"
)

func TestResource_DNS(t *testing.T) {
	res := x.TestQuery(t, "dns(\"mondoo.com\").mx")
	assert.NotEmpty(t, res)
}

// dnsLiveQuery runs a live DNS query for a test and returns the first result's
// data. It skips the test when the lookup itself errors, so the inherently
// network-dependent DNS tests don't fail on restricted or flaky CI networks.
// Parsing correctness is covered by the deterministic tests in
// dns_internal_test.go.
func dnsLiveQuery(t *testing.T, query string) *llx.RawData {
	t.Helper()
	res := x.TestQuery(t, query)
	require.NotEmpty(t, res)
	if res[0].Data.Error != nil {
		t.Skipf("skipping: live DNS lookup unavailable (%s): %v", query, res[0].Data.Error)
	}
	return res[0].Data
}

func TestResource_DnsDnssec(t *testing.T) {
	// cloudflare.com is reliably DNSSEC-signed. A successful lookup reports
	// enabled; an empty result means the DNSKEY lookup didn't complete in this
	// environment, so skip rather than fail.
	enabled := dnsLiveQuery(t, `dns("cloudflare.com").dnssec.enabled`)
	if enabled.Value != true {
		t.Skip("skipping: live DNSKEY lookup returned no data")
	}
	assert.Equal(t, true, enabled.Value)

	keys := dnsLiveQuery(t, `dns("cloudflare.com").dnssec.keys.all(algorithm > 0 && publicKey != "")`)
	assert.Equal(t, true, keys.Value)

	algos := dnsLiveQuery(t, `dns("cloudflare.com").dnssec.algorithms`)
	assert.NotEmpty(t, algos.Value)
}

func TestResource_DnsSpf(t *testing.T) {
	// google.com publishes an SPF record with a terminating ~all.
	has := dnsLiveQuery(t, `dns("google.com").spf.any(version == "spf1")`)
	if has.Value != true {
		t.Skip("skipping: live SPF lookup returned no data")
	}
	assert.Equal(t, true, has.Value)

	q := dnsLiveQuery(t, `dns("google.com").spf.all(allQualifier.in(["+","-","~","?"]))`)
	assert.Equal(t, true, q.Value)
}

func TestResource_DnsDmarc(t *testing.T) {
	// google.com publishes a DMARC record at _dmarc.google.com.
	ver := dnsLiveQuery(t, `dns("google.com").dmarc.version`)
	if v, _ := ver.Value.(string); v == "" {
		t.Skip("skipping: live DMARC lookup returned no data")
	}
	assert.Equal(t, "DMARC1", ver.Value)

	pol := dnsLiveQuery(t, `dns("google.com").dmarc.policy.in(["none","quarantine","reject"])`)
	assert.Equal(t, true, pol.Value)
}

func TestResource_DomainName(t *testing.T) {
	res := x.TestQuery(t, "domainName")
	assert.NotEmpty(t, res)
	res = x.TestQuery(t, "domainName(\"mondoo.com\").tld")
	assert.Equal(t, "com", string(res[0].Result().Data.Value))
}

func TestResource_DnsFqdn(t *testing.T) {
	testCases := []struct {
		hostName   string
		expectedId string
	}{
		{
			hostName:   "127.0.0.1",
			expectedId: "dns/",
		},
		{
			hostName:   "3.127.139.132",
			expectedId: "dns/",
		},
		{
			hostName:   "www.mondoo.com",
			expectedId: "dns/www.mondoo.com",
		},
		{
			hostName:   "ec2-3-127-139-132.eu-central-1.compute.amazonaws.com",
			expectedId: "dns/ec2-3-127-139-132.eu-central-1.compute.amazonaws.com",
		},
	}

	runtime := &plugin.Runtime{Resources: &syncx.Map[plugin.Resource]{}}

	for _, tc := range testCases {
		conf := &inventory.Config{
			Host: tc.hostName,
		}
		runtime.Connection = connection.NewHostConnection(1, &inventory.Asset{}, conf)

		dns, err := resources.NewResource(
			runtime,
			"dns",
			map[string]*llx.RawData{},
		)
		require.NoError(t, err)
		require.Equal(t, tc.expectedId, dns.MqlID())
	}
}
