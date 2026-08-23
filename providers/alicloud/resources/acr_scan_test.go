// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

func TestAcrScanComplete(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{"COMPLETE", true},
		{"complete", true},
		{" COMPLETE ", true},
		// a scan still running has produced no findings yet
		{"SCANNING", false},
		{"RETRYING", false},
		// a failed scan produced nothing, which is not the same as clean
		{"FAILED", false},
		// never scanned, or the status could not be read
		{"", false},
	}
	for _, test := range tests {
		t.Run(test.status, func(t *testing.T) {
			assert.Equal(t, test.expected, acrScanComplete(test.status))
		})
	}
}

// acrTestVuln builds a vulnerability resource with only the member the tally
// reads.
func acrTestVuln(severity string) *mqlAlicloudAcrVulnerability {
	return &mqlAlicloudAcrVulnerability{
		Severity: plugin.TValue[string]{Data: severity, State: plugin.StateIsSet},
	}
}

func TestAcrSeverityCounts(t *testing.T) {
	t.Run("clean image", func(t *testing.T) {
		counts := acrSeverityCounts([]any{})
		assert.Equal(t, int64(0), counts["high"])
		assert.Empty(t, counts)
	})

	t.Run("tallies by severity, case insensitively", func(t *testing.T) {
		counts := acrSeverityCounts([]any{
			acrTestVuln("High"),
			acrTestVuln("high"),
			acrTestVuln("HIGH"),
			acrTestVuln("Medium"),
			acrTestVuln("Low"),
		})
		assert.Equal(t, int64(3), counts["high"])
		assert.Equal(t, int64(1), counts["medium"])
		assert.Equal(t, int64(1), counts["low"])
	})

	t.Run("unrated findings are counted, not dropped", func(t *testing.T) {
		counts := acrSeverityCounts([]any{
			acrTestVuln(""),
			acrTestVuln("   "),
			acrTestVuln("Unknown"),
		})
		assert.Equal(t, int64(3), counts["unknown"])
	})

	t.Run("non-vulnerability entries are ignored", func(t *testing.T) {
		counts := acrSeverityCounts([]any{"not a finding", nil, acrTestVuln("High")})
		assert.Equal(t, int64(1), counts["high"])
		assert.Len(t, counts, 1)
	})
}
