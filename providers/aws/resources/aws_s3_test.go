// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/stretchr/testify/assert"
)

func TestS3BucketArnValidation(t *testing.T) {
	tests := []struct {
		name   string
		arnStr string
		valid  bool
	}{
		{"standard partition", "arn:aws:s3:::my-bucket", true},
		{"govcloud partition", "arn:aws-us-gov:s3:::my-bucket", true},
		{"china partition", "arn:aws-cn:s3:::my-bucket", true},
		{"wrong service", "arn:aws:ec2:us-east-1:123456789012:instance/i-1234", false},
		{"not an ARN", "not-an-arn", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := arn.Parse(tt.arnStr)
			isValidS3 := err == nil && parsed.Service == "s3"
			assert.Equal(t, tt.valid, isValidS3)
		})
	}
}
