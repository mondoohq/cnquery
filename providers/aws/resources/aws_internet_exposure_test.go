// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
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

func setBoolValue(v bool) plugin.TValue[bool] {
	return plugin.TValue[bool]{Data: v, State: plugin.StateIsSet}
}

func TestEsDomainIsPublic(t *testing.T) {
	tests := []struct {
		name               string
		inVPC              bool
		policyAllowsPublic bool
		want               bool
	}{
		{"public endpoint, public policy", false, true, true},
		{"public endpoint, scoped policy", false, false, false},
		{"vpc domain, public policy", true, true, false},
		{"vpc domain, scoped policy", true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, esDomainIsPublic(tt.inVPC, tt.policyAllowsPublic))
		})
	}
}

func TestRedshiftClusterInternetReachable(t *testing.T) {
	tests := []struct {
		name               string
		publiclyAccessible bool
		want               bool
	}{
		{"publicly accessible", true, true},
		{"private", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &mqlAwsRedshiftCluster{
				PubliclyAccessible: setBoolValue(tt.publiclyAccessible),
			}
			got, err := cluster.internetReachable()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOpenIngressRulesFromSecurityGroupsNil(t *testing.T) {
	// A nil security-group TValue (e.g. a Neptune instance whose parent cluster
	// could not be located) must yield no open rules rather than panicking.
	rules, err := openIngressRulesFromSecurityGroups(nil)
	require.NoError(t, err)
	assert.Empty(t, rules)
}
