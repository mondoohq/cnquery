// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// update regenerates the golden files: go test ./apps/mql/cmd -run Golden -update
var update = flag.Bool("update", false, "update golden files")

// healthyRegisteredStatus returns a fully-populated Status representing a
// client that is registered, authenticated, on the latest version, and talking
// to a healthy Mondoo Platform. Individual tests mutate copies of it to cover
// the conditional branches (not registered, update available, etc.).
func healthyRegisteredStatus() Status {
	s := Status{
		Client: ClientStatus{
			Timestamp:      "2026-06-29T14:37:00-07:00",
			Version:        "13.25.0",
			APIVersion:     "13",
			LatestVersion:  "13.25.0",
			Platform:       &inventory.Platform{Name: "macos", Version: "26.5.1", Arch: "arm64"},
			Hostname:       "workstation.localdomain",
			IP:             "192.0.2.10",
			Registered:     true,
			Mrn:            "//agents.api.mondoo.app/spaces/test/agents/123",
			ServiceAccount: "//agents.api.mondoo.app/spaces/test/serviceaccounts/abc",
			ParentMrn:      "//captain.api.mondoo.app/spaces/test",
			UpdatesURL:     "https://releases.mondoo.com",
			ProvidersURL:   "https://releases.mondoo.com/providers",
			ConfigFile:     "",
			Providers: []ProviderStatus{
				{Name: "aws", Installed: "13.18.0", Latest: "13.30.0", Outdated: true},
				{Name: "core", Installed: "13.25.0", Latest: "13.25.0", Outdated: false},
				{Name: "os", Installed: "13.22.0", Latest: "13.25.2", Outdated: true},
			},
		},
	}
	s.Upstream.API.Endpoint = "https://us.api.mondoo.com"
	s.Upstream.API.Status = "SERVING"
	s.Upstream.API.Timestamp = "2026-06-29T21:37:00Z"
	s.Upstream.API.Version = "13"
	return s
}

func TestRenderCli_Header(t *testing.T) {
	s := healthyRegisteredStatus()

	out := s.RenderCli(RenderOptions{Color: false})

	// The header is the first line: the command name plus a metadata strip with
	// the version, os/arch, and the timestamp.
	firstLine := strings.SplitN(strings.TrimLeft(out, "\n"), "\n", 2)[0]
	assert.Contains(t, firstLine, "mql status")
	assert.Contains(t, firstLine, "13.25.0")
	assert.Contains(t, firstLine, "macos/arm64")
	assert.Contains(t, firstLine, "2026-06-29")
}

func TestRenderCli_NoColorHasNoEscapes(t *testing.T) {
	s := healthyRegisteredStatus()

	out := s.RenderCli(RenderOptions{Color: false})

	assert.NotContains(t, out, "\x1b[", "Color:false output must contain no ANSI escape sequences")
}

func TestRenderCli_HealthSummary_NotRegistered(t *testing.T) {
	s := healthyRegisteredStatus()
	s.Client.Registered = false
	s.Client.Mrn = ""
	s.Client.ServiceAccount = ""

	out := s.RenderCli(RenderOptions{Color: false})

	assert.Contains(t, out, "Client")
	assert.Contains(t, out, "not registered")
	assert.Contains(t, out, "action needed")
}

// registeredAuthFailedStatus mirrors a client whose config is present and
// registered, but whose service account was revoked or expired on the platform:
// the ping fails even though the API itself is reachable and SERVING.
func registeredAuthFailedStatus() Status {
	s := healthyRegisteredStatus()
	s.Client.PingPongError = errors.New("rpc error: code = Unauthenticated desc = invalid service account")
	// isolate the auth-failed path: everything else current
	for i := range s.Client.Providers {
		s.Client.Providers[i].Outdated = false
		s.Client.Providers[i].Latest = s.Client.Providers[i].Installed
	}
	return s
}

func TestRenderCli_HealthSummary_AuthFailed(t *testing.T) {
	s := registeredAuthFailedStatus()

	out := s.RenderCli(RenderOptions{Color: false})

	// still registered, but the ping was rejected
	assert.Contains(t, out, "registered")
	assert.Contains(t, out, "authentication failed")
}

