// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	nethttp "net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
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

// s3HTTPError builds an S3-style error carrying an HTTP status code, mirroring
// what the SDK surfaces for GetBucketLogging / GetBucketPolicyStatus etc.
func s3HTTPError(code int, apiErr error) error {
	return &smithyhttp.ResponseError{
		Response: &smithyhttp.Response{Response: &nethttp.Response{StatusCode: code}},
		Err:      apiErr,
	}
}

// TestIsS3BucketInaccessible guards the cross-account regression: a CloudTrail
// organization trail references a log bucket owned by the management account, so
// a member-account scan must treat the resulting 403 (and a deleted-bucket 404)
// as "no data" rather than failing the check, while genuine errors still surface.
func TestIsS3BucketInaccessible(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"cross-account 403 AccessDenied", s3HTTPError(403, &smithy.GenericAPIError{Code: "AccessDenied", Message: "Access Denied"}), true},
		{"bare 403 forbidden", s3HTTPError(403, &smithy.GenericAPIError{Code: "Forbidden"}), true},
		{"deleted bucket 404", s3HTTPError(404, &smithy.GenericAPIError{Code: "NoSuchBucket"}), true},
		{"typed NotFound", &s3types.NotFound{}, true},
		{"transient 500", s3HTTPError(500, &smithy.GenericAPIError{Code: "InternalError"}), false},
		{"unrelated error", errors.New("boom"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isS3BucketInaccessible(tt.err))
		})
	}
}
