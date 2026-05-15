// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

func TestParseWinRegistryHotpatch(t *testing.T) {

	t.Run("parse hptpatching settings correctly", func(t *testing.T) {
		data := `{
			"Name":  "Hotpatch Enrollment Package",
			"HotPatchTableSize": "4096",
			"EnableVirtualizationBasedSecurity": "1"
		}`

		m, err := ParseWinRegistryHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.True(t, m)
	})

	t.Run("parse missing table size", func(t *testing.T) {
		data := `{
			"Name":  "Hotpatch Enrollment Package",
			"HotPatchTableSize": "0",
			"EnableVirtualizationBasedSecurity": "1"
		}`

		m, err := ParseWinRegistryHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.False(t, m)
	})

	t.Run("parse missing name", func(t *testing.T) {
		data := `{
			"Name":  "",
			"HotPatchTableSize": "4096",
			"EnableVirtualizationBasedSecurity": "1"
		}`

		m, err := ParseWinRegistryHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.False(t, m)
	})

	t.Run("parse missing VBS", func(t *testing.T) {
		data := `{
			"Name":  "Hotpatch Enrollment Package",
			"HotPatchTableSize": "1",
			"EnableVirtualizationBasedSecurity": "0"
		}`

		m, err := ParseWinRegistryHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.False(t, m)
	})

	t.Run("parse empty JSON", func(t *testing.T) {
		data := `{
			"Name":  "",
			"HotPatchTableSize": "0",
			"EnableVirtualizationBasedSecurity": "0"
		}`

		m, err := ParseWinRegistryHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.False(t, m)
	})
}

func TestParseWinRegistryClientHotpatch(t *testing.T) {
	t.Run("client hotpatch enabled", func(t *testing.T) {
		data := `{
			"AllowRebootlessUpdates": "1",
			"EnableVirtualizationBasedSecurity": "1"
		}`

		m, err := ParseWinRegistryClientHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.True(t, m)
	})

	t.Run("client no rebootless updates", func(t *testing.T) {
		data := `{
			"AllowRebootlessUpdates": "0",
			"EnableVirtualizationBasedSecurity": "1"
		}`

		m, err := ParseWinRegistryClientHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.False(t, m)
	})

	t.Run("client no VBS", func(t *testing.T) {
		data := `{
			"AllowRebootlessUpdates": "1",
			"EnableVirtualizationBasedSecurity": "0"
		}`

		m, err := ParseWinRegistryClientHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.False(t, m)
	})

	t.Run("client empty values", func(t *testing.T) {
		data := `{
			"AllowRebootlessUpdates": "",
			"EnableVirtualizationBasedSecurity": ""
		}`

		m, err := ParseWinRegistryClientHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.False(t, m)
	})
}

