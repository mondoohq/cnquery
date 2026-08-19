// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	madmin "github.com/minio/madmin-go/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebhookTargetsFromRealConfig pins the mapping against the configuration a
// real deployment returned, which carries all four shapes at once: the unnamed
// default target, a target with an authentication token, one without, and one
// explicitly disabled.
func TestWebhookTargetsFromRealConfig(t *testing.T) {
	configs, err := madmin.ParseServerConfigOutput(fixture(t, "config_kv_audit_webhook.txt"))
	require.NoError(t, err)

	status := map[string]string{
		"audit-_":       "offline",
		"audit-primary": "online",
		"audit-noauth":  "offline",
	}
	targets := webhookTargetsFromConfig(webhookTypeAudit, configs, status)
	require.Len(t, targets, 4)

	byName := map[string]webhookTarget{}
	for _, target := range targets {
		byName[target.Name] = target
	}

	t.Run("unnamed default target", func(t *testing.T) {
		target, ok := byName[defaultTargetName]
		require.True(t, ok, "the unnamed target is reported as _")
		assert.Equal(t, "http://127.0.0.1:18099/default", target.Endpoint)
		assert.True(t, target.Enabled)
		assert.Equal(t, "offline", target.Status)
		assert.Equal(t, int64(100000), target.QueueSize)
		assert.Equal(t, "5s", target.HTTPTimeout)
		assert.False(t, target.ClientCertConfigured)
	})

	t.Run("enabled target", func(t *testing.T) {
		// The deployment omits the enable key on a target it is using, so an
		// absent key has to read as enabled. Defaulting it to false would
		// report an active audit trail as switched off.
		target := byName["primary"]
		assert.True(t, target.Enabled)
		assert.Equal(t, "online", target.Status)
		assert.Equal(t, "http://127.0.0.1:18080/audit", target.Endpoint)
	})

	t.Run("disabled target is reported, not hidden", func(t *testing.T) {
		// A configured-but-disabled destination is exactly what an audit wants
		// to see, so it must survive into the result rather than being dropped.
		target := byName["offtarget"]
		assert.False(t, target.Enabled)
		assert.Equal(t, "", target.Status, "a disabled target gets no status")
		assert.Equal(t, "http://127.0.0.1:18098/off", target.Endpoint)
	})

	t.Run("no target carries the authentication token", func(t *testing.T) {
		for _, target := range targets {
			assert.NotContains(t, target.Endpoint, "supersecrettoken")
		}
	})
}

// TestAuthTokenIsWithheldByTheDeployment records the behaviour that rules out an
// "is this target authenticated" field: the key is dropped entirely when a
// token is set, and present-and-empty when it is not. Deriving authentication
// from either spelling gets the answer backwards the moment the behaviour
// changes, so nothing in the schema depends on it.
func TestAuthTokenIsWithheldByTheDeployment(t *testing.T) {
	configs, err := madmin.ParseServerConfigOutput(fixture(t, "config_kv_audit_webhook.txt"))
	require.NoError(t, err)

	byTarget := map[string]map[string]string{}
	for _, cfg := range configs {
		byTarget[cfg.Target] = configKV(cfg)
	}

	_, withToken := byTarget["primary"]["auth_token"]
	assert.False(t, withToken, "the target that has a token reports no auth_token key at all")

	value, withoutToken := byTarget["noauth"]["auth_token"]
	assert.True(t, withoutToken, "the target with no token reports an empty auth_token")
	assert.Equal(t, "", value)
}

func TestWebhookTargetsFromLoggerConfig(t *testing.T) {
	configs, err := madmin.ParseServerConfigOutput(fixture(t, "config_kv_logger_webhook.txt"))
	require.NoError(t, err)

	targets := webhookTargetsFromConfig(webhookTypeLogger, configs,
		map[string]string{"logger-primary": "offline"})
	require.Len(t, targets, 1, "the placeholder row with no endpoint is not a target")

	assert.Equal(t, webhookTypeLogger, targets[0].Type)
	assert.Equal(t, "primary", targets[0].Name)
	assert.Equal(t, "http://127.0.0.1:18081/logs", targets[0].Endpoint)
	assert.True(t, targets[0].Enabled)
	assert.Equal(t, "offline", targets[0].Status)
}

