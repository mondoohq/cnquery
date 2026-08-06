// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identitydomains"
	"github.com/stretchr/testify/assert"
)

func TestOciScimNextIndex(t *testing.T) {
	tests := []struct {
		name       string
		startIndex int
		returned   int
		total      *int
		wantNext   int
		wantMore   bool
	}{
		// SCIM indexes are 1-based, so the second page of a 200-item page
		// size starts at 201, not 200.
		{"first full page of many", 1, ociScimPageSize, common.Int(500), ociScimPageSize + 1, true},
		{"second full page of many", 201, ociScimPageSize, common.Int(500), 401, true},

		// next > total ends the loop.
		{"last partial page", 401, 100, common.Int(500), 0, false},
		{"exactly consumed", 1, 200, common.Int(200), 0, false},

		// The guard that matters: an empty page always terminates. A service
		// reporting a total larger than it will actually return would
		// otherwise spin forever.
		{"empty page with a larger total", 201, 0, common.Int(10000), 0, false},
		{"empty page with no total", 1, 0, nil, 0, false},
		{"empty first page", 1, 0, common.Int(0), 0, false},

		// With no total reported, only a full page implies more to come.
		{"no total, full page", 1, ociScimPageSize, nil, ociScimPageSize + 1, true},
		{"no total, short page", 1, 5, nil, 0, false},

		// A single item in a single-item collection must not loop.
		{"one of one", 1, 1, common.Int(1), 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, more := ociScimNextIndex(tt.startIndex, tt.returned, tt.total)
			assert.Equal(t, tt.wantMore, more, "more")
			if tt.wantMore {
				assert.Equal(t, tt.wantNext, next, "next")
			}
		})
	}
}

// TestOciScimNextIndexTerminates walks the loop the way the listers do and
// asserts it both terminates and visits every item exactly once.
func TestOciScimNextIndexTerminates(t *testing.T) {
	for _, total := range []int{0, 1, 199, 200, 201, 999, 1000} {
		t.Run(map[bool]string{true: "many", false: "few"}[total > 200], func(t *testing.T) {
			startIndex := 1
			seen := 0
			for iterations := 0; ; iterations++ {
				if iterations > 100 {
					t.Fatalf("pagination did not terminate for total=%d", total)
				}

				remaining := total - seen
				returned := min(remaining, ociScimPageSize)
				seen += returned

				next, more := ociScimNextIndex(startIndex, returned, &total)
				if !more {
					break
				}
				startIndex = next
			}
			assert.Equal(t, total, seen, "every item visited exactly once for total=%d", total)
		})
	}
}

func TestOciPrimaryEmail(t *testing.T) {
	t.Run("no emails", func(t *testing.T) {
		value, verified := ociPrimaryEmail(nil)
		assert.Empty(t, value)
		assert.False(t, verified)
	})

	t.Run("picks the primary, not the first", func(t *testing.T) {
		value, verified := ociPrimaryEmail([]identitydomains.UserEmails{
			{Value: common.String("alt@example.com"), Verified: common.Bool(true)},
			{Value: common.String("primary@example.com"), Primary: common.Bool(true), Verified: common.Bool(true)},
		})
		assert.Equal(t, "primary@example.com", value)
		assert.True(t, verified)
	})

	t.Run("falls back to the first when none is primary", func(t *testing.T) {
		value, verified := ociPrimaryEmail([]identitydomains.UserEmails{
			{Value: common.String("only@example.com"), Verified: common.Bool(false)},
		})
		assert.Equal(t, "only@example.com", value)
		assert.False(t, verified)
	})

	t.Run("an unverified primary reports unverified", func(t *testing.T) {
		// The verified flag has to follow the address that was selected,
		// not the first verified address in the list.
		value, verified := ociPrimaryEmail([]identitydomains.UserEmails{
			{Value: common.String("alt@example.com"), Verified: common.Bool(true)},
			{Value: common.String("primary@example.com"), Primary: common.Bool(true)},
		})
		assert.Equal(t, "primary@example.com", value)
		assert.False(t, verified)
	})
}
