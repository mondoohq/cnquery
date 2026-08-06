// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== parseInfluxdbExpiry =====

func TestParseInfluxdbExpiry(t *testing.T) {
	tests := []struct {
		title string
		value *string
		want  *time.Time
	}{
		{"nil", nil, nil},
		{"empty", ptrStr(""), nil},
		{"not a time", ptrStr("tomorrow"), nil},
		// the API reports the expiry as a string, not as a timestamp
		{"utc", ptrStr("2026-08-06T14:35:11Z"), timePtr(time.Date(2026, 8, 6, 14, 35, 11, 0, time.UTC))},
		{"with offset", ptrStr("2026-08-06T07:35:11-07:00"), timePtr(time.Date(2026, 8, 6, 14, 35, 11, 0, time.UTC))},
		// a unix timestamp is not RFC 3339, and must not be read as one
		{"unix timestamp", ptrStr("1786026911"), nil},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			got := parseInfluxdbExpiry(test.value)
			if test.want == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.True(t, test.want.Equal(*got), "want %s, got %s", test.want, got)
		})
	}
}

// ===== backup typed-ref null-state =====

func TestInfluxdbBackupDbInstanceNullWhenNoResourceId(t *testing.T) {
	for _, cached := range []*string{nil, ptrStr("")} {
		b := &mqlAwsTimestreamInfluxdbBackup{}
		b.cacheDbResourceId = cached
		got, err := b.dbInstance()
		require.NoError(t, err)
		assert.Nil(t, got)
		assert.True(t, b.DbInstance.IsNull())
		assert.True(t, b.DbInstance.IsSet())
	}
}

func TestInfluxdbBackupKmsKeyNullWhenNoKeyId(t *testing.T) {
	for _, cached := range []*string{nil, ptrStr("")} {
		b := &mqlAwsTimestreamInfluxdbBackup{}
		b.cacheKmsKeyId = cached
		got, err := b.kmsKey()
		require.NoError(t, err)
		assert.Nil(t, got)
		assert.True(t, b.KmsKey.IsNull())
		assert.True(t, b.KmsKey.IsSet())
	}
}

func ptrStr(s string) *string { return &s }

func timePtr(t time.Time) *time.Time { return &t }
