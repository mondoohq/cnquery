// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
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
