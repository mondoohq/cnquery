// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogpushJobs(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/logpush/jobs", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("logpush_jobs"))
	})

	result, err := zone.logpushJobs()
	require.NoError(t, err)
	require.Len(t, result, 1)

	job := result[0].(*mqlCloudflareZoneLogpushJob)
	// The cache key must embed the zone: id() reads a zoneID set only after
	// NewResource returns, so the fix passes __id explicitly. Without it the
	// key was `logpush@@42` (empty zone) and jobs with the same numeric id in
	// different zones would alias.
	assert.Contains(t, job.MqlID(), testZoneID, "logpush job __id must be zone-scoped")
	assert.Equal(t, int64(42), job.Id.Data)
	assert.Equal(t, "HTTP Requests to S3", job.Name.Data)
	assert.Equal(t, "http_requests", job.Dataset.Data)
	assert.True(t, job.Enabled.Data)
	assert.Equal(t, "high", job.Frequency.Data)
	assert.Equal(t, "s3://mybucket/logs?region=us-east-1", job.DestinationConf.Data)
	assert.Equal(t, "", job.ErrorMessage.Data)
	assert.False(t, job.LastComplete.Data.IsZero())
}
