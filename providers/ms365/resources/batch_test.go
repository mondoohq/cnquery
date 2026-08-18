// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// batchStatusHasNoBody gates whether a sub-response reaches the deserializer.
// Getting it wrong in either direction is a real bug: too narrow and a
// body-less success panics the parse, too wide and a real payload is silently
// dropped.
func TestBatchStatusHasNoBody(t *testing.T) {
	tests := []struct {
		name string
		code int32
		want bool
	}{
		{"204 no content", http.StatusNoContent, true},
		{"205 reset content", http.StatusResetContent, true},
		{"200 ok carries a body", http.StatusOK, false},
		{"201 created carries a body", http.StatusCreated, false},
		{"202 accepted carries a body", http.StatusAccepted, false},
		{"206 partial content carries a body", http.StatusPartialContent, false},
		{"403 is handled by the non-2xx guard", http.StatusForbidden, false},
		{"404 is handled by the non-2xx guard", http.StatusNotFound, false},
		{"429 is handled by the non-2xx guard", http.StatusTooManyRequests, false},
		{"zero value", 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, batchStatusHasNoBody(tc.code))
		})
	}
}

// The two guards in batchGet must partition every status code: a non-2xx is an
// error, a body-less 2xx is an absence, and everything else is parsed. A code
// falling through both that has no body is exactly the 204 case that panicked.
func TestBatchStatusGuardsPartitionAllCodes(t *testing.T) {
	isFailure := func(code int32) bool { return code < 200 || code >= 300 }

	for code := int32(100); code < 600; code++ {
		failure := isFailure(code)
		noBody := batchStatusHasNoBody(code)
		assert.Falsef(t, failure && noBody,
			"status %d is classified as both a failure and a body-less success", code)

		if noBody {
			assert.Truef(t, code >= 200 && code < 300,
				"only a 2xx should reach the body-less branch, got %d", code)
		}
	}

	assert.True(t, batchStatusHasNoBody(http.StatusNoContent),
		"204 must be caught before the parse: it is a success, so the non-2xx guard lets it through, "+
			"and GetBatchResponseById panics on the absent body")
}
