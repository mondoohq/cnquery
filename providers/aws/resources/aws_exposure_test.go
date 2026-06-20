// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestViewerPolicyEnforcesHttps(t *testing.T) {
	assert.True(t, viewerPolicyEnforcesHttps("https-only"))
	assert.True(t, viewerPolicyEnforcesHttps("redirect-to-https"))
	assert.False(t, viewerPolicyEnforcesHttps("allow-all"))
	assert.False(t, viewerPolicyEnforcesHttps("Allow-All"))
	assert.False(t, viewerPolicyEnforcesHttps(""))
}

func TestListenerProtocolIsPlaintext(t *testing.T) {
	for _, p := range []string{"HTTP", "TCP", "UDP", "TCP_UDP", "http", "tcp"} {
		assert.True(t, listenerProtocolIsPlaintext(p), p)
	}
	for _, p := range []string{"HTTPS", "TLS", "https", "tls", ""} {
		assert.False(t, listenerProtocolIsPlaintext(p), p)
	}
}
