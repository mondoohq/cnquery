// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	route53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheckProtocol(t *testing.T) {
	tests := []struct {
		name     string
		hcType   route53types.HealthCheckType
		expected string
	}{
		{"HTTP", route53types.HealthCheckTypeHttp, "HTTP"},
		{"HTTP_STR_MATCH", route53types.HealthCheckTypeHttpStrMatch, "HTTP"},
		{"HTTPS", route53types.HealthCheckTypeHttps, "HTTPS"},
		{"HTTPS_STR_MATCH", route53types.HealthCheckTypeHttpsStrMatch, "HTTPS"},
		{"TCP", route53types.HealthCheckTypeTcp, "TCP"},
		{"CALCULATED returns empty", route53types.HealthCheckTypeCalculated, ""},
		{"CLOUDWATCH_METRIC returns empty", route53types.HealthCheckTypeCloudwatchMetric, ""},
		{"RECOVERY_CONTROL returns empty", route53types.HealthCheckTypeRecoveryControl, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, healthCheckProtocol(tt.hcType))
		})
	}
}

func TestHostedZoneIdToArn(t *testing.T) {
	t.Run("nil returns empty string", func(t *testing.T) {
		assert.Equal(t, "", hostedZoneIdToArn(nil))
	})

	t.Run("plain zone ID", func(t *testing.T) {
		assert.Equal(t,
			"arn:aws:route53:::hostedzone/Z1234567890",
			hostedZoneIdToArn(aws.String("Z1234567890")),
		)
	})

	t.Run("strips /hostedzone/ prefix", func(t *testing.T) {
		assert.Equal(t,
			"arn:aws:route53:::hostedzone/Z1234567890",
			hostedZoneIdToArn(aws.String("/hostedzone/Z1234567890")),
		)
	})
}

func TestHealthCheckIdToArn(t *testing.T) {
	t.Run("nil returns empty string", func(t *testing.T) {
		assert.Equal(t, "", healthCheckIdToArn(nil))
	})

	t.Run("returns ARN with health check ID", func(t *testing.T) {
		assert.Equal(t,
			"arn:aws:route53:::healthcheck/abcdef-1234",
			healthCheckIdToArn(aws.String("abcdef-1234")),
		)
	})
}
