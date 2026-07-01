// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManagedByFromTags(t *testing.T) {
	tests := []struct {
		name string
		tags map[string]any
		want string
	}{
		{"nil tags", nil, ""},
		{"empty tags", map[string]any{}, ""},
		{"unrelated tags", map[string]any{"Name": "web", "env": "prod"}, ""},
		{"cloudformation", map[string]any{"aws:cloudformation:stack-name": "my-stack"}, "cloudformation"},
		{"elasticbeanstalk", map[string]any{"elasticbeanstalk:environment-name": "my-env"}, "elasticbeanstalk"},
		{"eks", map[string]any{"eks:cluster-name": "my-cluster"}, "eks"},
		{"servicecatalog", map[string]any{"aws:servicecatalog:provisioningPrincipalArn": "arn:aws:iam::1:user/x"}, "servicecatalog"},
		{"autoscaling", map[string]any{"aws:autoscaling:groupName": "my-asg"}, "autoscaling"},
		// The signal list is ordered; cloudformation is checked before eks, so a
		// resource carrying both reports cloudformation.
		{"precedence cloudformation over eks", map[string]any{
			"eks:cluster-name":              "my-cluster",
			"aws:cloudformation:stack-name": "my-stack",
		}, "cloudformation"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, managedByFromTags(tt.tags))
		})
	}
}

func TestManagedByWithCreationToken(t *testing.T) {
	tests := []struct {
		name          string
		tagOwner      string
		creationToken string
		want          string
	}{
		{"tag owner wins over terraform token", "cloudformation", "terraform-20240101000000000000000001", "cloudformation"},
		{"terraform token when no tag owner", "", "terraform-20240101000000000000000001", "terraform"},
		{"custom token, no tag owner", "", "my-own-token", ""},
		{"empty token, no tag owner", "", "", ""},
		{"tag owner with no token", "eks", "", "eks"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, managedByWithCreationToken(tt.tagOwner, tt.creationToken))
		})
	}
}
