// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignedIdentifiersToDicts(t *testing.T) {
	str := func(s string) *string { return &s }
	ts := func(s string) *time.Time {
		parsed, err := time.Parse(time.RFC3339, s)
		require.NoError(t, err)
		return &parsed
	}

	t.Run("no policies is an empty list, never nil", func(t *testing.T) {
		res := signedIdentifiersToDicts(nil)
		require.NotNil(t, res)
		assert.Empty(t, res)
	})

	t.Run("nil entries are skipped", func(t *testing.T) {
		res := signedIdentifiersToDicts([]*container.SignedIdentifier{nil, nil})
		assert.Empty(t, res)
	})

	// The keys have to match what the table resource already reports, or a
	// query that spans containers and tables reads different keys per row.
	t.Run("a full policy maps to the table resource's key set", func(t *testing.T) {
		res := signedIdentifiersToDicts([]*container.SignedIdentifier{{
			ID: str("readers"),
			AccessPolicy: &container.AccessPolicy{
				Permission: str("rl"),
				Start:      ts("2026-01-02T03:04:05Z"),
				Expiry:     ts("2027-01-02T03:04:05Z"),
			},
		}})
		require.Len(t, res, 1)
		assert.Equal(t, map[string]any{
			"id":         "readers",
			"permission": "rl",
			"startTime":  "2026-01-02T03:04:05Z",
			"expiryTime": "2027-01-02T03:04:05Z",
		}, res[0])
	})

	t.Run("a policy with no access policy still reports its id", func(t *testing.T) {
		res := signedIdentifiersToDicts([]*container.SignedIdentifier{{ID: str("orphan")}})
		require.Len(t, res, 1)
		assert.Equal(t, map[string]any{"id": "orphan"}, res[0])
	})

	t.Run("absent times are omitted, not zero-valued", func(t *testing.T) {
		res := signedIdentifiersToDicts([]*container.SignedIdentifier{{
			ID:           str("perm"),
			AccessPolicy: &container.AccessPolicy{Permission: str("r")},
		}})
		require.Len(t, res, 1)
		entry := res[0].(map[string]any)
		assert.NotContains(t, entry, "startTime")
		assert.NotContains(t, entry, "expiryTime")
	})
}

// The regression this guards: classifying a transport failure as "unreadable"
// would turn a network blip into a null field, and a policy that treats null as
// "nothing to check" would pass on data that was never read.
func TestIsBlobDataPlaneUnreadable(t *testing.T) {
	respErr := func(code int) error {
		return &azcore.ResponseError{StatusCode: code, RawResponse: &http.Response{StatusCode: code}}
	}

	for _, code := range []int{http.StatusForbidden, http.StatusUnauthorized, http.StatusNotFound} {
		assert.True(t, isBlobDataPlaneUnreadable(respErr(code)), "status %d", code)
	}

	for _, code := range []int{http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusServiceUnavailable} {
		assert.False(t, isBlobDataPlaneUnreadable(respErr(code)), "status %d", code)
	}

	assert.False(t, isBlobDataPlaneUnreadable(errors.New("connection reset by peer")))
	assert.False(t, isBlobDataPlaneUnreadable(nil))
}
