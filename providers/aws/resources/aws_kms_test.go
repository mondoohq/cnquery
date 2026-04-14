// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeKmsKeyRef(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		region    string
		accountId string
		wantARN   string
		wantErr   string
	}{
		{
			name:      "full key ARN is returned as-is",
			input:     "arn:aws:kms:us-east-1:123456789012:key/7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2",
			region:    "us-west-2",
			accountId: "999999999999",
			wantARN:   "arn:aws:kms:us-east-1:123456789012:key/7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2",
		},
		{
			name:      "alias ARN is returned as-is",
			input:     "arn:aws:kms:us-east-1:123456789012:alias/my-key",
			region:    "us-east-1",
			accountId: "123456789012",
			wantARN:   "arn:aws:kms:us-east-1:123456789012:alias/my-key",
		},
		{
			name:      "single-region key ID is normalized to an ARN",
			input:     "7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2",
			region:    "us-east-1",
			accountId: "123456789012",
			wantARN:   "arn:aws:kms:us-east-1:123456789012:key/7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2",
		},
		{
			name:      "multi-region key ID is normalized to an ARN",
			input:     "mrk-1234abcd12ab34cd56ef1234567890ab",
			region:    "us-east-1",
			accountId: "123456789012",
			wantARN:   "arn:aws:kms:us-east-1:123456789012:key/mrk-1234abcd12ab34cd56ef1234567890ab",
		},
		{
			name:      "govcloud key ID uses govcloud partition",
			input:     "7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2",
			region:    "us-gov-west-1",
			accountId: "123456789012",
			wantARN:   "arn:aws-us-gov:kms:us-gov-west-1:123456789012:key/7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2",
		},
		{
			name:      "china key ID uses china partition",
			input:     "7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2",
			region:    "cn-north-1",
			accountId: "123456789012",
			wantARN:   "arn:aws-cn:kms:cn-north-1:123456789012:key/7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2",
		},
		{
			name:      "bare key ID with empty region returns error",
			input:     "7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2",
			region:    "",
			accountId: "123456789012",
			wantErr:   "cannot normalize KMS key ID",
		},
		{
			name:      "bare key ID with empty account returns error",
			input:     "7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2",
			region:    "us-east-1",
			accountId: "",
			wantErr:   "cannot normalize KMS key ID",
		},
		{
			name:      "non-KMS ARN returns error",
			input:     "arn:aws:s3:::my-bucket",
			region:    "us-east-1",
			accountId: "123456789012",
			wantErr:   "expected a KMS key or alias ARN",
		},
		{
			name:      "alias name requires DescribeKey resolution",
			input:     "alias/my-key",
			region:    "us-east-1",
			accountId: "123456789012",
			wantErr:   "invalid KMS key reference",
		},
		{
			name:      "invalid input returns error",
			input:     "not-a-valid-key-ref",
			region:    "us-east-1",
			accountId: "123456789012",
			wantErr:   "invalid KMS key reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeKmsKeyRef(tt.input, tt.region, tt.accountId)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantARN, got.String())
		})
	}
}

func TestIsKmsKeyID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "single-region key ID", input: "7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2", want: true},
		{name: "multi-region key ID", input: "mrk-1234abcd12ab34cd56ef1234567890ab", want: true},
		{name: "alias name", input: "alias/my-key", want: false},
		{name: "garbage", input: "not-a-key", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isKmsKeyID(tt.input))
		})
	}
}

func TestExtractKmsKeyId(t *testing.T) {
	assert.Equal(t, "7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2", extractKmsKeyId("key/7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2"))
	assert.Equal(t, "mrk-1234abcd12ab34cd56ef1234567890ab", extractKmsKeyId("key/mrk-1234abcd12ab34cd56ef1234567890ab"))
	assert.Equal(t, "alias/my-key", extractKmsKeyId("alias/my-key"))
}
