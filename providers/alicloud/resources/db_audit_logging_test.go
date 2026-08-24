// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"

	rkvclient "github.com/alibabacloud-go/r-kvstore-20150101/v7/client"
	tea "github.com/alibabacloud-go/tea/tea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisAuditEnabled(t *testing.T) {
	// the API returns this member as a string, not a boolean
	assert.True(t, redisAuditEnabled(tea.String("true")))
	assert.True(t, redisAuditEnabled(tea.String("True")))
	assert.True(t, redisAuditEnabled(tea.String(" true ")))
	assert.False(t, redisAuditEnabled(tea.String("false")))
	// an instance nobody could read must fail an "auditing is on" check, not
	// pass it
	assert.False(t, redisAuditEnabled(nil))
	assert.False(t, redisAuditEnabled(tea.String("")))
	assert.False(t, redisAuditEnabled(tea.String("1")))
}

func TestRedisAuditRetentionDays(t *testing.T) {
	tests := []struct {
		name      string
		retention *string
		expected  int64
	}{
		{"typical", tea.String("30"), 30},
		{"whitespace tolerated", tea.String(" 365 "), 365},
		{"zero", tea.String("0"), 0},
		// an unusable value must fail a minimum-retention check rather than
		// satisfy it
		{"absent", nil, 0},
		{"empty", tea.String(""), 0},
		{"unparseable", tea.String("thirty"), 0},
		{"negative", tea.String("-1"), 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, redisAuditRetentionDays(test.retention))
		})
	}
}

func TestPolardbAuditCollecting(t *testing.T) {
	assert.True(t, polardbAuditCollecting(tea.String("Enable")))
	assert.True(t, polardbAuditCollecting(tea.String("enable")))
	// transitional states are not yet recording
	assert.False(t, polardbAuditCollecting(tea.String("Enabling")))
	assert.False(t, polardbAuditCollecting(tea.String("Disabling")))
	assert.False(t, polardbAuditCollecting(tea.String("Disable")))
	assert.False(t, polardbAuditCollecting(nil))
	assert.False(t, polardbAuditCollecting(tea.String("")))
}

// TestRedisAuditConfigDecode pins the struct tags the two Redis fields depend
// on. Both members are strings on the wire, so a mistyped tag would silently
// report an audited instance as unaudited.
func TestRedisAuditConfigDecode(t *testing.T) {
	var body rkvclient.DescribeAuditLogConfigResponseBody
	require.NoError(t, json.Unmarshal([]byte(`{"DbAudit":"true","Retention":"30"}`), &body))

	assert.True(t, redisAuditEnabled(body.DbAudit))
	assert.Equal(t, int64(30), redisAuditRetentionDays(body.Retention))

	var empty rkvclient.DescribeAuditLogConfigResponseBody
	require.NoError(t, json.Unmarshal([]byte(`{}`), &empty))
	assert.False(t, redisAuditEnabled(empty.DbAudit))
	assert.Equal(t, int64(0), redisAuditRetentionDays(empty.Retention))
}
