// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	scheduler_types "github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/stretchr/testify/assert"
)

func TestScheduleTargetType(t *testing.T) {
	cases := []struct {
		name string
		arn  string
		t    *scheduler_types.Target
		want string
	}{
		// universal target wins over any other prefix match
		{"universal target", "arn:aws:scheduler:::aws-sdk:s3:PutObject", nil, "universal"},

		// ARN-prefix dispatch
		{"lambda", "arn:aws:lambda:us-east-1:111:function:foo", nil, "lambda"},
		{"sqs", "arn:aws:sqs:us-east-1:111:my-queue", nil, "sqs"},
		{"sns", "arn:aws:sns:us-east-1:111:topic", nil, "sns"},
		{"step function", "arn:aws:states:us-east-1:111:stateMachine:foo", nil, "stepFunction"},
		{"kinesis", "arn:aws:kinesis:us-east-1:111:stream/foo", nil, "kinesis"},
		{"firehose", "arn:aws:firehose:us-east-1:111:deliverystream/foo", nil, "firehose"},
		{"ecs", "arn:aws:ecs:us-east-1:111:cluster/foo", nil, "ecs"},
		{"event bus", "arn:aws:events:us-east-1:111:event-bus/default", nil, "eventBridge"},
		{"sagemaker pipeline", "arn:aws:sagemaker:us-east-1:111:pipeline/foo", nil, "sagemakerPipeline"},
		{"sagemaker non-pipeline ignored", "arn:aws:sagemaker:us-east-1:111:notebook-instance/foo", nil, "unknown"},
		{"codebuild", "arn:aws:codebuild:us-east-1:111:project/foo", nil, "codebuild"},

		// fallback: parameter block tells us when ARN doesn't match
		{"unknown ARN with EcsParameters falls back to ecs", "arn:aws:other:something", &scheduler_types.Target{EcsParameters: &scheduler_types.EcsParameters{}}, "ecs"},
		{"unknown ARN with EventBridgeParameters falls back", "arn:aws:other:something", &scheduler_types.Target{EventBridgeParameters: &scheduler_types.EventBridgeParameters{}}, "eventBridge"},
		{"unknown ARN with KinesisParameters falls back", "arn:aws:other:something", &scheduler_types.Target{KinesisParameters: &scheduler_types.KinesisParameters{}}, "kinesis"},
		{"unknown ARN with SageMakerPipelineParameters falls back", "arn:aws:other:something", &scheduler_types.Target{SageMakerPipelineParameters: &scheduler_types.SageMakerPipelineParameters{}}, "sagemakerPipeline"},
		{"unknown ARN with SqsParameters falls back", "arn:aws:other:something", &scheduler_types.Target{SqsParameters: &scheduler_types.SqsParameters{}}, "sqs"},

		// empty / unknown
		{"empty ARN, nil target", "", nil, "unknown"},
		{"unknown ARN, nil target", "arn:aws:s3:::bucket", nil, "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, scheduleTargetType(tc.arn, tc.t))
		})
	}
}

func TestParseUniversalTargetArn(t *testing.T) {
	cases := []struct {
		name        string
		arn         string
		wantService string
		wantAction  string
	}{
		{"empty", "", "", ""},
		{"non-universal ARN", "arn:aws:lambda:us-east-1:111:function:foo", "", ""},
		{"missing prefix", "arn:aws:scheduler:us-east-1:111:schedule/default/foo", "", ""},
		{"valid s3 PutObject", "arn:aws:scheduler:::aws-sdk:s3:PutObject", "s3", "PutObject"},
		{"valid ec2 RunInstances", "arn:aws:scheduler:::aws-sdk:ec2:RunInstances", "ec2", "RunInstances"},
		{"valid action with colons in path-style ARN", "arn:aws:scheduler:::aws-sdk:dynamodb:UpdateItem", "dynamodb", "UpdateItem"},

		// malformed: missing action
		{"no action separator", "arn:aws:scheduler:::aws-sdk:s3", "", ""},
		{"trailing colon, no action", "arn:aws:scheduler:::aws-sdk:s3:", "", ""},
		{"leading colon, no service", "arn:aws:scheduler:::aws-sdk::PutObject", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotService, gotAction := parseUniversalTargetArn(tc.arn)
			assert.Equal(t, tc.wantService, gotService, "service")
			assert.Equal(t, tc.wantAction, gotAction, "action")
		})
	}
}
