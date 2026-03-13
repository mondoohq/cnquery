// Copyright (c) Mondoo, Inc.
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
			name:      "full ARN is returned as-is",
			input:     "arn:aws:kms:us-east-1:123456789012:key/7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2",
			region:    "us-west-2",
			accountId: "999999999999",
			wantARN:   "arn:aws:kms:us-east-1:123456789012:key/7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2",
		},
		{
			name:      "bare UUID is normalized to ARN",
			input:     "7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2",
			region:    "us-east-1",
			accountId: "123456789012",
			wantARN:   "arn:aws:kms:us-east-1:123456789012:key/7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2",
		},
		{
			name:      "bare UUID with empty region returns error",
			input:     "7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2",
			region:    "",
			accountId: "123456789012",
			wantErr:   "cannot normalize KMS key UUID",
		},
		{
			name:      "invalid input returns error",
			input:     "not-a-valid-key-ref",
			region:    "us-east-1",
			accountId: "123456789012",
			wantErr:   "invalid KMS key reference",
		},
		{
			name:      "alias ARN is returned as-is",
			input:     "arn:aws:kms:us-east-1:123456789012:alias/my-key",
			region:    "us-east-1",
			accountId: "123456789012",
			wantARN:   "arn:aws:kms:us-east-1:123456789012:alias/my-key",
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
