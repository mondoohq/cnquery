// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractConstraintName(t *testing.T) {
	t.Run("organization policy path", func(t *testing.T) {
		result := extractConstraintName("organizations/123456/policies/compute.disableSerialPortAccess")
		assert.Equal(t, "compute.disableSerialPortAccess", result)
	})

	t.Run("project policy path", func(t *testing.T) {
		result := extractConstraintName("projects/my-project/policies/iam.allowedPolicyMemberDomains")
		assert.Equal(t, "iam.allowedPolicyMemberDomains", result)
	})

	t.Run("folder policy path", func(t *testing.T) {
		result := extractConstraintName("folders/987654/policies/storage.uniformBucketLevelAccess")
		assert.Equal(t, "storage.uniformBucketLevelAccess", result)
	})

	t.Run("no policies segment returns full name", func(t *testing.T) {
		result := extractConstraintName("compute.disableSerialPortAccess")
		assert.Equal(t, "compute.disableSerialPortAccess", result)
	})

	t.Run("empty string", func(t *testing.T) {
		result := extractConstraintName("")
		assert.Equal(t, "", result)
	})

	t.Run("path ending with /policies/ returns empty constraint", func(t *testing.T) {
		result := extractConstraintName("organizations/123/policies/")
		assert.Equal(t, "", result)
	})

	t.Run("uses last occurrence of /policies/", func(t *testing.T) {
		result := extractConstraintName("organizations/123/policies/nested/policies/actual.constraint")
		assert.Equal(t, "actual.constraint", result)
	})
}
