// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveRdpValue(t *testing.T) {
	t.Run("group policy value wins over the effective value", func(t *testing.T) {
		policy := map[string]int64{"userauthentication": 0}
		effective := map[string]int64{"userauthentication": 1}
		assert.Equal(t, int64(0), resolveRdpValue(policy, effective, "UserAuthentication", 1))
	})

	t.Run("effective value is used when policy is unset", func(t *testing.T) {
		effective := map[string]int64{"securitylayer": 2}
		assert.Equal(t, int64(2), resolveRdpValue(nil, effective, "SecurityLayer", 1))
	})

	t.Run("default is used when neither policy nor effective is set", func(t *testing.T) {
		assert.Equal(t, int64(2), resolveRdpValue(nil, nil, "MinEncryptionLevel", 2))
	})

	t.Run("value name matching is case insensitive", func(t *testing.T) {
		policy := map[string]int64{"fdisablecdm": 1}
		assert.Equal(t, int64(1), resolveRdpValue(policy, nil, "fDisableCdm", 0))
	})

	t.Run("an explicit zero is honored, not treated as unset", func(t *testing.T) {
		policy := map[string]int64{"fencryptrpctraffic": 0}
		// default is 1, but the policy explicitly disables it
		assert.Equal(t, int64(0), resolveRdpValue(policy, nil, "fEncryptRPCTraffic", 1))
	})
}

// TestRdpSettingDefaults locks in the Windows default each newly added field
// resolves to when neither group policy nor the effective key is configured.
func TestRdpSettingDefaults(t *testing.T) {
	cases := []struct {
		name string
		def  int64
	}{
		{"fDenyTSConnections", 1},
		{"fSingleSessionPerUser", 1},
		{"PerSessionTempDir", 1},
		{"fAllowToGetHelp", 0},
		{"fAllowUnsolicited", 0},
		{"fDisableWebAuthn", 0},
		{"fDisableLocationRedir", 0},
		{"EnableUiaRedirection", 0},
		{"SCClipLevel", 3},
		{"fResetBroken", 0},
		{"DisableCloudClipboardIntegration", 0},
	}
	for _, c := range cases {
		t.Run(c.name+" default", func(t *testing.T) {
			assert.Equal(t, c.def, resolveRdpValue(nil, nil, c.name, c.def))
		})
		t.Run(c.name+" policy overrides default", func(t *testing.T) {
			policy := map[string]int64{toLowerASCII(c.name): 2}
			assert.Equal(t, int64(2), resolveRdpValue(policy, nil, c.name, c.def))
		})
	}
}

func toLowerASCII(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}
