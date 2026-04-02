// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

func TestUacHasFlag(t *testing.T) {
	tests := []struct {
		name string
		uac  int64
		flag int64
		want bool
	}{
		// Single flags
		{"disabled account", UACAccountDisable, UACAccountDisable, true},
		{"password never expires", UACDontExpirePassword, UACDontExpirePassword, true},
		{"preauth not required", UACDontRequirePreauth, UACDontRequirePreauth, true},
		{"normal account", UACNormalAccount, UACNormalAccount, true},

		// Flag not set
		{"disabled not set on normal", UACNormalAccount, UACAccountDisable, false},
		{"zero has no flags", 0, UACAccountDisable, false},
		{"zero has no preauth flag", 0, UACDontRequirePreauth, false},

		// Combined flags: 0x0202 = NormalAccount (0x0200) + AccountDisable (0x0002)
		{"combined: disabled detected", 0x0202, UACAccountDisable, true},
		{"combined: normal detected", 0x0202, UACNormalAccount, true},
		{"combined: preauth not present", 0x0202, UACDontRequirePreauth, false},

		// Real-world UAC values
		{"512 enabled normal account", 0x0200, UACNormalAccount, true},
		{"512 not disabled", 0x0200, UACAccountDisable, false},
		{"514 disabled normal account", 0x0202, UACAccountDisable, true},
		{"66048 password never expires", 0x10200, UACDontExpirePassword, true},
		{"66048 not disabled", 0x10200, UACAccountDisable, false},
		{"4260352 preauth not required", 0x410200, UACDontRequirePreauth, true},
		{"4260352 normal account", 0x410200, UACNormalAccount, true},

		// Delegation flags
		{"trusted for delegation", UACTrustedForDelegation, UACTrustedForDelegation, true},
		{"not delegated", UACNotDelegated, UACNotDelegated, true},
		{"trusted to auth for delegation", UACTrustedToAuthForDelegation, UACTrustedToAuthForDelegation, true},
		{"DES only", UACUseDesKeyOnly, UACUseDesKeyOnly, true},

		// Server trust (domain controllers)
		{"server trust", UACServerTrustAccount, UACServerTrustAccount, true},
		{"RODC partial secrets", UACPartialSecretsAccount, UACPartialSecretsAccount, true},

		// All security-relevant flags combined
		{"all flags: disabled", 0x05E703FF, UACAccountDisable, true},
		{"all flags: lockout", 0x05E703FF, UACLockout, true},
		{"all flags: reversible", 0x05E703FF, UACEncryptedTextPwdAllowed, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uacHasFlag(tt.uac, tt.flag)
			if got != tt.want {
				t.Errorf("uacHasFlag(0x%X, 0x%X) = %v, want %v", tt.uac, tt.flag, got, tt.want)
			}
		})
	}
}
