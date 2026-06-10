// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

func TestSecurityNotificationEmailFlag(t *testing.T) {
	emails := map[string]any{
		"reportSuspiciousActivityEnabled":     true,
		"sendEmailForNewDeviceEnabled":        false,
		"sendEmailForFactorEnrollmentEnabled": true,
		"sendEmailForFactorResetEnabled":      false,
		"sendEmailForPasswordChangedEnabled":  true,
	}

	tests := []struct {
		name  string
		input any
		key   string
		want  bool
	}{
		{"true value", emails, "reportSuspiciousActivityEnabled", true},
		{"false value", emails, "sendEmailForNewDeviceEnabled", false},
		{"another true value", emails, "sendEmailForFactorEnrollmentEnabled", true},
		{"missing key", emails, "doesNotExist", false},
		{"nil input", nil, "reportSuspiciousActivityEnabled", false},
		{"wrong input type", "not a map", "reportSuspiciousActivityEnabled", false},
		{"non-bool value", map[string]any{"x": "true"}, "x", false},
		{"empty map", map[string]any{}, "reportSuspiciousActivityEnabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := securityNotificationEmailFlag(tt.input, tt.key); got != tt.want {
				t.Errorf("securityNotificationEmailFlag(%v, %q) = %v, want %v", tt.input, tt.key, got, tt.want)
			}
		})
	}
}