func TestWebhookTargetsEdgeCases(t *testing.T) {
	t.Run("no configuration at all", func(t *testing.T) {
		assert.Empty(t, webhookTargetsFromConfig(webhookTypeAudit, nil, nil))
	})

	t.Run("placeholder row with no endpoint is dropped", func(t *testing.T) {
		configs := []madmin.SubsysConfig{{
			SubSystem: subsysAuditWebhook,
			KV:        []madmin.ConfigKV{{Key: "enable", Value: "off"}, {Key: "endpoint", Value: ""}},
		}}
		assert.Empty(t, webhookTargetsFromConfig(webhookTypeAudit, configs, nil))
	})

	t.Run("a target missing from the status map gets no status, not a guess", func(t *testing.T) {
		configs := []madmin.SubsysConfig{{
			SubSystem: subsysAuditWebhook,
			Target:    "somewhere",
			KV:        []madmin.ConfigKV{{Key: "endpoint", Value: "http://x/y"}},
		}}
		targets := webhookTargetsFromConfig(webhookTypeAudit, configs, map[string]string{})
		require.Len(t, targets, 1)
		assert.Equal(t, "", targets[0].Status)
	})

	t.Run("client certificate is reported when configured", func(t *testing.T) {
		configs := []madmin.SubsysConfig{{
			SubSystem: subsysAuditWebhook,
			Target:    "mtls",
			KV: []madmin.ConfigKV{
				{Key: "endpoint", Value: "https://x/y"},
				{Key: "client_cert", Value: "/etc/minio/client.crt"},
			},
		}}
		targets := webhookTargetsFromConfig(webhookTypeAudit, configs, nil)
		require.Len(t, targets, 1)
		assert.True(t, targets[0].ClientCertConfigured)
	})

	t.Run("enable is matched case-insensitively", func(t *testing.T) {
		configs := []madmin.SubsysConfig{{
			SubSystem: subsysAuditWebhook,
			Target:    "t",
			KV: []madmin.ConfigKV{
				{Key: "endpoint", Value: "http://x/y"},
				{Key: "enable", Value: "OFF"},
			},
		}}
		targets := webhookTargetsFromConfig(webhookTypeAudit, configs, nil)
		require.Len(t, targets, 1)
		assert.False(t, targets[0].Enabled)
	})
}

func TestWebhookStatus(t *testing.T) {
	entries := []madmin.Audit{
		{"audit-primary": madmin.Status{Status: "online"}},
		{"audit-noauth": madmin.Status{Status: "offline"}},
	}
	assert.Equal(t, map[string]string{
		"audit-primary": "online",
		"audit-noauth":  "offline",
	}, webhookStatus(entries))

	assert.Equal(t, map[string]string{}, webhookStatus([]madmin.Logger(nil)))
}

// TestUnquoteConfigValue covers the quoting the deployment applies to a value
// containing spaces. The parser keeps the quotes, so a value read straight out
// of it would carry them into the schema.
func TestUnquoteConfigValue(t *testing.T) {
	cases := []struct{ in, want string }{
		{`plain`, `plain`},
		{`"quoted value"`, `quoted value`},
		{``, ``},
		{`"`, `"`},
		{`""`, ``},
		{`"unbalanced`, `"unbalanced`},
		{`unbalanced"`, `unbalanced"`},
		{`"default-src 'self'; script-src 'self'"`, `default-src 'self'; script-src 'self'`},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, unquoteConfigValue(tc.in), "input %q", tc.in)
	}
}

// TestBrowserConfigIsReadable proves the console settings survive the quoting,
// using the value shape a real deployment returned for the CSP policy.
func TestBrowserConfigIsReadable(t *testing.T) {
	configs, err := madmin.ParseServerConfigOutput(
		`browser csp_policy="default-src 'self' 'unsafe-eval'; script-src 'self' https://unpkg.com;" ` +
			`hsts_seconds=0 hsts_include_subdomains=off hsts_preload=off ` +
			`referrer_policy=strict-origin-when-cross-origin `)
	require.NoError(t, err)
	require.Len(t, configs, 1)

	kv := configKV(configs[0])
	assert.Equal(t, int64(0), parseInt(kv["hsts_seconds"]))
	assert.False(t, parseOnOff(kv["hsts_include_subdomains"]))
	assert.False(t, parseOnOff(kv["hsts_preload"]))
	assert.Equal(t, "default-src 'self' 'unsafe-eval'; script-src 'self' https://unpkg.com;", kv["csp_policy"])
}
