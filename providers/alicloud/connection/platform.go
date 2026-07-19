// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

const (
	// Scope options that pin a discovered child connection to a single object.
	OptionClusterID     = "cluster-id"
	OptionAlbID         = "alb-id"
	OptionNlbID         = "nlb-id"
	OptionVpcID         = "vpc-id"
	OptionWafInstanceID = "waf-instance-id"
	OptionCloudFirewall = "cloud-firewall"
)

const (
	platformIDAccount       = "//platformid.api.mondoo.app/runtime/alicloud/account/"
	platformIDAckCluster    = "//platformid.api.mondoo.app/runtime/alicloud/ack/cluster/"
	platformIDAlb           = "//platformid.api.mondoo.app/runtime/alicloud/alb/loadbalancer/"
	platformIDNlb           = "//platformid.api.mondoo.app/runtime/alicloud/nlb/loadbalancer/"
	platformIDVpc           = "//platformid.api.mondoo.app/runtime/alicloud/vpc/network/"
	platformIDWafInstance   = "//platformid.api.mondoo.app/runtime/alicloud/waf/instance/"
	platformIDCloudFirewall = "//platformid.api.mondoo.app/runtime/alicloud/cloudfirewall/"
)

// Platforms is the static catalog of platforms the alicloud provider emits: the
// account itself, and the per-object platforms produced by fine-grained asset
// discovery.
var Platforms = []*plugin.PlatformInfo{
	{Name: "alicloud", Title: "Alibaba Cloud account", Family: []string{"alicloud"}, Kind: []string{"api"}, Runtime: []string{"alicloud"}},
	{Name: "alicloud-ack-cluster", Title: "Alibaba Cloud ACK Cluster", Family: []string{"alicloud"}, Kind: []string{"api"}, Runtime: []string{"alicloud"}},
	{Name: "alicloud-alb-loadbalancer", Title: "Alibaba Cloud Application Load Balancer", Family: []string{"alicloud"}, Kind: []string{"api"}, Runtime: []string{"alicloud"}},
	{Name: "alicloud-nlb-loadbalancer", Title: "Alibaba Cloud Network Load Balancer", Family: []string{"alicloud"}, Kind: []string{"api"}, Runtime: []string{"alicloud"}},
	{Name: "alicloud-vpc", Title: "Alibaba Cloud VPC", Family: []string{"alicloud"}, Kind: []string{"api"}, Runtime: []string{"alicloud"}},
	{Name: "alicloud-waf-instance", Title: "Alibaba Cloud Web Application Firewall", Family: []string{"alicloud"}, Kind: []string{"api"}, Runtime: []string{"alicloud"}},
	{Name: "alicloud-cloud-firewall", Title: "Alibaba Cloud Cloud Firewall", Family: []string{"alicloud"}, Kind: []string{"api"}, Runtime: []string{"alicloud"}},
}

var platformsByName = plugin.PlatformsByName(Platforms)

// PlatformByName returns the catalog entry for the given platform name.
func PlatformByName(name string) *plugin.PlatformInfo {
	return platformsByName[name]
}

func newPlatform(name, title string, segments []string) *inventory.Platform {
	p := &inventory.Platform{}
	if title != "" {
		p.Title = title
	}
	p.TechnologyUrlSegments = segments
	PlatformByName(name).Apply(p)
	return p
}

// NewAccountPlatform returns the platform for an Alibaba Cloud account asset.
func NewAccountPlatform(accountID string) *inventory.Platform {
	return newPlatform("alicloud", "Alibaba Cloud account "+accountID,
		[]string{"technology=alicloud", "kind=account", "account=" + accountID})
}

// NewAckClusterPlatform returns the platform for a discovered ACK cluster asset.
func NewAckClusterPlatform(clusterID string) *inventory.Platform {
	return newPlatform("alicloud-ack-cluster", "",
		[]string{"technology=alicloud", "kind=ack-cluster", "cluster=" + clusterID})
}

// NewAlbPlatform returns the platform for a discovered ALB load balancer asset.
func NewAlbPlatform(lbID string) *inventory.Platform {
	return newPlatform("alicloud-alb-loadbalancer", "",
		[]string{"technology=alicloud", "kind=alb-loadbalancer", "loadbalancer=" + lbID})
}

// NewNlbPlatform returns the platform for a discovered NLB load balancer asset.
func NewNlbPlatform(lbID string) *inventory.Platform {
	return newPlatform("alicloud-nlb-loadbalancer", "",
		[]string{"technology=alicloud", "kind=nlb-loadbalancer", "loadbalancer=" + lbID})
}

// NewVpcPlatform returns the platform for a discovered VPC asset.
func NewVpcPlatform(vpcID string) *inventory.Platform {
	return newPlatform("alicloud-vpc", "",
		[]string{"technology=alicloud", "kind=vpc", "vpc=" + vpcID})
}

// NewWafInstancePlatform returns the platform for a discovered WAF instance asset.
func NewWafInstancePlatform(instanceID string) *inventory.Platform {
	return newPlatform("alicloud-waf-instance", "",
		[]string{"technology=alicloud", "kind=waf-instance", "instance=" + instanceID})
}

// NewCloudFirewallPlatform returns the platform for the account's Cloud Firewall asset.
func NewCloudFirewallPlatform(accountID string) *inventory.Platform {
	return newPlatform("alicloud-cloud-firewall", "",
		[]string{"technology=alicloud", "kind=cloud-firewall", "account=" + accountID})
}

func NewAccountIdentifier(accountID string) string       { return platformIDAccount + accountID }
func NewAckClusterIdentifier(clusterID string) string    { return platformIDAckCluster + clusterID }
func NewAlbIdentifier(lbID string) string                { return platformIDAlb + lbID }
func NewNlbIdentifier(lbID string) string                { return platformIDNlb + lbID }
func NewVpcIdentifier(vpcID string) string               { return platformIDVpc + vpcID }
func NewWafInstanceIdentifier(instanceID string) string  { return platformIDWafInstance + instanceID }
func NewCloudFirewallIdentifier(accountID string) string { return platformIDCloudFirewall + accountID }
