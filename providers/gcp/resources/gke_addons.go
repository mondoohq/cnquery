// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"cloud.google.com/go/container/apiv1/containerpb"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// gkeAddonsConfigArgs maps a cluster's addon configuration onto resource
// arguments.
//
// Every addon in containerpb.AddonsConfig is its own optional message wrapping
// a single bool. GKE omits the message entirely when the addon sits at its
// default, so reading the addon through the dict gives null rather than false,
// and a check written as `networkPolicyConfig.disabled == false` evaluates
// against null and passes without ever having read the cluster. Flattening the
// wrappers to plain bools makes an absent message report the default it stands
// for: an unset Disabled is false, an unset Enabled is false.
func gkeAddonsConfigArgs(id string, cfg *containerpb.AddonsConfig) map[string]*llx.RawData {
	var httpLoadBalancing, horizontalPodAutoscaling, kubernetesDashboard, networkPolicyConfig map[string]any
	var cloudRunConfig, dnsCacheConfig, configConnectorConfig map[string]any
	var gcePersistentDiskCsiDriverConfig, gcpFilestoreCsiDriverConfig map[string]any
	var gkeBackupAgentConfig, gcsFuseCsiDriverConfig, statefulHaConfig map[string]any

	var httpLoadBalancingDisabled, horizontalPodAutoscalingDisabled bool
	var kubernetesDashboardDisabled, networkPolicyConfigDisabled, cloudRunDisabled bool
	var cloudRunLoadBalancerType string
	var dnsCacheEnabled, configConnectorEnabled bool
	var gcePersistentDiskCsiDriverEnabled, gcpFilestoreCsiDriverEnabled bool
	var gkeBackupAgentEnabled, gcsFuseCsiDriverEnabled, statefulHaEnabled bool

	if cfg != nil {
		if c := cfg.HttpLoadBalancing; c != nil {
			httpLoadBalancingDisabled = c.Disabled
			httpLoadBalancing = map[string]any{"disabled": c.Disabled}
		}
		if c := cfg.HorizontalPodAutoscaling; c != nil {
			horizontalPodAutoscalingDisabled = c.Disabled
			horizontalPodAutoscaling = map[string]any{"disabled": c.Disabled}
		}
		if c := cfg.KubernetesDashboard; c != nil {
			kubernetesDashboardDisabled = c.Disabled
			kubernetesDashboard = map[string]any{"disabled": c.Disabled}
		}
		if c := cfg.NetworkPolicyConfig; c != nil {
			networkPolicyConfigDisabled = c.Disabled
			networkPolicyConfig = map[string]any{"disabled": c.Disabled}
		}
		if c := cfg.CloudRunConfig; c != nil {
			cloudRunDisabled = c.Disabled
			cloudRunLoadBalancerType = c.LoadBalancerType.String()
			cloudRunConfig = map[string]any{
				"disabled":         c.Disabled,
				"loadBalancerType": c.LoadBalancerType.String(),
			}
		}
		if c := cfg.DnsCacheConfig; c != nil {
			dnsCacheEnabled = c.Enabled
			dnsCacheConfig = map[string]any{"enabled": c.Enabled}
		}
		if c := cfg.ConfigConnectorConfig; c != nil {
			configConnectorEnabled = c.Enabled
			configConnectorConfig = map[string]any{"enabled": c.Enabled}
		}
		if c := cfg.GcePersistentDiskCsiDriverConfig; c != nil {
			gcePersistentDiskCsiDriverEnabled = c.Enabled
			gcePersistentDiskCsiDriverConfig = map[string]any{"enabled": c.Enabled}
		}
		if c := cfg.GcpFilestoreCsiDriverConfig; c != nil {
			gcpFilestoreCsiDriverEnabled = c.Enabled
			gcpFilestoreCsiDriverConfig = map[string]any{"enabled": c.Enabled}
		}
		if c := cfg.GkeBackupAgentConfig; c != nil {
			gkeBackupAgentEnabled = c.Enabled
			gkeBackupAgentConfig = map[string]any{"enabled": c.Enabled}
		}
		if c := cfg.GcsFuseCsiDriverConfig; c != nil {
			gcsFuseCsiDriverEnabled = c.Enabled
			gcsFuseCsiDriverConfig = map[string]any{"enabled": c.Enabled}
		}
		if c := cfg.StatefulHaConfig; c != nil {
			statefulHaEnabled = c.Enabled
			statefulHaConfig = map[string]any{"enabled": c.Enabled}
		}
	}

	return map[string]*llx.RawData{
		"id":                                llx.StringData(id),
		"httpLoadBalancing":                 llx.DictData(httpLoadBalancing),
		"horizontalPodAutoscaling":          llx.DictData(horizontalPodAutoscaling),
		"kubernetesDashboard":               llx.DictData(kubernetesDashboard),
		"networkPolicyConfig":               llx.DictData(networkPolicyConfig),
		"cloudRunConfig":                    llx.DictData(cloudRunConfig),
		"dnsCacheConfig":                    llx.DictData(dnsCacheConfig),
		"configConnectorConfig":             llx.DictData(configConnectorConfig),
		"gcePersistentDiskCsiDriverConfig":  llx.DictData(gcePersistentDiskCsiDriverConfig),
		"gcpFilestoreCsiDriverConfig":       llx.DictData(gcpFilestoreCsiDriverConfig),
		"gkeBackupAgentConfig":              llx.DictData(gkeBackupAgentConfig),
		"gcsFuseCsiDriverConfig":            llx.DictData(gcsFuseCsiDriverConfig),
		"statefulHaConfig":                  llx.DictData(statefulHaConfig),
		"httpLoadBalancingDisabled":         llx.BoolData(httpLoadBalancingDisabled),
		"horizontalPodAutoscalingDisabled":  llx.BoolData(horizontalPodAutoscalingDisabled),
		"kubernetesDashboardDisabled":       llx.BoolData(kubernetesDashboardDisabled),
		"networkPolicyConfigDisabled":       llx.BoolData(networkPolicyConfigDisabled),
		"cloudRunDisabled":                  llx.BoolData(cloudRunDisabled),
		"cloudRunLoadBalancerType":          llx.StringData(cloudRunLoadBalancerType),
		"dnsCacheEnabled":                   llx.BoolData(dnsCacheEnabled),
		"configConnectorEnabled":            llx.BoolData(configConnectorEnabled),
		"gcePersistentDiskCsiDriverEnabled": llx.BoolData(gcePersistentDiskCsiDriverEnabled),
		"gcpFilestoreCsiDriverEnabled":      llx.BoolData(gcpFilestoreCsiDriverEnabled),
		"gkeBackupAgentEnabled":             llx.BoolData(gkeBackupAgentEnabled),
		"gcsFuseCsiDriverEnabled":           llx.BoolData(gcsFuseCsiDriverEnabled),
		"statefulHaEnabled":                 llx.BoolData(statefulHaEnabled),
	}
}

