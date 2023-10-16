// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers/aws/connection"
)

func TestFilters(t *testing.T) {
	require.True(t, instanceMatchesFilters(&mqlAwsEc2Instance{InstanceId: plugin.TValue[string]{Data: "i-test"}}, connection.DiscoveryFilters{
		Ec2DiscoveryFilters: connection.Ec2DiscoveryFilters{
			InstanceIds: []string{"i-test"},
		},
	},
	))
	require.True(t, instanceMatchesFilters(&mqlAwsEc2Instance{
		InstanceId: plugin.TValue[string]{Data: "i-test"},
		Tags:       plugin.TValue[map[string]interface{}]{Data: map[string]interface{}{"tester2": "val2", "test-tag": "val"}},
	}, connection.DiscoveryFilters{
		Ec2DiscoveryFilters: connection.Ec2DiscoveryFilters{
			InstanceIds: []string{"i-test"},
			Tags:        map[string]string{"tester2": "val2"},
		},
	},
	))
	require.True(t, instanceMatchesFilters(&mqlAwsEc2Instance{
		InstanceId: plugin.TValue[string]{Data: "i-test"},
		Tags:       plugin.TValue[map[string]interface{}]{Data: map[string]interface{}{"tester2": "val2", "test-tag": "val"}},
		Region:     plugin.TValue[string]{Data: "us-west-1"},
	}, connection.DiscoveryFilters{
		Ec2DiscoveryFilters: connection.Ec2DiscoveryFilters{
			Regions: []string{"us-east-2", "us-east-1", "us-west-1"},
		},
	},
	))
	require.True(t, instanceMatchesFilters(&mqlAwsEc2Instance{
		InstanceId: plugin.TValue[string]{Data: "i-test"},
		Tags:       plugin.TValue[map[string]interface{}]{Data: map[string]interface{}{"tester2": "val2", "test-tag": "val"}},
		Region:     plugin.TValue[string]{Data: "us-west-1"},
	}, connection.DiscoveryFilters{}))
	require.False(t, instanceMatchesFilters(&mqlAwsEc2Instance{InstanceId: plugin.TValue[string]{Data: "i-test"}}, connection.DiscoveryFilters{
		Ec2DiscoveryFilters: connection.Ec2DiscoveryFilters{
			InstanceIds: []string{"i-test2"},
		},
	},
	))
	require.False(t, instanceMatchesFilters(&mqlAwsEc2Instance{
		InstanceId: plugin.TValue[string]{Data: "i-test"},
		Tags:       plugin.TValue[map[string]interface{}]{Data: map[string]interface{}{"tester2": "val2", "test-tag": "val"}},
	}, connection.DiscoveryFilters{
		Ec2DiscoveryFilters: connection.Ec2DiscoveryFilters{
			InstanceIds: []string{"i-test"},
			Tags:        map[string]string{"test-tag": "val2"},
		},
	},
	))
	require.False(t, instanceMatchesFilters(&mqlAwsEc2Instance{
		InstanceId: plugin.TValue[string]{Data: "i-test"},
		Tags:       plugin.TValue[map[string]interface{}]{Data: map[string]interface{}{"tester2": "val2", "test-tag": "val"}},
		Region:     plugin.TValue[string]{Data: "us-west-2"},
	}, connection.DiscoveryFilters{
		Ec2DiscoveryFilters: connection.Ec2DiscoveryFilters{
			Regions: []string{"us-east-2", "us-east-1", "us-west-1"},
		},
	},
	))

	require.True(t, imageMatchesFilters(&mqlAwsEcrImage{
		Tags: plugin.TValue[[]interface{}]{Data: []interface{}{"latest"}},
	}, connection.DiscoveryFilters{}))
	require.True(t, imageMatchesFilters(&mqlAwsEcrImage{
		Tags: plugin.TValue[[]interface{}]{Data: []interface{}{"latest"}},
	}, connection.DiscoveryFilters{
		EcrDiscoveryFilters: connection.EcrDiscoveryFilters{
			Tags: []string{"latest"},
		},
	}))
	require.False(t, imageMatchesFilters(&mqlAwsEcrImage{
		Tags: plugin.TValue[[]interface{}]{Data: []interface{}{"ubu", "test"}},
	}, connection.DiscoveryFilters{
		EcrDiscoveryFilters: connection.EcrDiscoveryFilters{
			Tags: []string{"latest"},
		},
	}))

	require.True(t, containerMatchesFilters(&mqlAwsEcsContainer{
		Status: plugin.TValue[string]{Data: "RUNNING"},
	}, connection.DiscoveryFilters{}))

	require.True(t, containerMatchesFilters(&mqlAwsEcsContainer{
		Status: plugin.TValue[string]{Data: "RUNNING"},
	}, connection.DiscoveryFilters{EcsDiscoveryFilters: connection.EcsDiscoveryFilters{
		OnlyRunningContainers: true,
	}}))

	require.False(t, containerMatchesFilters(&mqlAwsEcsContainer{
		Status: plugin.TValue[string]{Data: "STOPPED"},
	}, connection.DiscoveryFilters{EcsDiscoveryFilters: connection.EcsDiscoveryFilters{
		OnlyRunningContainers: true,
	}}))

	require.True(t, discoveredAssetMatchesGeneralFilters(&inventory.Asset{
		Labels: map[string]string{"test": "val", "another": "value"},
	}, connection.GeneralResourceDiscoveryFilters{}))
	require.True(t, discoveredAssetMatchesGeneralFilters(&inventory.Asset{
		Labels: nil,
	}, connection.GeneralResourceDiscoveryFilters{}))
	require.True(t, discoveredAssetMatchesGeneralFilters(&inventory.Asset{
		Labels: map[string]string{"test": "val", "another": "value"},
	}, connection.GeneralResourceDiscoveryFilters{Tags: map[string]string{"another": "value"}}))

	require.False(t, discoveredAssetMatchesGeneralFilters(&inventory.Asset{
		Labels: map[string]string{"test": "val", "another": "value"},
	}, connection.GeneralResourceDiscoveryFilters{Tags: map[string]string{"something": "else"}}))
	require.False(t, discoveredAssetMatchesGeneralFilters(&inventory.Asset{
		Labels: nil,
	}, connection.GeneralResourceDiscoveryFilters{Tags: map[string]string{"something": "else"}}))

	require.True(t, shouldScanEcsContainerImages(connection.DiscoveryFilters{
		EcsDiscoveryFilters: connection.EcsDiscoveryFilters{
			DiscoverImages: true,
		},
	}))
	require.False(t, shouldScanEcsContainerImages(connection.DiscoveryFilters{}))
	require.True(t, shouldScanEcsContainerInstances(connection.DiscoveryFilters{
		EcsDiscoveryFilters: connection.EcsDiscoveryFilters{
			DiscoverImages:    false,
			DiscoverInstances: true,
		},
	}))
	require.False(t, shouldScanEcsContainerInstances(connection.DiscoveryFilters{}))
}
