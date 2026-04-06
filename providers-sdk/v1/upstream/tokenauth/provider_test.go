// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package tokenauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name      string
		issuerURI string
		wantType  TokenProvider
		wantErr   string
	}{
		{
			name:      "AWS STS",
			issuerURI: "https://sts.amazonaws.com",
			wantType:  &AWSTokenProvider{},
		},
		{
			name:      "GCP",
			issuerURI: "https://accounts.google.com",
			wantType:  &GCPTokenProvider{},
		},
		{
			name:      "Azure",
			issuerURI: "https://login.microsoftonline.com/tenant-id/v2.0",
			wantType:  &AzureTokenProvider{},
		},
		{
			name:      "GitHub Actions",
			issuerURI: "https://token.actions.githubusercontent.com",
			wantType:  &GitHubTokenProvider{},
		},
		{
			name:      "unsupported issuer",
			issuerURI: "https://unknown.example.com",
			wantErr:   "unsupported WIF issuer URI: https://unknown.example.com",
		},
		{
			name:      "empty issuer",
			issuerURI: "",
			wantErr:   "unsupported WIF issuer URI: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := Resolve(tt.issuerURI)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				assert.IsType(t, tt.wantType, provider)
			}
		})
	}
}
