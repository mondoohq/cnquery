// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	cloudwatchlogstypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLogGroupArn(t *testing.T) {
	tests := []struct {
		name         string
		arn          string
		expectRegion string
		expectGroup  string
	}{
		{
			"standard log group",
			"arn:aws:logs:us-east-1:123456789012:log-group:/my/log/group:*",
			"us-east-1",
			"/my/log/group",
		},
		{
			"log group name with colons",
			"arn:aws:logs:eu-west-1:123456789012:log-group:my:group:with:colons:*",
			"eu-west-1",
			"my:group:with:colons",
		},
		{
			"simple name",
			"arn:aws:logs:ap-southeast-1:999999999999:log-group:simple:*",
			"ap-southeast-1",
			"simple",
		},
		{
			"govcloud partition",
			"arn:aws-us-gov:logs:us-gov-west-1:123456789012:log-group:/gov/logs:*",
			"us-gov-west-1",
			"/gov/logs",
		},
		{
			"china partition",
			"arn:aws-cn:logs:cn-north-1:123456789012:log-group:/cn/app:*",
			"cn-north-1",
			"/cn/app",
		},
		{
			"malformed ARN - too few parts",
			"arn:aws:logs",
			"",
			"",
		},
		{
			"empty string",
			"",
			"",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			region, groupName := parseLogGroupArn(tt.arn)
			assert.Equal(t, tt.expectRegion, region)
			assert.Equal(t, tt.expectGroup, groupName)
		})
	}
}

// TestBuildLogGroupResourceRetention pins how a log group reports its
// retention. CloudWatch signals "never expire" by returning no retention at
// all, so defaulting that to 0 put the strongest retention at the bottom of
// any numeric comparison: `retentionInDays >= 365` excluded exactly the groups
// that keep their logs forever.
func TestBuildLogGroupResourceRetention(t *testing.T) {
	t.Run("a group with a retention reports it", func(t *testing.T) {
		lg, err := buildLogGroupResource(testRuntime(), "us-east-1", cloudwatchlogstypes.LogGroup{
			Arn:             aws.String("arn:aws:logs:us-east-1:000000000000:log-group:/aws/lambda/fn:*"),
			LogGroupName:    aws.String("/aws/lambda/fn"),
			RetentionInDays: aws.Int32(365),
		})
		require.NoError(t, err)

		assert.False(t, lg.RetentionInDays.IsNull())
		assert.Equal(t, int64(365), lg.RetentionInDays.Data)
		assert.False(t, lg.NeverExpires.Data)
	})

	t.Run("a group that never expires reports null, not zero", func(t *testing.T) {
		lg, err := buildLogGroupResource(testRuntime(), "us-east-1", cloudwatchlogstypes.LogGroup{
			Arn:          aws.String("arn:aws:logs:us-east-1:000000000000:log-group:/aws/lambda/forever:*"),
			LogGroupName: aws.String("/aws/lambda/forever"),
		})
		require.NoError(t, err)

		assert.True(t, lg.RetentionInDays.IsNull(),
			"no retention means no expiry, which is not a retention of zero days")
		assert.True(t, lg.NeverExpires.Data)
	})
}
