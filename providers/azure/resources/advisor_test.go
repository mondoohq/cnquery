// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMeanOfPresentScores verifies the average divides by the count of scores
// actually present (non-nil), not the input length — otherwise unresolved
// scores skew the average toward zero.
func TestMeanOfPresentScores(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	// 10 + 20 over 2 present (the nil is skipped in BOTH sum and denominator)
	assert.Equal(t, 15.0, meanOfPresentScores([]*float64{f(10), nil, f(20)}))
	// all present
	assert.Equal(t, 20.0, meanOfPresentScores([]*float64{f(10), f(30)}))
	// single
	assert.Equal(t, 7.0, meanOfPresentScores([]*float64{f(7)}))
	// none present / empty
	assert.Equal(t, 0.0, meanOfPresentScores([]*float64{nil, nil}))
	assert.Equal(t, 0.0, meanOfPresentScores(nil))
}
