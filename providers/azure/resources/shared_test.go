// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"
)

func TestParseAzureTimestamp(t *testing.T) {
	str := func(s string) *string { return &s }

	t.Run("nil input", func(t *testing.T) {
		if got := parseAzureTimestamp(nil); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		if got := parseAzureTimestamp(str("")); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})

	t.Run("invalid string", func(t *testing.T) {
		if got := parseAzureTimestamp(str("not-a-timestamp")); got != nil {
			t.Fatalf("expected nil for unparseable input, got %v", got)
		}
	})

	t.Run("valid RFC 3339", func(t *testing.T) {
		got := parseAzureTimestamp(str("2023-04-05T12:34:56Z"))
		if got == nil {
			t.Fatal("expected a parsed timestamp, got nil")
		}
		want := time.Date(2023, 4, 5, 12, 34, 56, 0, time.UTC)
		if !got.Equal(want) {
			t.Fatalf("expected %v, got %v", want, *got)
		}
	})

	t.Run("valid RFC 3339 with offset", func(t *testing.T) {
		got := parseAzureTimestamp(str("2023-04-05T12:34:56+02:00"))
		if got == nil {
			t.Fatal("expected a parsed timestamp, got nil")
		}
		want := time.Date(2023, 4, 5, 10, 34, 56, 0, time.UTC)
		if !got.Equal(want) {
			t.Fatalf("expected %v (UTC), got %v", want, got.UTC())
		}
	})
}