func TestHotpatchSupported(t *testing.T) {
	t.Run("client amd64 build 26100 with sufficient UBR", func(t *testing.T) {
		pf := &inventory.Platform{
			Title:   "Windows 11 Enterprise",
			Version: "26100",
			Build:   "5000",
			Arch:    "AMD64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.True(t, hotpatchSupported(pf))
	})

	t.Run("client amd64 build 26100 with exact minimum UBR", func(t *testing.T) {
		pf := &inventory.Platform{
			Title:   "Windows 11 Enterprise",
			Version: "26100",
			Build:   "2033",
			Arch:    "AMD64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.True(t, hotpatchSupported(pf))
	})

	t.Run("client amd64 build 26100 with UBR below minimum", func(t *testing.T) {
		pf := &inventory.Platform{
			Title:   "Windows 11 Enterprise",
			Version: "26100",
			Build:   "2000",
			Arch:    "AMD64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.False(t, hotpatchSupported(pf))
	})

	t.Run("client arm64 build 26100 with sufficient UBR", func(t *testing.T) {
		pf := &inventory.Platform{
			Title:   "Windows 11 Enterprise",
			Version: "26100",
			Build:   "5000",
			Arch:    "ARM64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.True(t, hotpatchSupported(pf))
	})

	t.Run("client arm64 build 26100 with exact minimum UBR", func(t *testing.T) {
		pf := &inventory.Platform{
			Title:   "Windows 11 Enterprise",
			Version: "26100",
			Build:   "4929",
			Arch:    "ARM64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.True(t, hotpatchSupported(pf))
	})

	t.Run("client arm64 build 26100 with UBR below arm64 minimum", func(t *testing.T) {
		pf := &inventory.Platform{
			Title:   "Windows 11 Enterprise",
			Version: "26100",
			Build:   "3775",
			Arch:    "ARM64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.False(t, hotpatchSupported(pf))
	})

	t.Run("client build 26100 with empty UBR", func(t *testing.T) {
		pf := &inventory.Platform{
			Title:   "Windows 11 Enterprise",
			Version: "26100",
			Build:   "",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.False(t, hotpatchSupported(pf))
	})

	t.Run("client build above 26100 always supported", func(t *testing.T) {
		pf := &inventory.Platform{
			Title:   "Windows 11 Enterprise",
			Version: "27000",
			Build:   "100",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.True(t, hotpatchSupported(pf))
	})

	t.Run("client build 22000 not supported", func(t *testing.T) {
		pf := &inventory.Platform{
			Title:   "Windows 11 Enterprise",
			Version: "22000",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.False(t, hotpatchSupported(pf))
	})

	t.Run("server build 20348 supported", func(t *testing.T) {
		pf := &inventory.Platform{
			Version: "20348",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "3"},
		}
		assert.True(t, hotpatchSupported(pf))
	})

	t.Run("server build 19041 not supported", func(t *testing.T) {
		pf := &inventory.Platform{
			Version: "19041",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "3"},
		}
		assert.False(t, hotpatchSupported(pf))
	})

	// Edition guard: Pro / Home / Pro Education boxes can never receive
	// hotpatches even when the build and UBR meet the prerequisites — no
	// Microsoft license activates hotpatching while the OS reports as Pro
	// or Home. KB Ranger flagged a real Win11 Pro AMD64 26200.8246 asset
	// for KB5089466 because the registry signals were set but the edition
	// makes it ineligible.

	t.Run("client Pro 26200 with high UBR rejected by edition guard", func(t *testing.T) {
		pf := &inventory.Platform{
			Title:   "Windows 11 Pro",
			Version: "26200",
			Build:   "8246",
			Arch:    "AMD64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.False(t, hotpatchSupported(pf))
	})

	t.Run("client Pro for Workstations rejected by edition guard", func(t *testing.T) {
		pf := &inventory.Platform{
			Title:   "Windows 11 Pro for Workstations",
			Version: "27000",
			Arch:    "AMD64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.False(t, hotpatchSupported(pf))
	})

	t.Run("client Home rejected by edition guard", func(t *testing.T) {
		pf := &inventory.Platform{
			Title:   "Windows 11 Home",
			Version: "27000",
			Arch:    "AMD64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.False(t, hotpatchSupported(pf))
	})

	t.Run("client Pro Education rejected by edition guard", func(t *testing.T) {
		// Pro Education is in the Pro family (Win Edu A1 license), distinct
		// from the Education A3/A5 SKUs that the hotpatch license list
		// covers. Must be rejected before the broader "education" match.
		pf := &inventory.Platform{
			Title:   "Windows 11 Pro Education",
			Version: "27000",
			Arch:    "AMD64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.False(t, hotpatchSupported(pf))
	})

	t.Run("client Education accepted", func(t *testing.T) {
		pf := &inventory.Platform{
			Title:   "Windows 11 Education",
			Version: "27000",
			Arch:    "AMD64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.True(t, hotpatchSupported(pf))
	})

	t.Run("client IoT Enterprise accepted", func(t *testing.T) {
		pf := &inventory.Platform{
			Title:   "Windows 11 IoT Enterprise",
			Version: "27000",
			Arch:    "AMD64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.True(t, hotpatchSupported(pf))
	})

	t.Run("client Enterprise multi-session accepted", func(t *testing.T) {
		// Cloud PC / Win365 Enterprise multi-session SKU is in the eligible
		// license list and reports an "Enterprise multi-session" title.
		pf := &inventory.Platform{
			Title:   "Windows 11 Enterprise multi-session",
			Version: "27000",
			Arch:    "AMD64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.True(t, hotpatchSupported(pf))
	})

	t.Run("client empty title rejected", func(t *testing.T) {
		// Missing Title means we can't tell what's running. Hotpatch eligibility
		// must default to refuse — failing open here surfaces hotpatch-only
		// KBs on every asset whose detection didn't fill Title.
		pf := &inventory.Platform{
			Title:   "",
			Version: "27000",
			Arch:    "AMD64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.False(t, hotpatchSupported(pf))
	})
}

func TestIsHotpatchEligibleClientEdition(t *testing.T) {
	cases := []struct {
		title    string
		eligible bool
	}{
		{"Windows 11 Enterprise", true},
		{"Windows 11 Enterprise Evaluation", true},
		{"Windows 11 Enterprise multi-session", true},
		{"Windows 11 IoT Enterprise", true},
		{"Windows 11 Education", true},
		{"WINDOWS 11 ENTERPRISE", true}, // case-insensitive
		{"Windows 11 Pro", false},
		{"Windows 11 Pro for Workstations", false},
		{"Windows 11 Pro Education", false}, // Pro family, not Education A3/A5
		{"Windows 11 Home", false},
		{"Windows 11 Home Single Language", false},
		{"Windows 11 SE", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			assert.Equal(t, tc.eligible, isHotpatchEligibleClientEdition(tc.title))
		})
	}
}