func TestRenderCli_Footer_AuthFailed(t *testing.T) {
	s := registeredAuthFailedStatus()

	out := s.RenderCli(RenderOptions{Color: false})

	// footer must flag the error and give an actionable re-registration step
	// rather than the misleading "not registered" line.
	assert.Contains(t, out, "authentication failed")
	assert.Contains(t, out, "exit 1")
	assert.Contains(t, out, "next steps")
	assert.Contains(t, out, "mql login --token <token>")
	assert.Contains(t, out, "fresh token")
	assert.NotContains(t, out, "not registered")
}

func TestRenderCli_HealthSummary_RegisteredAndHealthy(t *testing.T) {
	s := healthyRegisteredStatus()

	out := s.RenderCli(RenderOptions{Color: false})

	// Platform dot reflects a reachable, SERVING API.
	assert.Contains(t, out, "Platform")
	assert.Contains(t, out, "SERVING")
	assert.Contains(t, out, "healthy")
}

func TestRenderCli_HealthSummary_ProvidersStale(t *testing.T) {
	s := healthyRegisteredStatus() // fixture has 2 outdated providers, mql itself current

	out := s.RenderCli(RenderOptions{Color: false})

	assert.Contains(t, out, "Updates")
	assert.Contains(t, out, "2 providers outdated")
}

func TestRenderCli_HealthSummary_FullyCurrent(t *testing.T) {
	s := healthyRegisteredStatus()
	// mark every provider current
	for i := range s.Client.Providers {
		s.Client.Providers[i].Outdated = false
		s.Client.Providers[i].Latest = s.Client.Providers[i].Installed
	}

	out := s.RenderCli(RenderOptions{Color: false})

	assert.Contains(t, out, "up to date")
}

func TestRenderCli_SystemSection(t *testing.T) {
	s := healthyRegisteredStatus()

	out := s.RenderCli(RenderOptions{Color: false})

	assert.Contains(t, out, "System")
	assert.Contains(t, out, "workstation.localdomain")
	assert.Contains(t, out, "192.0.2.10")
	assert.Contains(t, out, "macos")
	assert.Contains(t, out, "26.5.1")
	assert.Contains(t, out, "arm64")
}

func TestRenderCli_MqlSection_DefaultsAndCurrent(t *testing.T) {
	s := healthyRegisteredStatus() // version == latest, no config file

	out := s.RenderCli(RenderOptions{Color: false})

	assert.Contains(t, out, "13.25.0")
	// API version surfaced
	assert.Contains(t, out, "API")
	// no config file loaded => communicates defaults
	assert.Contains(t, out, "defaults")
	assert.Contains(t, out, "https://releases.mondoo.com")
}

func TestRenderCli_MqlSection_ConfigFileShown(t *testing.T) {
	s := healthyRegisteredStatus()
	s.Client.ConfigFile = "/home/user/.config/mondoo/mondoo.yml"

	out := s.RenderCli(RenderOptions{Color: false})

	assert.Contains(t, out, "/home/user/.config/mondoo/mondoo.yml")
	assert.NotContains(t, out, "defaults — no config file")
}

func TestRenderCli_MqlSection_UpdateAvailableShowsArrow(t *testing.T) {
	s := healthyRegisteredStatus()
	s.Client.Version = "13.22.0"
	s.Client.LatestVersion = "13.25.0"

	out := s.RenderCli(RenderOptions{Color: false})

	assert.Contains(t, out, "13.22.0")
	assert.Contains(t, out, "13.25.0")
	assert.Contains(t, out, "→")
}

func TestRenderCli_PlatformSection(t *testing.T) {
	s := healthyRegisteredStatus()

	out := s.RenderCli(RenderOptions{Color: false})

	assert.Contains(t, out, "Mondoo Platform")
	assert.Contains(t, out, "https://us.api.mondoo.com")
	assert.Contains(t, out, "SERVING")
	assert.Contains(t, out, "2026-06-29T21:37:00Z")
}

func TestRenderCli_PlatformSection_NotRegisteredShowsLoginHint(t *testing.T) {
	s := healthyRegisteredStatus()
	s.Client.Registered = false
	s.Client.Mrn = ""

	out := s.RenderCli(RenderOptions{Color: false})

	assert.Contains(t, out, "mql login")
}

