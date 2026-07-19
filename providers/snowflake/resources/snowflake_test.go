// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"reflect"
	"testing"
	"time"

	"go.mondoo.com/mql/v13/types"
)

func TestSplitCommaList(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []any
	}{
		{"empty string", "", []any{}},
		{"whitespace only", "   ", []any{}},
		{"single value", "s3://bucket/", []any{"s3://bucket/"}},
		{"two values", "s3://a/,s3://b/", []any{"s3://a/", "s3://b/"}},
		{"values with spaces", "s3://a/, s3://b/", []any{"s3://a/", "s3://b/"}},
		{"bracket wrapped", "[s3://a/,s3://b/]", []any{"s3://a/", "s3://b/"}},
		{"trailing comma dropped", "s3://a/,", []any{"s3://a/"}},
		{"empty entries dropped", "s3://a/,,s3://b/", []any{"s3://a/", "s3://b/"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := splitCommaList(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("splitCommaList(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseSnowflakeTime(t *testing.T) {
	// null cases: empty and unparseable strings must resolve to null, never error.
	nullCases := []struct {
		name string
		in   string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"unrecognized format", "not-a-timestamp"},
		{"date only", "2023-01-02"},
	}
	for _, tc := range nullCases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseSnowflakeTime(tc.in)
			if got.Type != types.Nil {
				t.Errorf("parseSnowflakeTime(%q) type = %v, want Nil", tc.in, got.Type)
			}
		})
	}

	// valid cases: each supported layout must parse to the same instant.
	want := time.Date(2023, 1, 2, 15, 4, 5, 0, time.FixedZone("", -7*3600))
	validCases := []struct {
		name string
		in   string
	}{
		{"snowflake show format", "2023-01-02 15:04:05.000 -0700"},
		{"snowflake show format nanos", "2023-01-02 15:04:05.000000000 -0700"},
		{"snowflake show format no fraction", "2023-01-02 15:04:05 -0700"},
		{"rfc3339", "2023-01-02T15:04:05-07:00"},
	}
	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseSnowflakeTime(tc.in)
			if got.Type != types.Time {
				t.Fatalf("parseSnowflakeTime(%q) type = %v, want Time", tc.in, got.Type)
			}
			gotTime, ok := got.Value.(*time.Time)
			if !ok {
				t.Fatalf("parseSnowflakeTime(%q) value is %T, want *time.Time", tc.in, got.Value)
			}
			if !gotTime.Equal(want) {
				t.Errorf("parseSnowflakeTime(%q) = %v, want %v", tc.in, gotTime, want)
			}
		})
	}
}
