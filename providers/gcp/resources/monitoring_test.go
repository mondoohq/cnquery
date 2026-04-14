// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/stretchr/testify/assert"
	calendarperiodpb "google.golang.org/genproto/googleapis/type/calendarperiod"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestSLORollingPeriodExtraction(t *testing.T) {
	t.Run("rolling period 30 days", func(t *testing.T) {
		slo := &monitoringpb.ServiceLevelObjective{
			Period: &monitoringpb.ServiceLevelObjective_RollingPeriod{
				RollingPeriod: durationpb.New(30 * 24 * 60 * 60 * 1e9), // 30 days in nanoseconds
			},
		}
		if rp, ok := slo.Period.(*monitoringpb.ServiceLevelObjective_RollingPeriod); ok && rp.RollingPeriod != nil {
			result := durationpb.New(rp.RollingPeriod.AsDuration()).String()
			assert.NotEmpty(t, result)
		}
	})

	t.Run("calendar period MONTH", func(t *testing.T) {
		slo := &monitoringpb.ServiceLevelObjective{
			Period: &monitoringpb.ServiceLevelObjective_CalendarPeriod{
				CalendarPeriod: calendarperiodpb.CalendarPeriod_MONTH,
			},
		}
		if cp, ok := slo.Period.(*monitoringpb.ServiceLevelObjective_CalendarPeriod); ok {
			assert.Equal(t, "MONTH", cp.CalendarPeriod.String())
		}
	})
}

func TestSLOGoalRange(t *testing.T) {
	t.Run("valid SLO goals", func(t *testing.T) {
		// SLO goals must be 0 < goal <= 0.9999
		slo := &monitoringpb.ServiceLevelObjective{
			Goal: 0.999,
		}
		assert.Greater(t, slo.Goal, 0.0)
		assert.LessOrEqual(t, slo.Goal, 0.9999)
	})
}
