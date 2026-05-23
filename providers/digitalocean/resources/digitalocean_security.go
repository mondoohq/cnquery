// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strings"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/digitalocean/connection"
)

// mqlDigitaloceanInternal holds parent-resource caches that let
// typed-ref accessors avoid re-scanning the full droplet / firewall /
// vpc list on every call. Indexes are built lazily on first access
// via sync.Once.
type mqlDigitaloceanInternal struct {
	vpcIndexOnce sync.Once
	vpcIndex     map[string]*mqlDigitaloceanVpc
	vpcIndexErr  error

	dropletIndexOnce sync.Once
	dropletIndex     map[int64]*mqlDigitaloceanDroplet
	dropletIndexErr  error

	firewallIndexOnce sync.Once
	firewallByDroplet map[int64][]*mqlDigitaloceanFirewall
	firewallByTag     map[string][]*mqlDigitaloceanFirewall
	firewallIndexErr  error
}

func parentDigitalocean(runtime *plugin.Runtime) (*mqlDigitalocean, error) {
	parent, err := CreateResource(runtime, "digitalocean", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	return parent.(*mqlDigitalocean), nil
}

func (r *mqlDigitalocean) vpcByID(uuid string) (*mqlDigitaloceanVpc, error) {
	r.vpcIndexOnce.Do(func() {
		vpcs := r.GetVpcs()
		if vpcs.Error != nil {
			r.vpcIndexErr = vpcs.Error
			return
		}
		idx := make(map[string]*mqlDigitaloceanVpc, len(vpcs.Data))
		for _, v := range vpcs.Data {
			mv := v.(*mqlDigitaloceanVpc)
			idx[mv.Id.Data] = mv
		}
		r.vpcIndex = idx
	})
	if r.vpcIndexErr != nil {
		return nil, r.vpcIndexErr
	}
	return r.vpcIndex[uuid], nil
}

func (r *mqlDigitalocean) dropletByIDs(ids []any) ([]any, error) {
	if len(ids) == 0 {
		return []any{}, nil
	}
	r.dropletIndexOnce.Do(func() {
		droplets := r.GetDroplets()
		if droplets.Error != nil {
			r.dropletIndexErr = droplets.Error
			return
		}
		idx := make(map[int64]*mqlDigitaloceanDroplet, len(droplets.Data))
		for _, d := range droplets.Data {
			md := d.(*mqlDigitaloceanDroplet)
			idx[md.Id.Data] = md
		}
		r.dropletIndex = idx
	})
	if r.dropletIndexErr != nil {
		return nil, r.dropletIndexErr
	}
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		i, ok := id.(int64)
		if !ok {
			continue
		}
		if d, ok := r.dropletIndex[i]; ok {
			out = append(out, d)
		}
	}
	return out, nil
}

// firewallsCovering returns firewalls that cover a given droplet via
// direct droplet-id assignment or tag intersection. Both indexes are
// built once on first call so subsequent droplet→firewall lookups are
// O(droplet_tags + matches) instead of O(firewalls × droplet_tags).
func (r *mqlDigitalocean) firewallsCovering(dropletID int64, dropletTags []any) ([]any, error) {
	r.firewallIndexOnce.Do(func() {
		fws := r.GetFirewalls()
		if fws.Error != nil {
			r.firewallIndexErr = fws.Error
			return
		}
		byDroplet := map[int64][]*mqlDigitaloceanFirewall{}
		byTag := map[string][]*mqlDigitaloceanFirewall{}
		for _, f := range fws.Data {
			fw := f.(*mqlDigitaloceanFirewall)
			for _, id := range fw.DropletIds.Data {
				if i, ok := id.(int64); ok {
					byDroplet[i] = append(byDroplet[i], fw)
				}
			}
			for _, t := range fw.Tags.Data {
				if s, ok := t.(string); ok {
					byTag[s] = append(byTag[s], fw)
				}
			}
		}
		r.firewallByDroplet = byDroplet
		r.firewallByTag = byTag
	})
	if r.firewallIndexErr != nil {
		return nil, r.firewallIndexErr
	}

	seen := map[*mqlDigitaloceanFirewall]struct{}{}
	out := make([]any, 0)
	for _, fw := range r.firewallByDroplet[dropletID] {
		if _, ok := seen[fw]; ok {
			continue
		}
		seen[fw] = struct{}{}
		out = append(out, fw)
	}
	for _, t := range dropletTags {
		s, ok := t.(string)
		if !ok {
			continue
		}
		for _, fw := range r.firewallByTag[s] {
			if _, ok := seen[fw]; ok {
				continue
			}
			seen[fw] = struct{}{}
			out = append(out, fw)
		}
	}
	return out, nil
}

