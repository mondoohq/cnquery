// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
)

func TestSaStatus(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name  string
		sa    *github.SecurityAndAnalysis
		field saField
		want  string
	}{
		{
			name:  "nil SecurityAndAnalysis",
			sa:    nil,
			field: saAdvancedSecurity,
			want:  "",
		},
		{
			name:  "nil sub-field",
			sa:    &github.SecurityAndAnalysis{},
			field: saAdvancedSecurity,
			want:  "",
		},
		{
			name: "advanced security enabled",
			sa: &github.SecurityAndAnalysis{
				AdvancedSecurity: &github.AdvancedSecurity{Status: strPtr("enabled")},
			},
			field: saAdvancedSecurity,
			want:  "enabled",
		},
		{
			name: "secret scanning disabled",
			sa: &github.SecurityAndAnalysis{
				SecretScanning: &github.SecretScanning{Status: strPtr("disabled")},
			},
			field: saSecretScanning,
			want:  "disabled",
		},
		{
			name: "secret scanning push protection",
			sa: &github.SecurityAndAnalysis{
				SecretScanningPushProtection: &github.SecretScanningPushProtection{Status: strPtr("enabled")},
			},
			field: saSecretScanningPushProtection,
			want:  "enabled",
		},
		{
			name: "dependabot security updates",
			sa: &github.SecurityAndAnalysis{
				DependabotSecurityUpdates: &github.DependabotSecurityUpdates{Status: strPtr("disabled")},
			},
			field: saDependabotSecurityUpdates,
			want:  "disabled",
		},
		{
			name: "secret scanning validity checks",
			sa: &github.SecurityAndAnalysis{
				SecretScanningValidityChecks: &github.SecretScanningValidityChecks{Status: strPtr("enabled")},
			},
			field: saSecretScanningValidityChecks,
			want:  "enabled",
		},
		{
			name: "querying one field ignores others",
			sa: &github.SecurityAndAnalysis{
				AdvancedSecurity: &github.AdvancedSecurity{Status: strPtr("enabled")},
				SecretScanning:   &github.SecretScanning{Status: strPtr("disabled")},
			},
			field: saSecretScanning,
			want:  "disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := saStatus(tt.sa, tt.field)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPermissionsFromUser(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	tests := []struct {
		name string
		user *github.User
		want []string
	}{
		{
			name: "nil permissions",
			user: &github.User{},
			want: []string{},
		},
		{
			name: "admin only",
			user: &github.User{
				Permissions: &github.RepositoryPermissions{
					Admin: boolPtr(true),
					Push:  boolPtr(false),
					Pull:  boolPtr(false),
				},
			},
			want: []string{"admin"},
		},
		{
			name: "all permissions",
			user: &github.User{
				Permissions: &github.RepositoryPermissions{
					Admin:    boolPtr(true),
					Maintain: boolPtr(true),
					Push:     boolPtr(true),
					Triage:   boolPtr(true),
					Pull:     boolPtr(true),
				},
			},
			want: []string{"admin", "maintain", "push", "triage", "pull"},
		},
		{
			name: "push and pull only",
			user: &github.User{
				Permissions: &github.RepositoryPermissions{
					Admin:    boolPtr(false),
					Maintain: boolPtr(false),
					Push:     boolPtr(true),
					Triage:   boolPtr(false),
					Pull:     boolPtr(true),
				},
			},
			want: []string{"push", "pull"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := permissionsFromUser(tt.user)
			assert.Equal(t, tt.want, got)
		})
	}
}
