// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEcsArnSpecParse keeps the ARN-shape cases that used to guard the ECS
// provider's own validateAndParseARN, now that arnSpec.parse does that work for
// every resource.
func TestEcsArnSpecParse(t *testing.T) {
	tests := []struct {
		name            string
		arn             string
		expectedService string
		wantErr         bool
	}{
		{
			name:            "valid ecs arn",
			arn:             "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster",
			expectedService: "ecs",
			wantErr:         false,
		},
		{
			name:            "sts arn instead of ecs arn",
			arn:             "arn:aws:sts::162854405951", // bug #6370
			expectedService: "ecs",
			wantErr:         true,
		},
		{
			name:            "invalid arn format - too short",
			arn:             "arn:aws:ecs",
			expectedService: "ecs",
			wantErr:         true,
		},
		{
			name:            "not an arn",
			arn:             "my-cluster",
			expectedService: "ecs",
			wantErr:         true,
		},
		{
			name:            "wrong service",
			arn:             "arn:aws:s3:::my-bucket",
			expectedService: "ecs",
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := arnSpec{resource: ResourceAwsEcsCluster, services: []string{tt.expectedService}}
			got, err := spec.parse(tt.arn)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.arn, got.RawArn)
				require.Equal(t, tt.expectedService, got.Service)
			}
		})
	}
}
