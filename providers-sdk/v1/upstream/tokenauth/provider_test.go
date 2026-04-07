// Copyright Mondoo, Inc. 2024, 2026
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
			name:      "Azure (legacy)",
			issuerURI: "https://sts.windows.net/tenant-id/",
			wantType:  &AzureTokenProvider{},
		},
		{
			name:      "GitHub Actions",
			issuerURI: "https://token.actions.githubusercontent.com",
			wantType:  &GitHubTokenProvider{},
		},
		{
			name:      "Capitalized letters don't affect matching",
			issuerURI: "https://Token.ACTIONS.GithubUserContent.com",
			wantType:  &GitHubTokenProvider{},
		},
		{
			name:      "unsupported issuer",
			issuerURI: "https://unknown.example.com",
			wantErr:   "issuer \"https://unknown.example.com\" not supported yet - open an issue https://github.com/mondoohq/mql/issues or see how to exchange tokens manually https://mondoo.com/docs/maintain/access/non-human/wif#exchange-tokens-manually",
		},
		{
			name:      "empty issuer",
			issuerURI: "",
			wantErr:   "issuer \"\" not supported yet - open an issue https://github.com/mondoohq/mql/issues or see how to exchange tokens manually https://mondoo.com/docs/maintain/access/non-human/wif#exchange-tokens-manually",
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
