// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
)

func TestAddConnInfoToEc2Instances(t *testing.T) {
	info := instanceInfo{}
	a := &inventory.Asset{}
	addMondooLabels(info, a)
	require.Equal(t, map[string]string{"mondoo.com/instance-id": "", "mondoo.com/instance-type": "", "mondoo.com/parent-id": "", "mondoo.com/platform": "", "mondoo.com/region": ""}, a.Labels)
	info = instanceInfo{
		region:          "us-west-1",
		platformDetails: "windows",
		instanceType:    "t4g.medium",
		accountId:       "00000000000000",
		instanceId:      "i-9049034093403",
		launchTime:      nil,
	}
	a = &inventory.Asset{}
	expectedLabels := map[string]string{"mondoo.com/instance-id": "i-9049034093403", "mondoo.com/instance-type": "t4g.medium", "mondoo.com/parent-id": "00000000000000", "mondoo.com/platform": "windows", "mondoo.com/region": "us-west-1"}
	addMondooLabels(info, a)
	require.Equal(t, expectedLabels, a.Labels)
	now := time.Now()
	info.launchTime = &now
	addMondooLabels(info, a)
	require.NotNil(t, expectedLabels[MondooLaunchTimeLabelKey])
	info.image = aws.String("test")
	addMondooLabels(info, a)
	require.NotNil(t, expectedLabels[MondooImageLabelKey])
	info.instanceTags = nil
	addMondooLabels(info, a)
	info.instanceTags = map[string]string{"testing-key": "testing-val"}
	addMondooLabels(info, a)
	require.Equal(t, a.Labels["testing-key"], "testing-val")
}

func TestGetDiscoveryTargets(t *testing.T) {
	cases := []struct {
		name    string
		targets []string
		want    []string
	}{
		{
			name:    "empty",
			targets: []string{},
			want:    Auto,
		},
		{
			name:    "all",
			targets: []string{"all"},
			want:    allDiscovery(),
		},
		{
			name:    "auto",
			targets: []string{"auto"},
			want:    Auto,
		},
		{
			name:    "resources",
			targets: []string{"resources"},
			want:    AllAPIResources,
		},
		{
			name:    "auto and resources",
			targets: []string{"auto", "resources"},
			want:    append(Auto, AllAPIResources...),
		},
		{
			name:    "all and resources",
			targets: []string{"all", "resources"},
			want:    allDiscovery(),
		},
		{
			name:    "all, auto and resources",
			targets: []string{"all", "resources"},
			want:    allDiscovery(),
		},
		{
			name:    "random",
			targets: []string{"s3-buckets", "iam-users", "instances"},
			want:    []string{DiscoveryS3Buckets, DiscoveryIAMUsers, DiscoveryInstances},
		},
		{
			name:    "duplicates",
			targets: []string{"auto", "s3-buckets", "iam-users", "s3-buckets", "auto"},
			want:    append(Auto, []string{DiscoveryS3Buckets, DiscoveryIAMUsers}...),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := &inventory.Config{
				Discover: &inventory.Discovery{
					Targets: tc.targets,
				},
			}
			got := getDiscoveryTargets(config)
			require.ElementsMatch(t, tc.want, got)
		})
	}
}
