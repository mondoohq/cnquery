// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractSharepointTenant(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"bare onmicrosoft", "contoso.onmicrosoft.com", "contoso", false},
		{"bare sharepoint", "contoso.sharepoint.com", "contoso", false},
		{"https scheme", "https://contoso.sharepoint.com", "contoso", false},
		{"http scheme", "http://contoso.sharepoint.com", "contoso", false},
		{"https with trailing slash", "https://contoso.sharepoint.com/", "contoso", false},
		{"https with path", "https://contoso.sharepoint.com/sites/foo", "contoso", false},
		{"empty", "", "", true},
		{"single label", "contoso", "", true},
		{"leading dot", ".contoso.com", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractSharepointTenant(tc.in)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
