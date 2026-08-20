// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/testutils"
	"go.mondoo.com/mql/providers/os/resources/firefox"
)

// These tests cover what only the resource layer can show: that a field
// resolves to an explicit null rather than staying unset, that an error
// reaches the caller naming the file it came from, and that a policy set is
// traversable by key from MQL. What a policy document parses to, which paths
// are probed, and how two sources merge are settled in the firefox package,
// against the same shapes but without a runtime in the way.

// The recording each test runs against is built here rather than checked in.
// A stored recording of an unmanaged host is a copy of the candidate list, and
// a copy goes stale silently: add a probe path to the resource and the file
// still describes the old host, so the new path is never exercised. Building
// it from PolicyFileCandidates means the fixture cannot drift from the code.
func firefoxLinuxHost(t *testing.T, files map[string]string) firefoxHost {
	t.Helper()

	type recordedResource struct {
		Resource string
		ID       string
		Fields   map[string]*llx.RawData
	}

	resources := []recordedResource{}
	for _, path := range firefox.PolicyFileCandidates("linux") {
		fields := map[string]*llx.RawData{
			"path":   llx.StringData(path),
			"exists": llx.BoolData(false),
		}
		if content, ok := files[path]; ok {
			fields["exists"] = llx.BoolData(true)
			fields["content"] = llx.StringData(content)
		}
		resources = append(resources, recordedResource{
			Resource: "file",
			ID:       path,
			Fields:   fields,
		})
	}

	recording := struct {
		Assets []struct {
			Asset       *inventory.Asset   `json:"asset"`
			Connections []any              `json:"connections"`
			Resources   []recordedResource `json:"resources"`
		} `json:"assets"`
	}{
		Assets: []struct {
			Asset       *inventory.Asset   `json:"asset"`
			Connections []any              `json:"connections"`
			Resources   []recordedResource `json:"resources"`
		}{{
			Asset: &inventory.Asset{
				Id:          "firefox-policies-linux",
				PlatformIds: []string{"//platformid.api.mondoo.app/test/firefox-policies-linux"},
				Name:        "firefox-policies-linux",
				Platform: &inventory.Platform{
					Name:    "debian",
					Arch:    "aarch64",
					Title:   "Debian GNU/Linux",
					Family:  []string{"debian", "linux", "unix", "os"},
					Version: "12",
				},
			},
			Connections: []any{map[string]any{
				"url":       "local://",
				"provider":  "os",
				"connector": "local",
				"version":   "",
			}},
			Resources: resources,
		}},
	}

	raw, err := json.Marshal(recording)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "recording.json")
	require.NoError(t, os.WriteFile(path, raw, 0o600))

	tester := testutils.InitTester(testutils.RecordingMock(path))
	return firefoxHost{
		query: func(t *testing.T, query string) []*llx.RawResult {
			t.Helper()
			res := tester.TestQuery(t, query)
			require.NotEmpty(t, res)
			return res
		},
	}
}

type firefoxHost struct {
	query func(t *testing.T, query string) []*llx.RawResult
}

// value returns the result of a single-value query.
func (h firefoxHost) value(t *testing.T, query string) *llx.RawResult {
	t.Helper()
	res := h.query(t, query)
	return res[0]
}

// outcome returns what a comparison evaluated to, which is what a policy check
// reports, rather than the value being compared.
func (h firefoxHost) outcome(t *testing.T, query string) *llx.RawResult {
	t.Helper()
	res := h.query(t, query)
	return res[len(res)-1]
}

const (
	// A managed host, in the shape Debian's firefox-esr package installs.
	esrPolicy = `{
  "policies": {
    "SSLVersionMin": "tls1.2",
    "SanitizeOnShutdown": { "Cache": true, "Cookies": false, "Locked": true },
    "Preferences": {
      "security.default_personal_cert": { "Value": "Ask Every Time", "Status": "locked" }
    }
  }
}`
	esrPolicyPath = "/usr/lib/firefox-esr/distribution/policies.json"

	systemPolicyPath = "/etc/firefox/policies/policies.json"
)

// An unmanaged Firefox has no policy file anywhere, and that is the normal
// state of the overwhelming majority of hosts. Every field has to resolve to
// an explicit answer rather than being left unset, because an unresolved field
// is what makes the runtime either re-fetch forever or panic on a missing
// value.
func TestResource_FirefoxPoliciesUnmanaged(t *testing.T) {
	x := firefoxLinuxHost(t, nil)

	t.Run("configured is false, not an error", func(t *testing.T) {
		res := x.value(t, "firefox.policies.configured")
		assert.Empty(t, res.Result().Error)
		assert.Equal(t, false, res.Data.Value)
	})

	t.Run("source names nothing", func(t *testing.T) {
		res := x.value(t, "firefox.policies.source")
		assert.Empty(t, res.Result().Error)
		assert.Equal(t, "", res.Data.Value)
	})

	t.Run("params resolves to null", func(t *testing.T) {
		res := x.value(t, "firefox.policies.params")
		assert.Empty(t, res.Result().Error)
		assert.Nil(t, res.Data.Value)
	})

	t.Run("file resolves to null", func(t *testing.T) {
		res := x.value(t, "firefox.policies.file")
		assert.Empty(t, res.Result().Error)
		assert.Nil(t, res.Data.Value)
	})

	t.Run("inputs is an empty list", func(t *testing.T) {
		res := x.value(t, "firefox.policies.inputs.length")
		assert.Empty(t, res.Result().Error)
		assert.Equal(t, int64(0), res.Data.Value)
	})

	// The direction that actually matters. A check that reads an absent policy
	// as satisfied is worse than no check at all, because it reports a host as
	// hardened precisely when it is not.
	t.Run("a check against an absent policy is false, never vacuously true", func(t *testing.T) {
		res := x.outcome(t, `firefox.policies.params["SSLVersionMin"] == "tls1.2"`)
		assert.NotEqual(t, true, res.Data.Value, "an unconfigured host must not satisfy a policy check")
	})

	t.Run("reading a key out of a null policy set does not panic", func(t *testing.T) {
		res := x.value(t, `firefox.policies.params["SanitizeOnShutdown"]["Cookies"]`)
		assert.Nil(t, res.Data.Value)
	})
}

