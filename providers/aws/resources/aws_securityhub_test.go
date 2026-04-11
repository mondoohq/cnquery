// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStandardNameFromArn(t *testing.T) {
	t.Run("aws foundational standards ARN", func(t *testing.T) {
		name := standardNameFromArn("arn:aws:securityhub:us-west-2::standards/aws-foundational-security-best-practices/v/1.0.0")
		assert.Equal(t, "aws-foundational-security-best-practices", name)
	})

	t.Run("cis ruleset ARN", func(t *testing.T) {
		name := standardNameFromArn("arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0")
		assert.Equal(t, "cis-aws-foundations-benchmark", name)
	})

	t.Run("pci-dss standards ARN", func(t *testing.T) {
		name := standardNameFromArn("arn:aws:securityhub:us-east-1::standards/pci-dss/v/3.2.1")
		assert.Equal(t, "pci-dss", name)
	})

	t.Run("nist standards ARN", func(t *testing.T) {
		name := standardNameFromArn("arn:aws:securityhub:us-east-1::standards/nist-800-53/v/5.0.0")
		assert.Equal(t, "nist-800-53", name)
	})

	t.Run("ARN without standards or ruleset prefix returns full ARN", func(t *testing.T) {
		arn := "arn:aws:securityhub:us-west-2:123456789012:hub/default"
		name := standardNameFromArn(arn)
		assert.Equal(t, arn, name)
	})

	t.Run("empty string returns empty", func(t *testing.T) {
		assert.Equal(t, "", standardNameFromArn(""))
	})

	t.Run("standards ARN without version suffix", func(t *testing.T) {
		name := standardNameFromArn("arn:aws:securityhub:::standards/my-standard")
		assert.Equal(t, "my-standard", name)
	})
}
