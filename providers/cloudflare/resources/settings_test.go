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

func TestZoneSettings(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/settings", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("zone_settings"))
	})

	s, err := zone.settings()
	require.NoError(t, err)
	require.NotNil(t, s)

	// String settings extracted from the heterogeneous settings array
	assert.Equal(t, "strict", s.Ssl.Data)
	assert.Equal(t, "on", s.AlwaysUseHttps.Data)
	assert.Equal(t, "1.2", s.MinTlsVersion.Data)
	assert.Equal(t, "on", s.Tls13.Data)
	assert.Equal(t, "on", s.AutomaticHttpsRewrites.Data)
	assert.Equal(t, "high", s.SecurityLevel.Data)
	assert.Equal(t, "on", s.Waf.Data)
	assert.Equal(t, "on", s.BrowserCheck.Data)
	assert.Equal(t, "on", s.OpportunisticEncryption.Data)
	assert.Equal(t, "on", s.EmailObfuscation.Data)
	assert.Equal(t, "off", s.HotlinkProtection.Data)
	assert.Equal(t, "on", s.ServerSideExcludes.Data)

	// HSTS sub-fields extracted from the nested security_header.strict_transport_security struct
	assert.True(t, s.HstsEnabled.Data)
	assert.Equal(t, int64(15552000), s.HstsMaxAge.Data)
	assert.True(t, s.HstsIncludeSubdomains.Data)
	assert.True(t, s.HstsPreload.Data)
	assert.True(t, s.HstsNoSniff.Data)
}

func TestBotManagement(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/bot_management", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("bot_management"))
	})

	result, err := zone.botManagement()
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.EnableJs.Data)
	assert.True(t, result.FightMode.Data)
	assert.Equal(t, "block", result.SbfmDefinitelyAutomated.Data)
	assert.Equal(t, "managed_challenge", result.SbfmLikelyAutomated.Data)
	assert.Equal(t, "allow", result.SbfmVerifiedBots.Data)
	assert.False(t, result.SbfmStaticResourceProtection.Data)
	assert.True(t, result.AutoUpdateModel.Data)
	assert.True(t, result.UsingLatestModel.Data)
	assert.Equal(t, "block", result.AiBotsProtection.Data)
}

func TestBotManagement_unavailable(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/bot_management", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		jsonResponse(w, `{"success":false,"errors":[{"code":10000,"message":"Authentication error"}]}`)
	})

	result, err := zone.botManagement()
	require.NoError(t, err)
	assert.Nil(t, result)
}
