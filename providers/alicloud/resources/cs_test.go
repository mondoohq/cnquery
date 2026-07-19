// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseCsMasterURL covers the ACK MasterUrl parser, which drives the
// apiServerInternetExposed security signal. A parsing bug would misreport
// whether a cluster's API server is reachable from the internet.
func TestParseCsMasterURL(t *testing.T) {
	t.Run("public and intranet endpoints present", func(t *testing.T) {
		public, intranet := parseCsMasterURL(strp(`{"api_server_endpoint":"https://1.2.3.4:6443","intranet_api_server_endpoint":"https://10.0.0.1:6443"}`))
		assert.Equal(t, "https://1.2.3.4:6443", public)
		assert.Equal(t, "https://10.0.0.1:6443", intranet)
		assert.True(t, public != "", "public endpoint present => internet exposed")
	})
	t.Run("private-only cluster has empty public endpoint", func(t *testing.T) {
		public, intranet := parseCsMasterURL(strp(`{"api_server_endpoint":"","intranet_api_server_endpoint":"https://10.0.0.1:6443"}`))
		assert.Equal(t, "", public)
		assert.Equal(t, "https://10.0.0.1:6443", intranet)
		assert.False(t, public != "", "no public endpoint => not internet exposed")
	})
	t.Run("nil pointer yields empties", func(t *testing.T) {
		public, intranet := parseCsMasterURL(nil)
		assert.Equal(t, "", public)
		assert.Equal(t, "", intranet)
	})
	t.Run("empty string (initializing cluster) yields empties", func(t *testing.T) {
		public, intranet := parseCsMasterURL(strp(""))
		assert.Equal(t, "", public)
		assert.Equal(t, "", intranet)
	})
	t.Run("malformed JSON yields empties, not a panic", func(t *testing.T) {
		public, intranet := parseCsMasterURL(strp("not-json"))
		assert.Equal(t, "", public)
		assert.Equal(t, "", intranet)
	})
}

// TestCsStrList covers the vSwitch/security-group slice flattening used to build
// the typed cross-references, ensuring empties are dropped.
func TestCsStrList(t *testing.T) {
	assert.Equal(t, []string{}, csStrList(nil))
	assert.Equal(t, []string{"vsw-a", "vsw-b"}, csStrList([]*string{strp("vsw-a"), strp(""), strp("vsw-b")}))
}
