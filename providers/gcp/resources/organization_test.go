// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuditConfigID(t *testing.T) {
	t.Run("organization parent", func(t *testing.T) {
		result := auditConfigID("organizations/123456", "allServices")
		assert.Equal(t, "organizations/123456-auditConfig-allServices", result)
	})

	t.Run("project parent", func(t *testing.T) {
		result := auditConfigID("projects/my-project", "storage.googleapis.com")
		assert.Equal(t, "projects/my-project-auditConfig-storage.googleapis.com", result)
	})
}

func TestAuditLogConfigID(t *testing.T) {
	t.Run("organization parent", func(t *testing.T) {
		result := auditLogConfigID("organizations/123456", "allServices", "ADMIN_READ")
		assert.Equal(t, "organizations/123456-auditConfig-allServices-ADMIN_READ", result)
	})

	t.Run("project parent", func(t *testing.T) {
		result := auditLogConfigID("projects/my-project", "storage.googleapis.com", "DATA_WRITE")
		assert.Equal(t, "projects/my-project-auditConfig-storage.googleapis.com-DATA_WRITE", result)
	})

	t.Run("different log types produce unique IDs", func(t *testing.T) {
		adminRead := auditLogConfigID("organizations/123", "allServices", "ADMIN_READ")
		dataWrite := auditLogConfigID("organizations/123", "allServices", "DATA_WRITE")
		dataRead := auditLogConfigID("organizations/123", "allServices", "DATA_READ")
		assert.NotEqual(t, adminRead, dataWrite)
		assert.NotEqual(t, adminRead, dataRead)
		assert.NotEqual(t, dataWrite, dataRead)
	})

	t.Run("different services produce unique IDs", func(t *testing.T) {
		all := auditLogConfigID("organizations/123", "allServices", "ADMIN_READ")
		storage := auditLogConfigID("organizations/123", "storage.googleapis.com", "ADMIN_READ")
		assert.NotEqual(t, all, storage)
	})
}
