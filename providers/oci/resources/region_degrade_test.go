// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
)

// fakeServiceError implements the SDK's common.ServiceError interface so the
// classifier can be exercised without a live client.
type fakeServiceError struct {
	status int
	code   string
	msg    string
}

func (f fakeServiceError) GetHTTPStatusCode() int  { return f.status }
func (f fakeServiceError) GetMessage() string      { return f.msg }
func (f fakeServiceError) GetCode() string         { return f.code }
func (f fakeServiceError) GetOpcRequestID() string { return "req-1" }
func (f fakeServiceError) Error() string {
	return fmt.Sprintf("Service error:%s. %s. http status code: %d", f.code, f.msg, f.status)
}

func TestOciRegionServiceUnavailable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		// Only a 404 can mean "no endpoint for this service in this region".
		{"404 not found", fakeServiceError{status: 404, code: "NotAuthorizedOrNotFound"}, true},

		// 401/403 are credential and IAM-policy failures. They come back from
		// *every* region, so swallowing them reports an under-scoped token as
		// an empty tenancy.
		{"401 not authenticated", fakeServiceError{status: 401, code: "NotAuthenticated"}, false},
		{"403 not authorized", fakeServiceError{status: 403, code: "NotAuthorizedOrNotFound"}, false},
		{"429 too many requests", fakeServiceError{status: 429, code: "TooManyRequests"}, false},
		{"500 internal", fakeServiceError{status: 500, code: "InternalServerError"}, false},

		// A service error carries a real API response, so a message that merely
		// mentions a timeout must not be mistaken for a transport failure.
		{"service error whose message says timeout", fakeServiceError{
			status: 400, code: "LimitExceeded", msg: "request timeout budget exceeded",
		}, false},

		// Transport-level signatures: the service genuinely has no endpoint.
		{"dns error", &net.DNSError{Err: "no such host", Name: "svc.us-foo-1.oci.example", IsNotFound: true}, true},
		{"deadline exceeded", context.DeadlineExceeded, true},
		{"wrapped deadline exceeded", fmt.Errorf("get failed: %w", context.DeadlineExceeded), true},
		{"no such host in message", errors.New("dial tcp: lookup svc: no such host"), true},
		{"connection refused in message", errors.New("dial tcp 1.2.3.4:443: connection refused"), true},

		{"unrelated error", errors.New("boom"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ociRegionServiceUnavailable(tt.err))
		})
	}
}

func TestOciRunRegionPool(t *testing.T) {
	ok := func(items ...any) *jobpool.Job {
		return jobpool.NewJob(func() (jobpool.JobResult, error) {
			return jobpool.JobResult(items), nil
		})
	}
	boom := func() *jobpool.Job {
		return jobpool.NewJob(func() (jobpool.JobResult, error) {
			return nil, errors.New("region unavailable")
		})
	}

	t.Run("all regions succeed", func(t *testing.T) {
		res, err := ociRunRegionPool([]*jobpool.Job{ok("a"), ok("b", "c")})
		require.NoError(t, err)
		assert.ElementsMatch(t, []any{"a", "b", "c"}, res)
	})

	t.Run("a failing region does not discard the others", func(t *testing.T) {
		// The whole point of the change: one unentitled or undeployed region
		// must not sink a tenancy-wide query.
		res, err := ociRunRegionPool([]*jobpool.Job{ok("a"), boom(), ok("b")})
		require.NoError(t, err)
		assert.ElementsMatch(t, []any{"a", "b"}, res)
	})

	t.Run("every region failing is still an error", func(t *testing.T) {
		// Degrading to an empty list here would report a broken scan as a
		// clean tenancy.
		res, err := ociRunRegionPool([]*jobpool.Job{boom(), boom()})
		require.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("no jobs", func(t *testing.T) {
		res, err := ociRunRegionPool(nil)
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("jobErr job is treated as a failed region", func(t *testing.T) {
		res, err := ociRunRegionPool(jobErr(errors.New("bad region type")))
		require.Error(t, err)
		assert.Nil(t, res)
	})
}