// resolveVpcRef sets the StateIsSet|StateIsNull bookkeeping on the
// target field so callers can't forget it. The VPC lookup is served
// from the parent resource's cached index.
func resolveVpcRef(runtime *plugin.Runtime, target *plugin.TValue[*mqlDigitaloceanVpc], vpcID string) (*mqlDigitaloceanVpc, error) {
	if strings.TrimSpace(vpcID) == "" {
		target.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	parent, err := parentDigitalocean(runtime)
	if err != nil {
		return nil, err
	}
	vpc, err := parent.vpcByID(vpcID)
	if err != nil {
		return nil, err
	}
	if vpc == nil {
		target.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return vpc, nil
}

// --- Droplet typed refs / computed fields ---

func (r *mqlDigitaloceanDroplet) vpc() (*mqlDigitaloceanVpc, error) {
	return resolveVpcRef(r.MqlRuntime, &r.Vpc, r.VpcUuid.Data)
}

// baseImage resolves the droplet's image to a typed digitalocean.image. The
// godo image is cached from the droplet list response, so no refetch happens.
func (r *mqlDigitaloceanDroplet) baseImage() (*mqlDigitaloceanImage, error) {
	if r.image == nil || r.image.ID == 0 {
		r.BaseImage.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return newMqlDigitaloceanImage(r.MqlRuntime, *r.image)
}

func (r *mqlDigitaloceanDroplet) firewalls() ([]any, error) {
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return parent.firewallsCovering(r.Id.Data, r.Tags.Data)
}

// dropletHasPublicAddress reports whether a droplet has any public-facing
// IP — IPv4 or IPv6. Extracted so the IPv6 branch is unit-testable
// without spinning up a full plugin runtime.
func dropletHasPublicAddress(publicIPv4, publicIPv6 string) bool {
	return publicIPv4 != "" || publicIPv6 != ""
}

// missingFirewall reports whether the droplet has any public-facing
// IP address (IPv4 or IPv6) yet no firewall covering it. Internal-only
// droplets are not considered missing-firewall regardless of coverage.
func (r *mqlDigitaloceanDroplet) missingFirewall() (bool, error) {
	if !dropletHasPublicAddress(r.PublicIpv4.Data, r.PublicIpv6.Data) {
		return false, nil
	}
	covers := r.GetFirewalls()
	if covers.Error != nil {
		return false, covers.Error
	}
	return len(covers.Data) == 0, nil
}

// --- Firewall typed refs ---

// droplets returns the droplets a firewall covers, either by direct
// droplet-id assignment or by tag intersection.
func (r *mqlDigitaloceanFirewall) droplets() ([]any, error) {
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	direct, err := parent.dropletByIDs(r.DropletIds.Data)
	if err != nil {
		return nil, err
	}

	tagSet := make(map[string]struct{}, len(r.Tags.Data))
	for _, t := range r.Tags.Data {
		if s, ok := t.(string); ok {
			tagSet[s] = struct{}{}
		}
	}
	if len(tagSet) == 0 {
		return direct, nil
	}

	droplets := parent.GetDroplets()
	if droplets.Error != nil {
		return nil, droplets.Error
	}

	seen := make(map[int64]struct{}, len(direct))
	for _, d := range direct {
		seen[d.(*mqlDigitaloceanDroplet).Id.Data] = struct{}{}
	}
	out := append([]any(nil), direct...)
	for _, d := range droplets.Data {
		dr := d.(*mqlDigitaloceanDroplet)
		if _, ok := seen[dr.Id.Data]; ok {
			continue
		}
		for _, t := range dr.Tags.Data {
			s, ok := t.(string)
			if !ok {
				continue
			}
			if _, ok := tagSet[s]; ok {
				out = append(out, dr)
				seen[dr.Id.Data] = struct{}{}
				break
			}
		}
	}
	return out, nil
}

// --- Database typed refs ---

func (r *mqlDigitaloceanDatabase) vpc() (*mqlDigitaloceanVpc, error) {
	return resolveVpcRef(r.MqlRuntime, &r.Vpc, r.PrivateNetworkUuid.Data)
}

// evictionPolicy returns the key eviction policy for Redis/Valkey clusters.
// Other engines have no eviction policy — the field is marked null on those
// so users can filter with `where(evictionPolicy != null)`.
//
// This issues one GetEvictionPolicy call per Redis/Valkey cluster — DigitalOcean
// exposes no batch endpoint — so querying it across many cache clusters results
// in N serial API calls. Non-cache engines short-circuit before any call.
func (r *mqlDigitaloceanDatabase) evictionPolicy() (string, error) {
	if engine := r.Engine.Data; engine != "redis" && engine != "valkey" {
		r.EvictionPolicy.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	policy, _, err := conn.Client().Databases.GetEvictionPolicy(context.Background(), r.Id.Data)
	if err != nil {
		return "", err
	}
	return policy, nil
}

// --- LoadBalancer typed refs ---

func (r *mqlDigitaloceanLoadBalancer) vpc() (*mqlDigitaloceanVpc, error) {
	return resolveVpcRef(r.MqlRuntime, &r.Vpc, r.VpcUuid.Data)
}

func (r *mqlDigitaloceanLoadBalancer) droplets() ([]any, error) {
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return parent.dropletByIDs(r.DropletIds.Data)
}

// --- Volume typed refs ---

func (r *mqlDigitaloceanVolume) droplets() ([]any, error) {
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return parent.dropletByIDs(r.DropletIds.Data)
}

// --- Kubernetes typed refs ---

func (r *mqlDigitaloceanKubernetesCluster) vpc() (*mqlDigitaloceanVpc, error) {
	return resolveVpcRef(r.MqlRuntime, &r.Vpc, r.VpcUuid.Data)
}
