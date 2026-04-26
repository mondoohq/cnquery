// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// resolveVpcRef resolves a VPC by ID via the parent digitalocean resource so
// the firewall/droplet/database results share the same cached VPC list (avoids
// redoing client.VPCs.List per accessor). It owns the StateIsSet|StateIsNull
// bookkeeping on the target field so callers can't forget it.
func resolveVpcRef(runtime *plugin.Runtime, target *plugin.TValue[*mqlDigitaloceanVpc], vpcID string) (*mqlDigitaloceanVpc, error) {
	if strings.TrimSpace(vpcID) == "" {
		target.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	parent, err := CreateResource(runtime, "digitalocean", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	parentRes := parent.(*mqlDigitalocean)
	vpcs := parentRes.GetVpcs()
	if vpcs.Error != nil {
		return nil, vpcs.Error
	}
	for _, v := range vpcs.Data {
		mqlVpc := v.(*mqlDigitaloceanVpc)
		if mqlVpc.Id.Data == vpcID {
			return mqlVpc, nil
		}
	}
	target.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

// listAllDroplets returns the cached list of all droplets resolved via the
// parent digitalocean resource (single API pagination across multiple
// accessors).
func listAllDroplets(runtime *plugin.Runtime) ([]any, error) {
	parent, err := CreateResource(runtime, "digitalocean", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	parentRes := parent.(*mqlDigitalocean)
	droplets := parentRes.GetDroplets()
	if droplets.Error != nil {
		return nil, droplets.Error
	}
	return droplets.Data, nil
}

// listAllFirewalls returns the cached list of all firewalls resolved via the
// parent digitalocean resource.
func listAllFirewalls(runtime *plugin.Runtime) ([]any, error) {
	parent, err := CreateResource(runtime, "digitalocean", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	parentRes := parent.(*mqlDigitalocean)
	firewalls := parentRes.GetFirewalls()
	if firewalls.Error != nil {
		return nil, firewalls.Error
	}
	return firewalls.Data, nil
}

// --- Droplet typed refs / computed fields ---

func (r *mqlDigitaloceanDroplet) vpc() (*mqlDigitaloceanVpc, error) {
	return resolveVpcRef(r.MqlRuntime, &r.Vpc, r.VpcUuid.Data)
}

// firewallCoversDroplet reports whether the given firewall covers this droplet
// either by direct droplet-id assignment or by tag intersection.
func firewallCoversDroplet(fw *mqlDigitaloceanFirewall, dropletID int64, dropletTags []any) bool {
	for _, id := range fw.DropletIds.Data {
		if i, ok := id.(int64); ok && i == dropletID {
			return true
		}
	}
	if len(fw.Tags.Data) == 0 || len(dropletTags) == 0 {
		return false
	}
	tagSet := make(map[string]struct{}, len(dropletTags))
	for _, t := range dropletTags {
		if s, ok := t.(string); ok {
			tagSet[s] = struct{}{}
		}
	}
	for _, t := range fw.Tags.Data {
		if s, ok := t.(string); ok {
			if _, ok := tagSet[s]; ok {
				return true
			}
		}
	}
	return false
}

func (r *mqlDigitaloceanDroplet) firewalls() ([]any, error) {
	all, err := listAllFirewalls(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	var out []any
	for _, f := range all {
		fw := f.(*mqlDigitaloceanFirewall)
		if firewallCoversDroplet(fw, r.Id.Data, r.Tags.Data) {
			out = append(out, fw)
		}
	}
	return out, nil
}

func (r *mqlDigitaloceanDroplet) unprotectedPublicIp() (bool, error) {
	if r.PublicIpv4.Data == "" {
		return false, nil
	}
	covers := r.GetFirewalls()
	if covers.Error != nil {
		return false, covers.Error
	}
	return len(covers.Data) == 0, nil
}

// --- Firewall typed refs ---

func (r *mqlDigitaloceanFirewall) droplets() ([]any, error) {
	all, err := listAllDroplets(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	// Build a set of droplet IDs for fast direct-id matches
	directIDs := make(map[int64]struct{}, len(r.DropletIds.Data))
	for _, id := range r.DropletIds.Data {
		if i, ok := id.(int64); ok {
			directIDs[i] = struct{}{}
		}
	}
	tagSet := make(map[string]struct{}, len(r.Tags.Data))
	for _, t := range r.Tags.Data {
		if s, ok := t.(string); ok {
			tagSet[s] = struct{}{}
		}
	}

	var out []any
	for _, d := range all {
		dr := d.(*mqlDigitaloceanDroplet)
		if _, ok := directIDs[dr.Id.Data]; ok {
			out = append(out, dr)
			continue
		}
		if len(tagSet) > 0 {
			matched := false
			for _, t := range dr.Tags.Data {
				if s, ok := t.(string); ok {
					if _, ok := tagSet[s]; ok {
						matched = true
						break
					}
				}
			}
			if matched {
				out = append(out, dr)
			}
		}
	}
	return out, nil
}

// --- Database typed refs ---

func (r *mqlDigitaloceanDatabase) vpc() (*mqlDigitaloceanVpc, error) {
	return resolveVpcRef(r.MqlRuntime, &r.Vpc, r.PrivateNetworkUuid.Data)
}

// --- LoadBalancer typed refs ---

func (r *mqlDigitaloceanLoadBalancer) vpc() (*mqlDigitaloceanVpc, error) {
	return resolveVpcRef(r.MqlRuntime, &r.Vpc, r.VpcUuid.Data)
}

func (r *mqlDigitaloceanLoadBalancer) droplets() ([]any, error) {
	all, err := listAllDroplets(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	wanted := make(map[int64]struct{}, len(r.DropletIds.Data))
	for _, id := range r.DropletIds.Data {
		if i, ok := id.(int64); ok {
			wanted[i] = struct{}{}
		}
	}
	if len(wanted) == 0 {
		return []any{}, nil
	}
	var out []any
	for _, d := range all {
		dr := d.(*mqlDigitaloceanDroplet)
		if _, ok := wanted[dr.Id.Data]; ok {
			out = append(out, dr)
		}
	}
	return out, nil
}

// --- Volume typed refs ---

func (r *mqlDigitaloceanVolume) droplets() ([]any, error) {
	all, err := listAllDroplets(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	wanted := make(map[int64]struct{}, len(r.DropletIds.Data))
	for _, id := range r.DropletIds.Data {
		if i, ok := id.(int64); ok {
			wanted[i] = struct{}{}
		}
	}
	if len(wanted) == 0 {
		return []any{}, nil
	}
	var out []any
	for _, d := range all {
		dr := d.(*mqlDigitaloceanDroplet)
		if _, ok := wanted[dr.Id.Data]; ok {
			out = append(out, dr)
		}
	}
	return out, nil
}

// --- Kubernetes typed refs ---

func (r *mqlDigitaloceanKubernetesCluster) vpc() (*mqlDigitaloceanVpc, error) {
	return resolveVpcRef(r.MqlRuntime, &r.Vpc, r.VpcUuid.Data)
}
