// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
