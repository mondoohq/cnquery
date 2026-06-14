// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package gcpinstancesnapshot

import (
	"regexp"
	"testing"
	"time"
)

func TestBuildDiskName(t *testing.T) {
	// GCP disk/snapshot name rule.
	nameRe := regexp.MustCompile(`^[a-z]([-a-z0-9]{0,61}[a-z0-9])?$`)

	// fixed timestamp so the test does not depend on the wall clock
	fixed := time.Date(2026, 6, 14, 13, 45, 7, 0, time.UTC)

	tests := []struct {
		name         string
		instanceName string
	}{
		{"short name", "web-1"},
		{"very long name (60 chars)", "super-important-server-with-a-really-long-descriptive-name-1"},
		{"uppercase, underscores, dots", "My_Server.Prod_01"},
		{"empty name", ""},
		{"only invalid chars", "____"},
		{"trailing dashes after truncation", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa--------------------"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildDiskName(tc.instanceName, fixed)

			if len(got) > 63 {
				t.Errorf("buildDiskName(%q) = %q has length %d, want <= 63", tc.instanceName, got, len(got))
			}
			if !nameRe.MatchString(got) {
				t.Errorf("buildDiskName(%q) = %q does not match GCP name rule %s", tc.instanceName, got, nameRe.String())
			}
			if got[:7] != "cnspec-" {
				t.Errorf("buildDiskName(%q) = %q does not start with %q", tc.instanceName, got, "cnspec-")
			}
		})
	}
}

func TestNewDiskName(t *testing.T) {
	nameRe := regexp.MustCompile(`^[a-z]([-a-z0-9]{0,61}[a-z0-9])?$`)
	got := newDiskName("super-important-server")
	if len(got) > 63 {
		t.Errorf("newDiskName produced %q with length %d, want <= 63", got, len(got))
	}
	if !nameRe.MatchString(got) {
		t.Errorf("newDiskName produced %q which does not match GCP name rule", got)
	}
}
