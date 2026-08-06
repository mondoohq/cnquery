// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
)

func TestOciCompartmentInaccessible(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		// OCI deliberately collapses "not found" and "not authorized" into one
		// response so callers cannot probe for resources they cannot see, so
		// both mean the same thing at compartment granularity.
		{"403 not authorized", fakeServiceError{status: 403, code: "NotAuthorizedOrNotFound"}, true},
		{"404 not found", fakeServiceError{status: 404, code: "NotAuthorizedOrNotFound"}, true},

		// A bad credential is tenancy-wide, not per-compartment, and must not
		// be mistaken for a compartment the caller merely lacks read on.
		{"401 not authenticated", fakeServiceError{status: 401, code: "NotAuthenticated"}, false},

		// Transient and server-side faults are real: skipping them per
		// compartment would silently drop resources that do exist.
		{"429 too many requests", fakeServiceError{status: 429, code: "TooManyRequests"}, false},
		{"500 internal", fakeServiceError{status: 500, code: "InternalServerError"}, false},
		{"400 bad request", fakeServiceError{status: 400, code: "InvalidParameter"}, false},

		// Transport errors carry no HTTP status and are not a compartment
		// permission signal.
		{"plain error", errors.New("boom"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ociCompartmentInaccessible(tt.err))
		})
	}
}

func TestOciCollectCompartmentJobs(t *testing.T) {
	ok := func(items ...any) *jobpool.Job {
		return jobpool.NewJob(func() (jobpool.JobResult, error) {
			return jobpool.JobResult(items), nil
		})
	}
	// A compartment the caller cannot read. Normal in OCI, where policies are
	// routinely scoped per compartment.
	denied := func() *jobpool.Job {
		return jobpool.NewJob(func() (jobpool.JobResult, error) {
			return nil, fakeServiceError{status: 403, code: "NotAuthorizedOrNotFound"}
		})
	}
	// A region where the service is not deployed at all.
	unavailable := func() *jobpool.Job {
		return jobpool.NewJob(func() (jobpool.JobResult, error) {
			return nil, fakeServiceError{status: 404, code: "NotAuthorizedOrNotFound"}
		})
	}
	throttled := func() *jobpool.Job {
		return jobpool.NewJob(func() (jobpool.JobResult, error) {
			return nil, fakeServiceError{status: 429, code: "TooManyRequests"}
		})
	}

	t.Run("all compartments succeed", func(t *testing.T) {
		res, err := ociCollectCompartmentJobs([]*jobpool.Job{ok("a"), ok("b", "c")})
		require.NoError(t, err)
		assert.ElementsMatch(t, []any{"a", "b", "c"}, res)
	})

	t.Run("a denied compartment does not discard the readable ones", func(t *testing.T) {
		// The whole point of the fan-out: a scanning principal with read on
		// some compartments is correctly configured, not broken.
		res, err := ociCollectCompartmentJobs([]*jobpool.Job{ok("a"), denied(), ok("b")})
		require.NoError(t, err)
		assert.ElementsMatch(t, []any{"a", "b"}, res)
	})

	t.Run("every compartment denied is an error, not an empty tenancy", func(t *testing.T) {
		// The failure this guard exists for: an under-scoped token must not
		// render as a clean scan of a tenancy with nothing in it.
		res, err := ociCollectCompartmentJobs([]*jobpool.Job{denied(), denied()})
		require.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "NotAuthorizedOrNotFound")
	})

	t.Run("a readable but empty compartment counts as success", func(t *testing.T) {
		// Distinguishes "read everything, found nothing" from "could not read".
		res, err := ociCollectCompartmentJobs([]*jobpool.Job{ok(), denied()})
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("an unavailable region is skipped without counting as denied", func(t *testing.T) {
		// A service with no endpoint in a region is an expected absence, so it
		// must not push an otherwise all-denied run into the success path.
		res, err := ociCollectCompartmentJobs([]*jobpool.Job{unavailable(), denied()})
		require.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("every region unavailable is an empty result, not an error", func(t *testing.T) {
		res, err := ociCollectCompartmentJobs([]*jobpool.Job{unavailable(), unavailable()})
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("a throttled compartment is reported", func(t *testing.T) {
		// Silently dropping a 429 would under-report resources that exist.
		res, err := ociCollectCompartmentJobs([]*jobpool.Job{ok("a"), throttled()})
		require.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("a hard error wins over partial success", func(t *testing.T) {
		res, err := ociCollectCompartmentJobs([]*jobpool.Job{ok("a"), denied(), throttled()})
		require.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("no jobs", func(t *testing.T) {
		res, err := ociCollectCompartmentJobs(nil)
		require.NoError(t, err)
		assert.Empty(t, res)
	})
}
