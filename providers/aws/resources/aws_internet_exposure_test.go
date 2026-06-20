// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNaclAllowsPublicIngress(t *testing.T) {
	tests := []struct {
		name  string
		rules []naclIngressRule
		want  bool
	}{
		{
			name:  "default nacl allows all",
			rules: []naclIngressRule{{ruleNumber: 100, allow: true, public: true}},
			want:  true,
		},
		{
			name: "lower-numbered deny shadows allow",
			rules: []naclIngressRule{
				{ruleNumber: 90, allow: false, public: true},
				{ruleNumber: 100, allow: true, public: true},
			},
			want: false,
		},
		{
			name: "lower-numbered allow wins over later deny",
			rules: []naclIngressRule{
				{ruleNumber: 200, allow: false, public: true},
				{ruleNumber: 100, allow: true, public: true},
			},
			want: true,
		},
		{
			name: "only specific-cidr allow, no public rule",
			rules: []naclIngressRule{
				{ruleNumber: 100, allow: true, public: false},
			},
			want: false,
		},
		{
			name:  "no rules",
			rules: []naclIngressRule{},
			want:  false,
		},
		{
			name: "non-public rules ignored, public deny decides",
			rules: []naclIngressRule{
				{ruleNumber: 50, allow: true, public: false},
				{ruleNumber: 100, allow: false, public: true},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, naclAllowsPublicIngress(tt.rules))
		})
	}
}

func TestFunctionUrlIsPublic(t *testing.T) {
	assert.True(t, functionUrlIsPublic("NONE"))
	assert.True(t, functionUrlIsPublic("none"))
	assert.False(t, functionUrlIsPublic("AWS_IAM"))
	assert.False(t, functionUrlIsPublic(""))
}

func TestRouteAuthIsPublic(t *testing.T) {
	assert.True(t, routeAuthIsPublic("NONE"))
	assert.True(t, routeAuthIsPublic("none"))
	assert.True(t, routeAuthIsPublic(""), "unset authorization type defaults to no auth")
	assert.False(t, routeAuthIsPublic("AWS_IAM"))
	assert.False(t, routeAuthIsPublic("JWT"))
	assert.False(t, routeAuthIsPublic("CUSTOM"))
}
