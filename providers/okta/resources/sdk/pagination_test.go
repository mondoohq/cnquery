// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNextLinkURL covers the RFC 5988 Link-header parsing that drives raw
// pagination for the endpoints not served by the generated SDK (network zones,
// custom roles, API tokens).
func TestNextLinkURL(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
		want    string
	}{
		{
			name:    "next present",
			headers: []string{`<https://x.okta.com/api/v1/zones?after=abc>; rel="next"`},
			want:    "https://x.okta.com/api/v1/zones?after=abc",
		},
		{
			name:    "self only, no next",
			headers: []string{`<https://x.okta.com/api/v1/zones>; rel="self"`},
			want:    "",
		},
		{
			name: "self and next as separate headers",
			headers: []string{
				`<https://x.okta.com/self>; rel="self"`,
				`<https://x.okta.com/next>; rel="next"`,
			},
			want: "https://x.okta.com/next",
		},
		{
			name:    "no headers",
			headers: nil,
			want:    "",
		},
		{
			name:    "malformed header",
			headers: []string{"garbage-without-rel"},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, nextLinkURL(tt.headers))
		})
	}
}
