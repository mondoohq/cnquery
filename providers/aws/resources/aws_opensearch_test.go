// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	opensearch_types "github.com/aws/aws-sdk-go-v2/service/opensearch/types"
	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
)

func TestParseAuditLogEnabled(t *testing.T) {
	t.Run("nil map returns false", func(t *testing.T) {
		assert.False(t, parseAuditLogEnabled(nil))
	})

	t.Run("empty map returns false", func(t *testing.T) {
		opts := map[string]opensearch_types.LogPublishingOption{}
		assert.False(t, parseAuditLogEnabled(opts))
	})

	t.Run("map without AUDIT_LOGS key returns false", func(t *testing.T) {
		opts := map[string]opensearch_types.LogPublishingOption{
			"INDEX_SLOW_LOGS": {Enabled: convert.ToPtr(true)},
		}
		assert.False(t, parseAuditLogEnabled(opts))
	})

	t.Run("AUDIT_LOGS with nil Enabled returns false", func(t *testing.T) {
		opts := map[string]opensearch_types.LogPublishingOption{
			"AUDIT_LOGS": {Enabled: nil},
		}
		assert.False(t, parseAuditLogEnabled(opts))
	})

	t.Run("AUDIT_LOGS with Enabled false returns false", func(t *testing.T) {
		opts := map[string]opensearch_types.LogPublishingOption{
			"AUDIT_LOGS": {Enabled: convert.ToPtr(false)},
		}
		assert.False(t, parseAuditLogEnabled(opts))
	})

	t.Run("AUDIT_LOGS with Enabled true returns true", func(t *testing.T) {
		opts := map[string]opensearch_types.LogPublishingOption{
			"AUDIT_LOGS": {Enabled: convert.ToPtr(true)},
		}
		assert.True(t, parseAuditLogEnabled(opts))
	})
}
