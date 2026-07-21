// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

// TestParseAlertPolicyConditionFilterMetricName ensures the function tolerates
// conditions without a "filter" key (e.g. monitoringQueryLanguage conditions
// carry only a "query"). Before the comma-ok guard, the unchecked type
// assertion panicked and crashed the whole scan.
func TestParseAlertPolicyConditionFilterMetricName(t *testing.T) {
	tests := []struct {
		name      string
		condition map[string]any
		want      string
	}{
		{
			name:      "user metric filter",
			condition: map[string]any{"filter": `metric.type="logging.googleapis.com/user/my-log-metric" AND resource.type="global"`},
			want:      "my-log-metric",
		},
		{
			name:      "filter without a user metric",
			condition: map[string]any{"filter": `metric.type="compute.googleapis.com/instance/cpu/utilization"`},
			want:      "",
		},
		{
			name:      "no filter key (monitoringQueryLanguage condition)",
			condition: map[string]any{"query": "fetch gce_instance"},
			want:      "",
		},
		{
			name:      "filter is not a string",
			condition: map[string]any{"filter": 42},
			want:      "",
		},
		{
			name:      "empty condition",
			condition: map[string]any{},
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseAlertPolicyConditionFilterMetricName(tt.condition); got != tt.want {
				t.Errorf("parseAlertPolicyConditionFilterMetricName() = %q, want %q", got, tt.want)
			}
		})
	}
}
