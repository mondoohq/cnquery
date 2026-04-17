// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	batch_types "github.com/aws/aws-sdk-go-v2/service/batch/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchContainerNullState(t *testing.T) {
	t.Run("nil containerProperties sets null state", func(t *testing.T) {
		jd := &mqlAwsBatchJobDefinition{}
		// cacheContainerProperties is nil by default
		result, err := jd.container()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, jd.Container.IsNull())
		assert.True(t, jd.Container.IsSet())
	})
}

func TestBatchRetryNullState(t *testing.T) {
	t.Run("nil retryStrategy sets null state", func(t *testing.T) {
		jd := &mqlAwsBatchJobDefinition{}
		result, err := jd.retry()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, jd.Retry.IsNull())
		assert.True(t, jd.Retry.IsSet())
	})
}

func TestBatchJobTimeoutNullState(t *testing.T) {
	t.Run("nil timeout sets null state", func(t *testing.T) {
		jd := &mqlAwsBatchJobDefinition{}
		result, err := jd.jobTimeout()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, jd.JobTimeout.IsNull())
		assert.True(t, jd.JobTimeout.IsSet())
	})
}

func TestBatchContainerPropertiesDict(t *testing.T) {
	t.Run("nil cache returns nil dict", func(t *testing.T) {
		jd := &mqlAwsBatchJobDefinition{}
		result, err := jd.containerProperties()
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("non-nil cache returns dict", func(t *testing.T) {
		jd := &mqlAwsBatchJobDefinition{}
		jd.cacheContainerProperties = &batch_types.ContainerProperties{
			Image:   aws.String("alpine:latest"),
			Command: []string{"echo", "hello"},
		}
		result, err := jd.containerProperties()
		require.NoError(t, err)
		require.NotNil(t, result)
		dict, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "alpine:latest", dict["Image"])
	})
}

func TestBatchRetryStrategyDict(t *testing.T) {
	t.Run("nil cache returns nil dict", func(t *testing.T) {
		jd := &mqlAwsBatchJobDefinition{}
		result, err := jd.retryStrategy()
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("non-nil cache returns dict", func(t *testing.T) {
		jd := &mqlAwsBatchJobDefinition{}
		attempts := int32(3)
		jd.cacheRetryStrategy = &batch_types.RetryStrategy{
			Attempts: &attempts,
		}
		result, err := jd.retryStrategy()
		require.NoError(t, err)
		require.NotNil(t, result)
		dict, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(3), dict["Attempts"])
	})
}

func TestBatchTimeoutDict(t *testing.T) {
	t.Run("nil cache returns nil dict", func(t *testing.T) {
		jd := &mqlAwsBatchJobDefinition{}
		result, err := jd.timeout()
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("non-nil cache returns dict", func(t *testing.T) {
		jd := &mqlAwsBatchJobDefinition{}
		dur := int32(600)
		jd.cacheTimeout = &batch_types.JobTimeout{
			AttemptDurationSeconds: &dur,
		}
		result, err := jd.timeout()
		require.NoError(t, err)
		require.NotNil(t, result)
		dict, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(600), dict["AttemptDurationSeconds"])
	})
}

func TestTimeFromBatchMillis(t *testing.T) {
	// Batch API returns *int64 milliseconds. nil and 0 must produce nil so
	// runtime applies StateIsNull (avoids surfacing fabricated 1970 timestamps).
	cases := []struct {
		name string
		in   *int64
		want *int64 // nil means expect nil *time.Time, non-nil is unix-milli
	}{
		{name: "nil", in: nil, want: nil},
		{name: "zero", in: ptrInt64(0), want: nil},
		{name: "positive", in: ptrInt64(1_700_000_000_000), want: ptrInt64(1_700_000_000_000)},
		{name: "small positive", in: ptrInt64(1), want: ptrInt64(1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := timeFromBatchMillis(tc.in)
			if tc.want == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, *tc.want, got.UnixMilli())
		})
	}
}

func TestBatchChildID(t *testing.T) {
	// batchChildID is the shared __id builder for sibling children where the
	// natural key (Name, JobId, etc.) may be empty. Unnamed siblings must never
	// collide — that was the class of bug fixed in `buildBatchSecrets` and
	// `buildBatchEksContainers`.
	t.Run("uses name when present", func(t *testing.T) {
		assert.Equal(t, "arn:aws:batch:us-west-2::job/abc/dep/xyz",
			batchChildID("arn:aws:batch:us-west-2::job/abc/dep", "xyz", 0))
	})

	t.Run("uses indexed fallback when name is empty", func(t *testing.T) {
		assert.Equal(t, "arn:aws:batch:us-west-2::job/abc/dep/#7",
			batchChildID("arn:aws:batch:us-west-2::job/abc/dep", "", 7))
	})

	t.Run("large indices stay ASCII (regression for rune('0'+i) overflow)", func(t *testing.T) {
		// The previous implementation used string(rune('0'+i)), which wrapped to
		// ':' at i=10 and non-digit ASCII beyond that, colliding trivially.
		// Assert every index through 64 produces a distinct ASCII-safe id.
		const base = "parent/kind"
		seen := make(map[string]struct{}, 64)
		for i := 0; i < 64; i++ {
			id := batchChildID(base, "", i)
			_, dup := seen[id]
			require.False(t, dup, "duplicate __id for index %d: %q", i, id)
			seen[id] = struct{}{}
		}
	})

	t.Run("many anonymous siblings are all distinct", func(t *testing.T) {
		// Direct regression for the eksContainer fix — multiple unnamed
		// containers in the same pod must get distinct __id values.
		const n = 50
		const base = "pod/container"
		ids := make(map[string]struct{}, n)
		for i := 0; i < n; i++ {
			ids[batchChildID(base, "", i)] = struct{}{}
		}
		assert.Len(t, ids, n)
	})
}

func ptrInt64(v int64) *int64 { return &v }
