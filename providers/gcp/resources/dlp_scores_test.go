// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"cloud.google.com/go/dlp/apiv2/dlppb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// The shipped docs claimed the score dict carried HIGH / MEDIUM_LOW values and
// "numeric breakdowns". Neither is true: the message has exactly one field and
// the enum names are SENSITIVITY_*. A policy written against the documented
// names never matched anything.
func TestDlpSensitivityScoreLevel(t *testing.T) {
	for _, tc := range []struct {
		name  string
		score *dlppb.SensitivityScore
		want  string
	}{
		{"high", &dlppb.SensitivityScore{Score: dlppb.SensitivityScore_SENSITIVITY_HIGH}, "SENSITIVITY_HIGH"},
		{"moderate", &dlppb.SensitivityScore{Score: dlppb.SensitivityScore_SENSITIVITY_MODERATE}, "SENSITIVITY_MODERATE"},
		{"low", &dlppb.SensitivityScore{Score: dlppb.SensitivityScore_SENSITIVITY_LOW}, "SENSITIVITY_LOW"},
		// DLP looked and could not tell.
		{"unknown", &dlppb.SensitivityScore{Score: dlppb.SensitivityScore_SENSITIVITY_UNKNOWN}, "SENSITIVITY_UNKNOWN"},
		{"unspecified", &dlppb.SensitivityScore{}, "SENSITIVITY_SCORE_UNSPECIFIED"},
		// No score reported at all, which is not the same as SENSITIVITY_UNKNOWN.
		{"absent", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, dlpSensitivityScoreLevel(tc.score))
		})
	}
}

func TestDlpDataRiskScore(t *testing.T) {
	for _, tc := range []struct {
		name  string
		level *dlppb.DataRiskLevel
		want  string
	}{
		{"high", &dlppb.DataRiskLevel{Score: dlppb.DataRiskLevel_RISK_HIGH}, "RISK_HIGH"},
		{"moderate", &dlppb.DataRiskLevel{Score: dlppb.DataRiskLevel_RISK_MODERATE}, "RISK_MODERATE"},
		{"low", &dlppb.DataRiskLevel{Score: dlppb.DataRiskLevel_RISK_LOW}, "RISK_LOW"},
		{"unknown", &dlppb.DataRiskLevel{Score: dlppb.DataRiskLevel_RISK_UNKNOWN}, "RISK_UNKNOWN"},
		{"unspecified", &dlppb.DataRiskLevel{}, "RISK_SCORE_UNSPECIFIED"},
		{"absent", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, dlpDataRiskScore(tc.level))
		})
	}
}

// A failed regeneration is the difference between a current profile and a stale
// one, so the code, the message, and the time of the attempt all have to survive.
func TestDlpProfileStatusArgsFailedGeneration(t *testing.T) {
	when := time.Date(2024, 7, 4, 9, 30, 0, 0, time.UTC)
	args := dlpProfileStatusArgs(&dlppb.ProfileStatus{
		Status:    &status.Status{Code: 7, Message: "permission denied on the source table"},
		Timestamp: timestamppb.New(when),
	})

	assert.EqualValues(t, 7, args["profileStatusCode"].Value)
	assert.Equal(t, "permission denied on the source table", args["profileStatusMessage"].Value)
	got, ok := args["profileStatusTimestamp"].Value.(*time.Time)
	require.True(t, ok)
	assert.Equal(t, when, got.UTC())
}

// Code 0 is a successful generation. It has to read as 0, not as null, or a
// check for "the profile generated cleanly" cannot be written.
func TestDlpProfileStatusArgsSuccessIsZeroNotNull(t *testing.T) {
	args := dlpProfileStatusArgs(&dlppb.ProfileStatus{
		Status: &status.Status{Code: 0},
	})

	assert.EqualValues(t, 0, args["profileStatusCode"].Value)
	assert.Equal(t, "", args["profileStatusMessage"].Value)
}

// No status reported must stay null rather than collapsing onto code 0, which
// would report an unrun generation as a successful one. The timestamp must
// likewise stay null rather than becoming the zero time.
func TestDlpProfileStatusArgsAbsent(t *testing.T) {
	args := dlpProfileStatusArgs(nil)

	assert.Nil(t, args["profileStatusCode"].Value)
	assert.Equal(t, "", args["profileStatusMessage"].Value)
	assert.Nil(t, args["profileStatusTimestamp"].Value)

	// A ProfileStatus with a timestamp but no status message behaves the same
	// way on the code, and still reports the time.
	partial := dlpProfileStatusArgs(&dlppb.ProfileStatus{Timestamp: timestamppb.Now()})
	assert.Nil(t, partial["profileStatusCode"].Value)
	assert.NotNil(t, partial["profileStatusTimestamp"].Value)
}
