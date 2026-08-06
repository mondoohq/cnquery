// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== status typed-ref null-state =====

func TestApplicationStatusInstanceNullWhenNoInstanceId(t *testing.T) {
	s := &mqlAwsEc2ApplicationStatusCheckStatus{}
	s.cacheInstanceId = ""
	got, err := s.instance()
	require.NoError(t, err)
	assert.Nil(t, got)
	assert.True(t, s.Instance.IsNull())
	assert.True(t, s.Instance.IsSet())
}

// ===== instance arn is built, not returned by the API =====

// The status only carries an instance id, and initAwsEc2Instance resolves by
// arn, so the arn is assembled from the region and account. A drift in the
// pattern would resolve to nothing rather than fail loudly.
func TestEc2InstanceArnPattern(t *testing.T) {
	got := fmt.Sprintf(ec2InstanceArnPattern, "us-east-1", "123456789012", "i-1234567890abcdef0")
	assert.Equal(t, "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0", got)
}
