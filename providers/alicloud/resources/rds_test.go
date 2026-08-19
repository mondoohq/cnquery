// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	rdsclient "github.com/alibabacloud-go/rds-20140815/v16/client"
	tea "github.com/alibabacloud-go/tea/tea"
	"github.com/stretchr/testify/assert"
)

// TestRdsStatusEnabled covers the on/off classifier shared by the TDE and SQL
// Explorer readers. The two APIs spell the same state differently (Enabled vs
// Enable), and every unrecognised value has to read as off: a status that
// silently became true would report encryption or auditing on an instance that
// has neither.
func TestRdsStatusEnabled(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status *string
		want   bool
	}{
		{"nil is off", nil, false},
		{"empty is off", tea.String(""), false},
		{"TDE spelling", tea.String("Enabled"), true},
		{"SQL Explorer spelling", tea.String("Enable"), true},
		{"lowercase", tea.String("enabled"), true},
		{"surrounding space", tea.String("  Enable  "), true},
		{"disabled", tea.String("Disabled"), false},
		{"disable", tea.String("Disable"), false},
		{"unrecognised value is off", tea.String("Pending"), false},
		{"substring must not match", tea.String("NotEnabled"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, rdsStatusEnabled(tc.status))
		})
	}
}

// TestRdsRetentionDays covers the string-to-days conversion for the SQL
// Explorer retention window. Anything unreadable must land on 0, which fails a
// minimum-retention assertion rather than satisfying it with a made-up number.
func TestRdsRetentionDays(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value *string
		want  int64
	}{
		{"nil is zero", nil, 0},
		{"empty is zero", tea.String(""), 0},
		{"garbage is zero", tea.String("forever"), 0},
		{"negative is zero", tea.String("-1"), 0},
		{"thirty days", tea.String("30"), 30},
		{"surrounding space", tea.String(" 365 "), 365},
		{"six months", tea.String("180"), 180},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, rdsRetentionDays(tc.value))
		})
	}
}

// TestRdsRunningParameters covers the parameter map build. It must read the
// running parameters and not the configured ones: a parameter changed but not
// yet applied is not in effect, and reporting it would say a PostgreSQL
// instance is logging connections before the restart that makes it true.
func TestRdsRunningParameters(t *testing.T) {
	param := func(name, value string) *rdsclient.DescribeParametersResponseBodyRunningParametersDBInstanceParameter {
		return &rdsclient.DescribeParametersResponseBodyRunningParametersDBInstanceParameter{
			ParameterName:  tea.String(name),
			ParameterValue: tea.String(value),
		}
	}

	t.Run("nil body is an empty map, not nil", func(t *testing.T) {
		assert.Equal(t, map[string]any{}, rdsRunningParameters(nil))
	})

	t.Run("no running parameters is an empty map", func(t *testing.T) {
		assert.Equal(t, map[string]any{}, rdsRunningParameters(&rdsclient.DescribeParametersResponseBody{}))
	})

	t.Run("running parameters are keyed by name", func(t *testing.T) {
		got := rdsRunningParameters(&rdsclient.DescribeParametersResponseBody{
			RunningParameters: &rdsclient.DescribeParametersResponseBodyRunningParameters{
				DBInstanceParameter: []*rdsclient.DescribeParametersResponseBodyRunningParametersDBInstanceParameter{
					param("log_connections", "on"),
					param("log_disconnections", "off"),
					param("log_duration", "on"),
				},
			},
		})
		assert.Equal(t, map[string]any{
			"log_connections":    "on",
			"log_disconnections": "off",
			"log_duration":       "on",
		}, got)
	})

	t.Run("nil and unnamed entries are dropped", func(t *testing.T) {
		got := rdsRunningParameters(&rdsclient.DescribeParametersResponseBody{
			RunningParameters: &rdsclient.DescribeParametersResponseBodyRunningParameters{
				DBInstanceParameter: []*rdsclient.DescribeParametersResponseBodyRunningParametersDBInstanceParameter{
					nil,
					{ParameterName: tea.String("")},
					{ParameterName: nil, ParameterValue: tea.String("on")},
					param("log_duration", "on"),
				},
			},
		})
		assert.Equal(t, map[string]any{"log_duration": "on"}, got)
	})

	t.Run("a parameter with no value reports empty, not missing", func(t *testing.T) {
		got := rdsRunningParameters(&rdsclient.DescribeParametersResponseBody{
			RunningParameters: &rdsclient.DescribeParametersResponseBodyRunningParameters{
				DBInstanceParameter: []*rdsclient.DescribeParametersResponseBodyRunningParametersDBInstanceParameter{
					{ParameterName: tea.String("log_statement")},
				},
			},
		})
		assert.Equal(t, map[string]any{"log_statement": ""}, got)
	})

	t.Run("configured-but-unapplied parameters are not reported", func(t *testing.T) {
		got := rdsRunningParameters(&rdsclient.DescribeParametersResponseBody{
			ConfigParameters: &rdsclient.DescribeParametersResponseBodyConfigParameters{
				DBInstanceParameter: []*rdsclient.DescribeParametersResponseBodyConfigParametersDBInstanceParameter{
					{ParameterName: tea.String("log_connections"), ParameterValue: tea.String("on")},
				},
			},
		})
		assert.Equal(t, map[string]any{}, got)
	})
}
