// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

func TestEpochMsTime(t *testing.T) {
	// A known epoch-ms: 2021-01-01T00:00:00Z = 1609459200000 ms.
	const knownMs int64 = 1609459200000

	tests := []struct {
		name string
		ms   int64
		// wantNil true means the function should return a nil *time.Time.
		wantNil bool
	}{
		{name: "zero is unset sentinel", ms: 0, wantNil: true},
		// NOTE: the guard is `ms <= 0`, so any negative value (not just -1)
		// maps to nil.
		{name: "negative sentinel -1", ms: -1, wantNil: true},
		{name: "large negative", ms: -1609459200000, wantNil: true},
		{name: "known positive epoch-ms", ms: knownMs, wantNil: false},
		{name: "one millisecond", ms: 1, wantNil: false},
		{name: "large positive", ms: 4102444800000, wantNil: false}, // 2100-01-01T00:00:00Z
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := epochMsTime(tc.ms)
			if tc.wantNil {
				if got != nil {
					t.Fatalf("epochMsTime(%d) = %v, want nil", tc.ms, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("epochMsTime(%d) = nil, want non-nil", tc.ms)
			}
			// The result must equal time.UnixMilli of the same input.
			want := time.UnixMilli(tc.ms)
			if !got.Equal(want) {
				t.Fatalf("epochMsTime(%d) = %v, want %v", tc.ms, got, want)
			}
		})
	}

	// Spot-check the actual UTC calendar time for the known epoch-ms.
	got := epochMsTime(knownMs)
	if got == nil {
		t.Fatal("epochMsTime(knownMs) = nil, want non-nil")
	}
	wantUTC := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	if !got.UTC().Equal(wantUTC) {
		t.Fatalf("epochMsTime(%d).UTC() = %v, want %v", knownMs, got.UTC(), wantUTC)
	}
}

func TestEpochMsRFC3339(t *testing.T) {
	tests := []struct {
		name string
		ms   int64
		want string
	}{
		{name: "zero is unset sentinel", ms: 0, want: ""},
		// NOTE: the guard is `ms <= 0`, so negatives render as "" too.
		{name: "negative sentinel -1", ms: -1, want: ""},
		{name: "known positive epoch-ms", ms: 1609459200000, want: "2021-01-01T00:00:00Z"},
		{name: "one millisecond after epoch", ms: 1, want: "1970-01-01T00:00:00Z"},
		{name: "large positive", ms: 4102444800000, want: "2100-01-01T00:00:00Z"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := epochMsRFC3339(tc.ms)
			if got != tc.want {
				t.Fatalf("epochMsRFC3339(%d) = %q, want %q", tc.ms, got, tc.want)
			}
		})
	}
}

func TestComplexValueIds(t *testing.T) {
	tests := []struct {
		name string
		vals []iam.ComplexValue
		want []string
	}{
		{
			name: "nil slice yields empty non-nil slice",
			vals: nil,
			want: []string{},
		},
		{
			name: "empty slice yields empty non-nil slice",
			vals: []iam.ComplexValue{},
			want: []string{},
		},
		{
			name: "entries with values",
			vals: []iam.ComplexValue{{Value: "1"}, {Value: "2"}},
			want: []string{"1", "2"},
		},
		{
			name: "empty value is skipped",
			vals: []iam.ComplexValue{{Value: ""}},
			want: []string{},
		},
		{
			name: "mixed populated and empty values",
			vals: []iam.ComplexValue{
				{Value: "grp-1", Display: "Group 1"},
				{Value: ""},
				{Value: "grp-2"},
			},
			want: []string{"grp-1", "grp-2"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := complexValueIds(tc.vals)
			// complexValueIds always returns a non-nil slice (make with cap).
			if got == nil {
				t.Fatalf("complexValueIds(%v) = nil, want non-nil", tc.vals)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("complexValueIds(%v) = %v, want %v", tc.vals, got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("complexValueIds(%v)[%d] = %q, want %q", tc.vals, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestSseEncryption(t *testing.T) {
	tests := []struct {
		name          string
		ed            *catalog.EncryptionDetails
		wantAlgorithm string
		wantKmsArn    string
	}{
		{
			name:          "nil pointer",
			ed:            nil,
			wantAlgorithm: "",
			wantKmsArn:    "",
		},
		{
			name:          "non-nil with nil SseEncryptionDetails",
			ed:            &catalog.EncryptionDetails{SseEncryptionDetails: nil},
			wantAlgorithm: "",
			wantKmsArn:    "",
		},
		{
			name: "fully populated SSE-KMS",
			ed: &catalog.EncryptionDetails{
				SseEncryptionDetails: &catalog.SseEncryptionDetails{
					Algorithm:    catalog.SseEncryptionDetailsAlgorithmAwsSseKms,
					AwsKmsKeyArn: "arn:aws:kms:us-east-1:123456789012:key/abc",
				},
			},
			wantAlgorithm: "AWS_SSE_KMS",
			wantKmsArn:    "arn:aws:kms:us-east-1:123456789012:key/abc",
		},
		{
			name: "SSE-S3 with no KMS ARN",
			ed: &catalog.EncryptionDetails{
				SseEncryptionDetails: &catalog.SseEncryptionDetails{
					Algorithm: catalog.SseEncryptionDetailsAlgorithmAwsSseS3,
				},
			},
			wantAlgorithm: "AWS_SSE_S3",
			wantKmsArn:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			algo, arn := sseEncryption(tc.ed)
			if algo != tc.wantAlgorithm {
				t.Fatalf("sseEncryption() algorithm = %q, want %q", algo, tc.wantAlgorithm)
			}
			if arn != tc.wantKmsArn {
				t.Fatalf("sseEncryption() kmsKeyArn = %q, want %q", arn, tc.wantKmsArn)
			}
		})
	}
}
