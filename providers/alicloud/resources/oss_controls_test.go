// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"testing"

	oss "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/stretchr/testify/assert"
)

// TestOssConfigAbsent covers the classifier that decides whether a failed
// per-bucket configuration read means "not configured" or "the read failed".
//
// The negative cases carry the weight. Folding a 403 or a transport failure
// into "not configured" would turn an unread bucket into one reporting no CORS
// rules, no retention policy and no TLS floor, which is the reading an audit
// would pass on.
func TestOssConfigAbsent(t *testing.T) {
	svcErr := func(status int, code string) error {
		return &oss.ServiceError{StatusCode: status, Code: code}
	}

	t.Run("404 means not configured", func(t *testing.T) {
		assert.True(t, ossConfigAbsent(svcErr(404, "NoSuchCORSConfiguration")))
	})
	t.Run("404 for a missing retention policy", func(t *testing.T) {
		assert.True(t, ossConfigAbsent(svcErr(404, "NoSuchWORMConfiguration")))
	})
	t.Run("wrapped errors are still matched", func(t *testing.T) {
		assert.True(t, ossConfigAbsent(fmt.Errorf("get bucket cors: %w", svcErr(404, "NoSuchCORSConfiguration"))))
	})
	t.Run("403 is a real failure", func(t *testing.T) {
		assert.False(t, ossConfigAbsent(svcErr(403, "AccessDenied")))
	})
	t.Run("500 is a real failure", func(t *testing.T) {
		assert.False(t, ossConfigAbsent(svcErr(500, "InternalError")))
	})
	t.Run("transport error is a real failure", func(t *testing.T) {
		assert.False(t, ossConfigAbsent(errors.New("dial tcp: i/o timeout")))
	})
}

// TestOssStrings covers the string-list flattening. A blank entry must be
// dropped rather than passed through, since a blank in a referer allow list or
// a CORS origin list reads as a configured value.
func TestOssStrings(t *testing.T) {
	assert.Equal(t, []any{}, ossStrings(nil))
	assert.Equal(t, []any{}, ossStrings([]string{"", "   "}))
	assert.Equal(t, []any{"*"}, ossStrings([]string{"*", ""}))
	assert.Equal(t, []any{"TLSv1.2", "TLSv1.3"}, ossStrings([]string{"TLSv1.2", "", "TLSv1.3"}))
}

// TestOssPolicyIsPublic covers how the two policy verdicts combine. The service
// verdict wins whenever it could be read; the parsed fallback only applies when
// it could not, so a credential that can read the policy document but not the
// policy status still gets an answer rather than a silent false.
func TestOssPolicyIsPublic(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	tests := []struct {
		name    string
		service *bool
		parsed  bool
		want    bool
	}{
		{"service says public", boolPtr(true), false, true},
		{"service says private and overrides a parsed public", boolPtr(false), true, false},
		{"service unavailable, parser says public", nil, true, true},
		{"service unavailable, parser says private", nil, false, false},
		{"both agree public", boolPtr(true), true, true},
		{"both agree private", boolPtr(false), false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ossPolicyIsPublic(tt.service, tt.parsed))
		})
	}
}
