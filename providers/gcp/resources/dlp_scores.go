// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"time"

	"cloud.google.com/go/dlp/apiv2/dlppb"
	"go.mondoo.com/mql/llx"
)

// dlpSensitivityScoreLevel reads the sensitivity level off a profile.
//
// dlppb.SensitivityScore has exactly one field, so the dict it was shipped as
// only ever carried this one value. Empty when the profile reports no score at
// all, which is a different claim from the explicit
// SENSITIVITY_SCORE_UNSPECIFIED the API sends when it has looked and cannot
// say.
func dlpSensitivityScoreLevel(s *dlppb.SensitivityScore) string {
	if s == nil {
		return ""
	}
	return s.Score.String()
}

// dlpDataRiskScore reads the data risk level off a profile. dlppb.DataRiskLevel
// likewise carries exactly one field.
func dlpDataRiskScore(l *dlppb.DataRiskLevel) string {
	if l == nil {
		return ""
	}
	return l.Score.String()
}

// dlpProfileStatusArgs flattens the status of the last profile-generation
// attempt.
//
// A non-zero code means the profile in hand is stale: DLP failed to regenerate
// it and the sensitivity and risk scores describe the data as it was at
// `profileStatusTimestamp`, not as it is now. The code stays null when no status
// was reported, so "generation succeeded" (code 0) and "no status" stay
// distinguishable.
func dlpProfileStatusArgs(s *dlppb.ProfileStatus) map[string]*llx.RawData {
	var code *int32
	var message string
	var ts *time.Time

	if s != nil {
		if st := s.GetStatus(); st != nil {
			c := st.Code
			code = &c
			message = st.Message
		}
		ts = timestampAsTimePtr(s.GetTimestamp())
	}

	return map[string]*llx.RawData{
		"profileStatusCode":      llx.IntDataPtr(code),
		"profileStatusMessage":   llx.StringData(message),
		"profileStatusTimestamp": llx.TimeDataPtr(ts),
	}
}