func TestRenderCli_ProvidersSection_Counts(t *testing.T) {
	s := healthyRegisteredStatus() // 3 installed, 2 outdated, 1 current

	out := s.RenderCli(RenderOptions{Color: false})

	assert.Contains(t, out, "Providers")
	assert.Contains(t, out, "3 installed")
	assert.Contains(t, out, "2 outdated")
	assert.Contains(t, out, "1 current")
}

func TestRenderCli_ProvidersSection_DriftRows(t *testing.T) {
	s := healthyRegisteredStatus()

	out := s.RenderCli(RenderOptions{Color: false})

	assert.Contains(t, out, "aws")
	assert.Contains(t, out, "13.18.0")
	assert.Contains(t, out, "13.30.0")
	assert.Contains(t, out, "os")
	assert.Contains(t, out, "13.25.2")
}

func TestRenderCli_ProvidersSection_RegistryUnreachable(t *testing.T) {
	s := healthyRegisteredStatus()
	for i := range s.Client.Providers {
		s.Client.Providers[i].Latest = ""
		s.Client.Providers[i].Outdated = false
	}

	out := s.RenderCli(RenderOptions{Color: false})

	assert.Contains(t, out, "3 installed")
	assert.Contains(t, out, "registry unreachable")
}

func TestRenderCli_Footer_NotRegistered(t *testing.T) {
	s := healthyRegisteredStatus()
	s.Client.Registered = false
	s.Client.Mrn = ""

	out := s.RenderCli(RenderOptions{Color: false})

	assert.Contains(t, out, "exit 1")
	assert.Contains(t, out, "next steps")
	assert.Contains(t, out, "mql login")
}

func TestRenderCli_Footer_AllHealthyHasNoExitBanner(t *testing.T) {
	s := healthyRegisteredStatus()
	for i := range s.Client.Providers {
		s.Client.Providers[i].Outdated = false
		s.Client.Providers[i].Latest = s.Client.Providers[i].Installed
	}

	out := s.RenderCli(RenderOptions{Color: false})

	assert.NotContains(t, out, "exit 1")
	assert.Contains(t, out, "healthy")
}

// staleNotRegisteredStatus mirrors a fresh, unregistered install that is behind
// on both mql and its providers — the worst-case footer with every next step.
func staleNotRegisteredStatus() Status {
	s := healthyRegisteredStatus()
	s.Client.Registered = false
	s.Client.Mrn = ""
	s.Client.ServiceAccount = ""
	s.Client.ParentMrn = ""
	s.Client.Version = "13.22.0"
	s.Client.LatestVersion = "13.25.0"
	return s
}

func TestRenderCli_Golden(t *testing.T) {
	cases := []struct {
		name   string
		status Status
	}{
		{"healthy_registered", healthyRegisteredStatus()},
		{"stale_not_registered", staleNotRegisteredStatus()},
		{"registered_auth_failed", registeredAuthFailedStatus()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.status.RenderCli(RenderOptions{Color: false})
			golden := filepath.Join("testdata", "status_"+tc.name+".golden")

			if *update {
				require.NoError(t, os.MkdirAll("testdata", 0o755))
				require.NoError(t, os.WriteFile(golden, []byte(got), 0o644))
			}

			want, err := os.ReadFile(golden)
			require.NoError(t, err, "missing golden file; run: go test ./apps/mql/cmd -run Golden -update")
			assert.Equal(t, string(want), got)
		})
	}
}

func TestPadRight_CountsRunesNotBytes(t *testing.T) {
	// "café" is 5 bytes but 4 runes; padding to width 6 must add 2 spaces
	// (rune-based), not 1 (byte-based).
	assert.Equal(t, "café  ", padRight("café", 6))
}

func TestRenderCli_PlatformSection_WarningsAndFeatures(t *testing.T) {
	s := healthyRegisteredStatus()
	s.Upstream.Features = []string{"alpha", "beta"}
	s.Upstream.Warnings = []string{"possible clock skew detected: 45s"}

	out := s.RenderCli(RenderOptions{Color: false})

	assert.Contains(t, out, "alpha, beta")
	assert.Contains(t, out, "possible clock skew detected: 45s")
	assert.Contains(t, out, "⚠")
}