// A managed host, reached through the probe order rather than through a path
// the test names. Debian and Ubuntu ship Firefox as firefox-esr, so the ESR
// install prefix has to be one of the paths that probe finds.
func TestResource_FirefoxPoliciesManaged(t *testing.T) {
	x := firefoxLinuxHost(t, map[string]string{esrPolicyPath: esrPolicy})

	t.Run("the file is found and reported", func(t *testing.T) {
		res := x.value(t, "firefox.policies.file.path")
		assert.Empty(t, res.Result().Error)
		assert.Equal(t, esrPolicyPath, res.Data.Value)
	})

	t.Run("configured is true and the source is the file", func(t *testing.T) {
		assert.Equal(t, true, x.value(t, "firefox.policies.configured").Data.Value)
		assert.Equal(t, "file", x.value(t, "firefox.policies.source").Data.Value)
	})

	t.Run("a check against a configured policy passes", func(t *testing.T) {
		res := x.outcome(t, `firefox.policies.params["SSLVersionMin"] == "tls1.2"`)
		assert.Empty(t, res.Result().Error)
		assert.Equal(t, true, res.Data.Value)
	})

	t.Run("nested policies are reachable by key", func(t *testing.T) {
		res := x.value(t, `firefox.policies.params["SanitizeOnShutdown"]["Locked"]`)
		assert.Equal(t, true, res.Data.Value)
	})

	t.Run("preferences keyed by preference name are reachable", func(t *testing.T) {
		res := x.value(t,
			`firefox.policies.params["Preferences"]["security.default_personal_cert"]["Status"]`)
		assert.Equal(t, "locked", res.Data.Value)
	})

	t.Run("inputs lists the one file that contributed", func(t *testing.T) {
		assert.Equal(t, int64(1), x.value(t, "firefox.policies.inputs.length").Data.Value)
		assert.Equal(t, "file", x.value(t, "firefox.policies.inputs[0].source").Data.Value)
	})
}

// /etc wins outright over the install prefix, and nothing is merged between
// them: the losing file's keys are absent, not overridden.
func TestResource_FirefoxPoliciesFirstMatchWins(t *testing.T) {
	x := firefoxLinuxHost(t, map[string]string{
		systemPolicyPath: `{"policies":{"AdminOwned":true,"SSLVersionMin":"tls1.3"}}`,
		esrPolicyPath:    esrPolicy,
	})

	assert.Equal(t, systemPolicyPath, x.value(t, "firefox.policies.file.path").Data.Value)
	assert.Equal(t, "tls1.3", x.value(t, `firefox.policies.params["SSLVersionMin"]`).Data.Value)
	assert.Nil(t, x.value(t, `firefox.policies.params["Preferences"]`).Data.Value,
		"a key of the losing file must not appear in the result")
}

// A policy file that exists but declares nothing is not a managed host, and
// must not be reported as one. The file is still reported, so a permission or
// ownership check can compose onto it.
func TestResource_FirefoxPoliciesEmptyFile(t *testing.T) {
	x := firefoxLinuxHost(t, map[string]string{systemPolicyPath: ""})

	t.Run("an empty file declares no configuration", func(t *testing.T) {
		res := x.value(t, "firefox.policies.configured")
		assert.Empty(t, res.Result().Error)
		assert.Equal(t, false, res.Data.Value)
	})

	t.Run("params resolves to null rather than an empty dict", func(t *testing.T) {
		res := x.value(t, "firefox.policies.params")
		assert.Empty(t, res.Result().Error)
		assert.Nil(t, res.Data.Value)
	})

	t.Run("the file that was found is still reported", func(t *testing.T) {
		assert.Equal(t, systemPolicyPath, x.value(t, "firefox.policies.file.path").Data.Value)
	})
}

// A deployed-but-broken policy file is a misconfiguration worth surfacing.
// Reporting it as "no policy deployed" would be a false all-clear on a host an
// administrator believes is locked down, so it has to be distinguishable from
// absent — and the error has to name the file, or "the JSON is broken" is not
// actionable on a host that could carry one in any of several locations.
func TestResource_FirefoxPoliciesMalformedFile(t *testing.T) {
	x := firefoxLinuxHost(t, map[string]string{systemPolicyPath: `{"policies": {`})

	res := x.value(t, "firefox.policies.configured")
	require.NotEmpty(t, res.Result().Error, "malformed JSON must surface as an error, not as an unmanaged host")
	assert.Contains(t, res.Result().Error, "failed to parse Firefox policy file")
	assert.Contains(t, res.Result().Error, systemPolicyPath)
}
