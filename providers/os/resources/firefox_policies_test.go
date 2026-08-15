// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/testutils"
)

func firefoxPolicyQuery(t *testing.T, recording, query string) *llx.RawResult {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("testdata", recording))
	require.NoError(t, err)
	tester := testutils.InitTester(testutils.RecordingMock(abs))
	res := tester.TestQuery(t, query)
	require.NotEmpty(t, res)
	return res[0]
}

// firefoxPolicyCheck runs a comparison and returns its outcome rather than the
// value being compared, which is what a policy check actually evaluates to.
func firefoxPolicyCheck(t *testing.T, recording, query string) *llx.RawResult {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("testdata", recording))
	require.NoError(t, err)
	tester := testutils.InitTester(testutils.RecordingMock(abs))
	res := tester.TestQuery(t, query)
	require.NotEmpty(t, res)
	return res[len(res)-1]
}

// An unmanaged Firefox has no policy file anywhere, and that is the normal
// state of the overwhelming majority of hosts — not an exceptional one. Every
// field has to resolve to an explicit answer rather than being left unset,
// because an unresolved field is what makes the runtime either re-fetch
// forever or panic on a missing value.
func TestResource_FirefoxPoliciesUnmanaged(t *testing.T) {
	const recording = "firefox_policies_unmanaged_linux.json"

	t.Run("configured is false, not an error", func(t *testing.T) {
		res := firefoxPolicyQuery(t, recording, "firefox.policies.configured")
		assert.Empty(t, res.Result().Error)
		assert.Equal(t, false, res.Data.Value)
	})

	t.Run("source names nothing", func(t *testing.T) {
		res := firefoxPolicyQuery(t, recording, "firefox.policies.source")
		assert.Empty(t, res.Result().Error)
		assert.Equal(t, "", res.Data.Value)
	})

	t.Run("params resolves to null", func(t *testing.T) {
		res := firefoxPolicyQuery(t, recording, "firefox.policies.params")
		assert.Empty(t, res.Result().Error)
		assert.Nil(t, res.Data.Value)
	})

	t.Run("file resolves to null", func(t *testing.T) {
		res := firefoxPolicyQuery(t, recording, "firefox.policies.file")
		assert.Empty(t, res.Result().Error)
		assert.Nil(t, res.Data.Value)
	})

	t.Run("inputs is an empty list", func(t *testing.T) {
		res := firefoxPolicyQuery(t, recording, "firefox.policies.inputs.length")
		assert.Empty(t, res.Result().Error)
		assert.Equal(t, int64(0), res.Data.Value)
	})

	// The direction that actually matters. A check that reads an absent policy
	// as satisfied is worse than no check at all, because it reports a host as
	// hardened precisely when it is not.
	t.Run("a check against an absent policy is false, never vacuously true", func(t *testing.T) {
		res := firefoxPolicyCheck(t, recording, `firefox.policies.params["SSLVersionMin"] == "tls1.2"`)
		assert.NotEqual(t, true, res.Data.Value, "an unconfigured host must not satisfy a policy check")
	})

	t.Run("reading a key out of a null policy set does not panic", func(t *testing.T) {
		res := firefoxPolicyQuery(t, recording, `firefox.policies.params["SanitizeOnShutdown"]["Cookies"]`)
		assert.Nil(t, res.Data.Value)
	})
}

// Debian and Ubuntu ship Firefox as firefox-esr, installed to
// /usr/lib/firefox-esr. A resource that misses that prefix reads a correctly
// managed host as unconfigured — and unconfigured looks exactly like "the
// policy was never applied", so every check fails with nothing to explain why.
func TestResource_FirefoxPoliciesESR(t *testing.T) {
	const recording = "firefox_policies_esr_linux.json"

	t.Run("the ESR install prefix is found", func(t *testing.T) {
		res := firefoxPolicyQuery(t, recording, "firefox.policies.file.path")
		assert.Empty(t, res.Result().Error)
		assert.Equal(t, "/usr/lib/firefox-esr/distribution/policies.json", res.Data.Value)
	})

	t.Run("configured is true and the source is the file", func(t *testing.T) {
		res := firefoxPolicyQuery(t, recording, "firefox.policies.configured")
		assert.Equal(t, true, res.Data.Value)

		res = firefoxPolicyQuery(t, recording, "firefox.policies.source")
		assert.Equal(t, "file", res.Data.Value)
	})

	t.Run("a check against a configured policy passes", func(t *testing.T) {
		res := firefoxPolicyCheck(t, recording, `firefox.policies.params["SSLVersionMin"] == "tls1.2"`)
		assert.Empty(t, res.Result().Error)
		assert.Equal(t, true, res.Data.Value)
	})

	t.Run("nested policies are reachable by key", func(t *testing.T) {
		res := firefoxPolicyQuery(t, recording, `firefox.policies.params["SanitizeOnShutdown"]["Locked"]`)
		assert.Equal(t, true, res.Data.Value)
	})

	t.Run("preferences keyed by preference name are reachable", func(t *testing.T) {
		res := firefoxPolicyQuery(t, recording,
			`firefox.policies.params["Preferences"]["security.default_personal_cert"]["Status"]`)
		assert.Equal(t, "locked", res.Data.Value)
	})

	t.Run("inputs lists the one file that contributed", func(t *testing.T) {
		res := firefoxPolicyQuery(t, recording, "firefox.policies.inputs.length")
		assert.Equal(t, int64(1), res.Data.Value)

		res = firefoxPolicyQuery(t, recording, "firefox.policies.inputs[0].source")
		assert.Equal(t, "file", res.Data.Value)
	})
}

// A policy file that exists but declares nothing is not the same as a managed
// host, and must not be reported as one.
func TestResource_FirefoxPoliciesEmptyFile(t *testing.T) {
	const recording = "firefox_policies_empty_linux.json"

	t.Run("an empty file declares no configuration", func(t *testing.T) {
		res := firefoxPolicyQuery(t, recording, "firefox.policies.configured")
		assert.Empty(t, res.Result().Error)
		assert.Equal(t, false, res.Data.Value)
	})

	t.Run("params resolves to null rather than an empty dict", func(t *testing.T) {
		res := firefoxPolicyQuery(t, recording, "firefox.policies.params")
		assert.Empty(t, res.Result().Error)
		assert.Nil(t, res.Data.Value)
	})

	// The file is still reported even though it contributed nothing, so a
	// permission or ownership check can compose onto it.
	t.Run("the file that was found is still reported", func(t *testing.T) {
		res := firefoxPolicyQuery(t, recording, "firefox.policies.file.path")
		assert.Equal(t, "/etc/firefox/policies/policies.json", res.Data.Value)
	})
}

// A deployed-but-broken policy file is a misconfiguration worth surfacing.
// Reporting it as "no policy deployed" would be a false all-clear on a host an
// administrator believes is locked down, so it has to be distinguishable from
// absent.
func TestResource_FirefoxPoliciesMalformedFile(t *testing.T) {
	res := firefoxPolicyQuery(t, "firefox_policies_malformed_linux.json", "firefox.policies.configured")
	require.NotEmpty(t, res.Result().Error, "malformed JSON must surface as an error, not as an unmanaged host")
	assert.Contains(t, res.Result().Error, "failed to parse Firefox policy file")
	// The error has to name the file, or "the JSON is broken" is not actionable
	// on a host that could carry a policy file in any of several locations.
	assert.Contains(t, res.Result().Error, "/etc/firefox/policies/policies.json")
}
