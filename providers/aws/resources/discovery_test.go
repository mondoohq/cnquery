// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sort"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/aws/connection"
)

func TestFilters(t *testing.T) {
	// image filters
	require.True(t, imageMatchesFilters(&mqlAwsEcrImage{
		Tags: plugin.TValue[[]any]{Data: []any{"latest"}},
	}, connection.EcrDiscoveryFilters{}))

	require.True(t, imageMatchesFilters(&mqlAwsEcrImage{
		Tags: plugin.TValue[[]any]{Data: []any{"latest"}},
	}, connection.EcrDiscoveryFilters{Tags: []string{"latest"}}))

	require.False(t, imageMatchesFilters(&mqlAwsEcrImage{
		Tags: plugin.TValue[[]any]{Data: []any{"ubu", "test"}},
	}, connection.EcrDiscoveryFilters{Tags: []string{"latest"}}))

	// container filters
	require.True(t, containerMatchesFilters(&mqlAwsEcsContainer{
		Status: plugin.TValue[string]{Data: "RUNNING"},
	}, connection.EcsDiscoveryFilters{}))

	require.True(t, containerMatchesFilters(&mqlAwsEcsContainer{
		Status: plugin.TValue[string]{Data: "RUNNING"},
	}, connection.EcsDiscoveryFilters{OnlyRunningContainers: true}))

	require.False(t, containerMatchesFilters(&mqlAwsEcsContainer{
		Status: plugin.TValue[string]{Data: "STOPPED"},
	}, connection.EcsDiscoveryFilters{OnlyRunningContainers: true}))
}

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
	config := &inventory.Config{
		Discover: &inventory.Discovery{
			Targets: []string{},
		},
	}
	// test all with other stuff
	config.Discover.Targets = []string{"all", "projects", "instances"}
	require.Equal(t, allDiscovery(), getDiscoveryTargets(config))

	// test just all
	config.Discover.Targets = []string{"all"}
	require.Equal(t, allDiscovery(), getDiscoveryTargets(config))

	// test auto with other stuff
	config.Discover.Targets = []string{"auto", "s3-buckets", "iam-users"}
	res := append(Auto, []string{DiscoveryS3Buckets, DiscoveryIAMUsers}...)
	sort.Strings(res)
	targets := getDiscoveryTargets(config)
	sort.Strings(targets)
	require.Equal(t, res, targets)

	// test just auto
	config.Discover.Targets = []string{"auto"}
	require.Equal(t, Auto, getDiscoveryTargets(config))

	// test random
	config.Discover.Targets = []string{"s3-buckets", "iam-users", "instances"}
	require.Equal(t, []string{DiscoveryS3Buckets, DiscoveryIAMUsers, DiscoveryInstances}, getDiscoveryTargets(config))
}
