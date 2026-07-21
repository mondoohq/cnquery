// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
)

// TestAssessment_ContextResourceListSurfacesFailures is an end-to-end regression
// test for "empty assessments": a failing list assertion over a resource that
// carries a `@context` annotation (here sshd.config.matchBlock) must surface the
// failing resources in the assessment's `actual` value.
//
// The context block has sub-fields (e.g. range) that can be unset; an unset
// sub-field serialized to an untyped primitive used to abort conversion of the
// entire failing-resource list, leaving `actual` empty. Reporters (CLI, SARIF)
// then showed no failing resources and no source locations.
func TestAssessment_ContextResourceListSurfacesFailures(t *testing.T) {
	bundle, err := x.Compile(`sshd.config.blocks.all(criteria == "nonexistent")`)
	require.NoError(t, err)

	results, err := x.ExecuteCode(bundle, nil)
	require.NoError(t, err)

	assessment := llx.Results2AssessmentLookupV2(bundle, func(s string) (*llx.RawResult, bool) {
		r, ok := results[s]
		return r, ok
	})
	require.NotNil(t, assessment)
	require.False(t, assessment.Success, "the check is expected to fail")
	require.NotEmpty(t, assessment.Results)

	item := assessment.Results[0]
	require.False(t, item.Success)
	require.NotNil(t, item.Actual, "failing resources must be surfaced in actual")

	rd := item.Actual.RawData()
	require.NoError(t, rd.Error)
	arr, ok := rd.Value.([]any)
	require.True(t, ok, "actual should be the list of failing resources")
	assert.NotEmpty(t, arr, "the failing-resource list must not be empty")
}
