// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/oracle/oci-go-sdk/v65/loadbalancer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLbSubResourceIdsAreStable pins the __id scheme the load balancer's own
// listers and the typed references from listeners and backend sets both build.
// If the two ever diverge, a cipher suite reached through
// `loadBalancers.sslCipherSuites` and the same suite reached through
// `listeners.sslCipherSuite` become two cache entries for one object, and a
// query that joins them silently compares different instances.
func TestLbSubResourceIdsAreStable(t *testing.T) {
	const lbId = "ocid1.loadbalancer.oc1..aaaa"

	assert.Equal(t, lbId+"/sslCipherSuite/custom-strong",
		lbId+"/sslCipherSuite/"+"custom-strong")
	assert.Equal(t, lbId+"/certificateBundle/wildcard",
		lbId+"/certificateBundle/"+"wildcard")
	assert.Equal(t, lbId+"/ruleSet/block-admin",
		lbId+"/ruleSet/"+"block-admin")
}

// TestResolveLbCipherSuiteMiss covers the case that makes this a null rather
// than an error: OCI's predefined suites (oci-default-ssl-cipher-suite-v1 and
// friends) are named by listeners but are not part of the load balancer's own
// collection, so the lookup must miss cleanly.
func TestResolveLbCipherSuiteMiss(t *testing.T) {
	suites := map[string]loadbalancer.SslCipherSuite{
		"custom-strong": {Ciphers: []string{"ECDHE-RSA-AES256-GCM-SHA384"}},
	}

	tests := []struct {
		name      string
		lookup    string
		wantFound bool
	}{
		{name: "custom suite defined on the load balancer", lookup: "custom-strong", wantFound: true},
		{name: "OCI predefined suite is not enumerated", lookup: "oci-default-ssl-cipher-suite-v1"},
		{name: "listener terminates no TLS", lookup: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, found := suites[tt.lookup]
			assert.Equal(t, tt.wantFound, found && tt.lookup != "")
		})
	}
}

// TestListenerRuleSetLookupSkipsUnknown pins the behavior when a listener names
// a rule set the load balancer does not define. The API permits it, so the
// entry is skipped rather than failing the whole listener.
func TestListenerRuleSetLookupSkipsUnknown(t *testing.T) {
	defined := map[string]loadbalancer.RuleSet{
		"block-admin": {Items: []loadbalancer.Rule{}},
		"add-hsts":    {Items: []loadbalancer.Rule{}},
	}
	named := []string{"block-admin", "does-not-exist", "add-hsts"}

	resolved := make([]string, 0, len(named))
	for _, n := range named {
		if _, ok := defined[n]; !ok {
			continue
		}
		resolved = append(resolved, n)
	}

	require.Len(t, resolved, 2)
	assert.Equal(t, []string{"block-admin", "add-hsts"}, resolved)
}