// newMqlGkeControlPlaneEndpoints promotes the control plane endpoint
// configuration of a cluster. Returns llx.NilData for a cluster that reports no
// configuration, which is what a cluster still on the legacy
// privateClusterConfig fields does.
//
// Every bool in the message is optional, so an unset one stays null rather than
// reporting false: "GKE did not say" and "the endpoint is off" are different
// claims about whether an API server is reachable.
func newMqlGkeControlPlaneEndpoints(runtime *plugin.Runtime, clusterID string, cfg *containerpb.ControlPlaneEndpointsConfig) (*llx.RawData, error) {
	if cfg == nil {
		return llx.NilData, nil
	}

	args := map[string]*llx.RawData{
		"__id": llx.StringData(clusterID + "/controlPlaneEndpoints"),
	}

	if ip := cfg.GetIpEndpointsConfig(); ip != nil {
		args["ipEndpointsEnabled"] = llx.BoolDataPtr(ip.Enabled)
		args["publicEndpointEnabled"] = llx.BoolDataPtr(ip.EnablePublicEndpoint)
		args["globalAccess"] = llx.BoolDataPtr(ip.GlobalAccess)
		args["publicEndpointAddress"] = llx.StringData(ip.PublicEndpoint)
		args["privateEndpointAddress"] = llx.StringData(ip.PrivateEndpoint)
		args["privateEndpointSubnetwork"] = llx.StringData(ip.PrivateEndpointSubnetwork)
	} else {
		args["ipEndpointsEnabled"] = llx.NilData
		args["publicEndpointEnabled"] = llx.NilData
		args["globalAccess"] = llx.NilData
		args["publicEndpointAddress"] = llx.StringData("")
		args["privateEndpointAddress"] = llx.StringData("")
		args["privateEndpointSubnetwork"] = llx.StringData("")
	}

	if dns := cfg.GetDnsEndpointConfig(); dns != nil {
		args["dnsEndpoint"] = llx.StringData(dns.Endpoint)
		args["dnsAllowExternalTraffic"] = llx.BoolDataPtr(dns.AllowExternalTraffic)
		args["dnsK8sTokensEnabled"] = llx.BoolDataPtr(dns.EnableK8STokensViaDns)
		args["dnsK8sCertsEnabled"] = llx.BoolDataPtr(dns.EnableK8SCertsViaDns)
	} else {
		args["dnsEndpoint"] = llx.StringData("")
		args["dnsAllowExternalTraffic"] = llx.NilData
		args["dnsK8sTokensEnabled"] = llx.NilData
		args["dnsK8sCertsEnabled"] = llx.NilData
	}

	res, err := CreateResource(runtime, "gcp.project.gkeService.cluster.controlPlaneEndpoints", args)
	if err != nil {
		return nil, err
	}
	return llx.ResourceData(res, "gcp.project.gkeService.cluster.controlPlaneEndpoints"), nil
}
