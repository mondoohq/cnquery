// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"cloud.google.com/go/container/apiv1/containerpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GKE omits an addon's wrapper message entirely when the addon sits at its
// default. Read through the dict that reported null, so a policy written as
// `networkPolicyConfig.disabled == false` evaluated against null and passed
// without having read anything. The flattened bools must report false for an
// absent message, which is the default the absence stands for.
func TestGkeAddonsConfigArgsAbsentAddonsReadFalseNotNull(t *testing.T) {
	args := gkeAddonsConfigArgs("cluster/addonsConfig", &containerpb.AddonsConfig{})

	for _, field := range []string{
		"httpLoadBalancingDisabled",
		"horizontalPodAutoscalingDisabled",
		"kubernetesDashboardDisabled",
		"networkPolicyConfigDisabled",
		"cloudRunDisabled",
		"dnsCacheEnabled",
		"configConnectorEnabled",
		"gcePersistentDiskCsiDriverEnabled",
		"gcpFilestoreCsiDriverEnabled",
		"gkeBackupAgentEnabled",
		"gcsFuseCsiDriverEnabled",
		"statefulHaEnabled",
	} {
		require.Contains(t, args, field)
		assert.Equal(t, false, args[field].Value, field)
	}
	assert.Equal(t, "", args["cloudRunLoadBalancerType"].Value)

	// The deprecated dicts keep their old behavior: absent message, null dict.
	assert.Nil(t, args["networkPolicyConfig"].Value)
	assert.Nil(t, args["cloudRunConfig"].Value)
}

// A nil AddonsConfig is the same claim as an empty one: every addon at its
// default. It must not panic and must not report an addon as enabled.
func TestGkeAddonsConfigArgsNilConfig(t *testing.T) {
	args := gkeAddonsConfigArgs("cluster/addonsConfig", nil)

	assert.Equal(t, false, args["kubernetesDashboardDisabled"].Value)
	assert.Equal(t, false, args["dnsCacheEnabled"].Value)
	assert.Nil(t, args["kubernetesDashboard"].Value)
}

// An addon message that is present carries the real value, in both directions.
func TestGkeAddonsConfigArgsPresentAddons(t *testing.T) {
	args := gkeAddonsConfigArgs("cluster/addonsConfig", &containerpb.AddonsConfig{
		NetworkPolicyConfig:              &containerpb.NetworkPolicyConfig{Disabled: true},
		KubernetesDashboard:              &containerpb.KubernetesDashboard{Disabled: false},
		HttpLoadBalancing:                &containerpb.HttpLoadBalancing{Disabled: true},
		DnsCacheConfig:                   &containerpb.DnsCacheConfig{Enabled: true},
		ConfigConnectorConfig:            &containerpb.ConfigConnectorConfig{Enabled: false},
		GkeBackupAgentConfig:             &containerpb.GkeBackupAgentConfig{Enabled: true},
		GcsFuseCsiDriverConfig:           &containerpb.GcsFuseCsiDriverConfig{Enabled: true},
		StatefulHaConfig:                 &containerpb.StatefulHAConfig{Enabled: true},
		GcpFilestoreCsiDriverConfig:      &containerpb.GcpFilestoreCsiDriverConfig{Enabled: true},
		GcePersistentDiskCsiDriverConfig: &containerpb.GcePersistentDiskCsiDriverConfig{Enabled: true},
		HorizontalPodAutoscaling:         &containerpb.HorizontalPodAutoscaling{Disabled: true},
		CloudRunConfig: &containerpb.CloudRunConfig{
			Disabled:         false,
			LoadBalancerType: containerpb.CloudRunConfig_LOAD_BALANCER_TYPE_EXTERNAL,
		},
	})

	assert.Equal(t, true, args["networkPolicyConfigDisabled"].Value)
	assert.Equal(t, false, args["kubernetesDashboardDisabled"].Value)
	assert.Equal(t, true, args["httpLoadBalancingDisabled"].Value)
	assert.Equal(t, true, args["horizontalPodAutoscalingDisabled"].Value)
	assert.Equal(t, true, args["dnsCacheEnabled"].Value)
	assert.Equal(t, false, args["configConnectorEnabled"].Value)
	assert.Equal(t, true, args["gkeBackupAgentEnabled"].Value)
	assert.Equal(t, true, args["gcsFuseCsiDriverEnabled"].Value)
	assert.Equal(t, true, args["statefulHaEnabled"].Value)
	assert.Equal(t, true, args["gcpFilestoreCsiDriverEnabled"].Value)
	assert.Equal(t, true, args["gcePersistentDiskCsiDriverEnabled"].Value)
	assert.Equal(t, false, args["cloudRunDisabled"].Value)

	// The enum has to reach MQL as its name. An external load balancer is what
	// publishes the cluster's Cloud Run services on the internet, and a bare
	// ordinal cannot be compared against in a policy.
	assert.Equal(t, "LOAD_BALANCER_TYPE_EXTERNAL", args["cloudRunLoadBalancerType"].Value)
}
