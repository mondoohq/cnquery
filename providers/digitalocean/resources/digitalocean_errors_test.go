// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"net/http"
	"testing"

	"github.com/aws/smithy-go"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestIsDoNotFound(t *testing.T) {
	t.Run("nil is not a 404", func(t *testing.T) {
		assert.False(t, isDoNotFound(nil))
	})

	t.Run("non-godo error is not a 404", func(t *testing.T) {
		// Plain errors should not be misclassified as "not configured".
		// Otherwise transient failures would be silently swallowed.
		assert.False(t, isDoNotFound(errors.New("boom")))
	})

	t.Run("godo 404 is a 404", func(t *testing.T) {
		er := &godo.ErrorResponse{
			Response: &http.Response{StatusCode: http.StatusNotFound},
			Message:  "not found",
		}
		assert.True(t, isDoNotFound(er))
	})

	t.Run("godo 500 is not a 404", func(t *testing.T) {
		// Regression guard for the original bug: registryRepositories
		// used to return empty-list on ANY error, masking 5xx as
		// "no registry configured".
		er := &godo.ErrorResponse{
			Response: &http.Response{StatusCode: http.StatusInternalServerError},
			Message:  "internal error",
		}
		assert.False(t, isDoNotFound(er))
	})

	t.Run("godo 401 is not a 404", func(t *testing.T) {
		er := &godo.ErrorResponse{
			Response: &http.Response{StatusCode: http.StatusUnauthorized},
			Message:  "unauthorized",
		}
		assert.False(t, isDoNotFound(er))
	})

	t.Run("godo error with nil Response is not a 404", func(t *testing.T) {
		// Pre-flight errors (e.g., network failure) carry no HTTP
		// response; treat as "not a 404" so callers propagate them.
		er := &godo.ErrorResponse{Message: "network error"}
		assert.False(t, isDoNotFound(er))
	})
}

// apiErr is a tiny smithy.APIError stub for testing the S3-error
// classifiers in digitalocean_spaces.go.
type apiErr struct{ code, msg string }

func (e apiErr) Error() string                 { return e.code + ": " + e.msg }
func (e apiErr) ErrorCode() string             { return e.code }
func (e apiErr) ErrorMessage() string          { return e.msg }
func (e apiErr) ErrorFault() smithy.ErrorFault { return smithy.FaultClient }

func TestIsNoSuchConfiguration(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"NoSuchBucketPolicy typed", apiErr{code: "NoSuchBucketPolicy"}, true},
		{"NoSuchPublicAccessBlockConfiguration typed", apiErr{code: "NoSuchPublicAccessBlockConfiguration"}, true},
		{"ServerSideEncryptionConfigurationNotFoundError typed", apiErr{code: "ServerSideEncryptionConfigurationNotFoundError"}, true},
		{"NoSuchCORSConfiguration typed", apiErr{code: "NoSuchCORSConfiguration"}, true},
		{"NoSuchLifecycleConfiguration typed", apiErr{code: "NoSuchLifecycleConfiguration"}, true},
		// DO Spaces sometimes surfaces the code only in the message body.
		{"code in plain error string", errors.New("server returned: NoSuchBucketPolicy"), true},
		{"unrelated typed code", apiErr{code: "AccessDenied"}, false},
		{"unrelated error string", errors.New("rate limited"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isNoSuchConfiguration(tc.err))
		})
	}
}

func TestIsAccessDenied(t *testing.T) {
	t.Run("nil is not access denied", func(t *testing.T) {
		assert.False(t, isAccessDenied(nil))
	})
	t.Run("typed AccessDenied code", func(t *testing.T) {
		assert.True(t, isAccessDenied(apiErr{code: "AccessDenied"}))
	})
	t.Run("substring fallback", func(t *testing.T) {
		assert.True(t, isAccessDenied(errors.New("AccessDenied: not allowed")))
	})
	t.Run("unrelated code", func(t *testing.T) {
		assert.False(t, isAccessDenied(apiErr{code: "NoSuchKey"}))
	})
}
