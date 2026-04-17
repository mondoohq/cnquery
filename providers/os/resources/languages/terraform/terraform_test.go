// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseProviderSource(t *testing.T) {
	tests := []struct {
		source            string
		expectedNamespace string
		expectedType      string
	}{
		// Default registry (prefix stripped)
		{"registry.terraform.io/hashicorp/aws", "hashicorp", "aws"},
		{"registry.terraform.io/hashicorp/random", "hashicorp", "random"},
		{"registry.terraform.io/integrations/github", "integrations", "github"},
		// Already stripped (namespace/type)
		{"hashicorp/aws", "hashicorp", "aws"},
		// Custom registry (3 parts after no prefix strip)
		{"custom.registry.io/myorg/myprovider", "myorg", "myprovider"},
		// Single part
		{"aws", "", "aws"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			ns, pt := ParseProviderSource(tt.source)
			assert.Equal(t, tt.expectedNamespace, ns)
			assert.Equal(t, tt.expectedType, pt)
		})
	}
}

func TestNewPackageUrl(t *testing.T) {
	assert.Equal(t, "pkg:terraform/hashicorp/aws@5.31.0", NewPackageUrl("hashicorp", "aws", "5.31.0"))
	assert.Equal(t, "pkg:terraform/integrations/github@5.42.0", NewPackageUrl("integrations", "github", "5.42.0"))
}
