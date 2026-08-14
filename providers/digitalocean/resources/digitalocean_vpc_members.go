// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// Reverse edges from digitalocean.vpc back to the resources that sit on it.
//
// Each child already carries a forward reference to its VPC, so the reverse
// direction is a regrouping of collections the parent resource has usually
// fetched anyway. Every index below is built once from the parent's own
// accessor, which means asking all VPCs what they hold costs the same single
// list call as asking one.

// groupByVpcID buckets already-fetched resources by the VPC ids each one
// reports. A resource attached to several VPCs lands under each of them, and
// one reporting no VPC is dropped rather than filed under the empty id.
func groupByVpcID(items []any, vpcIDsOf func(any) []string) map[string][]any {
	idx := make(map[string][]any, len(items))
	for _, item := range items {
		for _, id := range vpcIDsOf(item) {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			idx[id] = append(idx[id], item)
		}
	}
	return idx
}

// vpcMemberList selects one VPC's members out of a reverse index. The second
// return is false when the answer is unknown rather than empty: the VPC has
// no id to match on, or the collection the index is built from could not be
// read. Callers turn that into a null field, because an empty list has to
// keep meaning "nothing sits on this VPC".
func vpcMemberList(idx map[string][]any, idxErr error, vpcID string) ([]any, bool) {
	id := strings.TrimSpace(vpcID)
	if id == "" || idxErr != nil {
		return nil, false
	}
	found := idx[id]
	out := make([]any, len(found))
	copy(out, found)
	return out, true
}

// vpcMembers answers one reverse edge, resolving the index through the
// account-level parent so the underlying collection is fetched at most once
// for the whole scan.
func vpcMembers(
	vpc *mqlDigitaloceanVpc,
	target *plugin.TValue[[]any],
	index func(*mqlDigitalocean) (map[string][]any, error),
) ([]any, error) {
	var (
		idx    map[string][]any
		idxErr error
	)
	if strings.TrimSpace(vpc.Id.Data) != "" {
		parent, err := parentDigitalocean(vpc.MqlRuntime)
		if err != nil {
			return nil, err
		}
		idx, idxErr = index(parent)
	}
	members, ok := vpcMemberList(idx, idxErr, vpc.Id.Data)
	if !ok {
		target.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return members, nil
}

// stringsFromRawList narrows a runtime []string field, whose elements arrive
// as any, dropping anything that is not a string.
func stringsFromRawList(raw []any) []string {
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// --- Per-collection VPC id extractors ---

func dropletVpcIDs(v any) []string {
	d, ok := v.(*mqlDigitaloceanDroplet)
	if !ok {
		return nil
	}
	return []string{d.VpcUuid.Data}
}

func databaseVpcIDs(v any) []string {
	d, ok := v.(*mqlDigitaloceanDatabase)
	if !ok {
		return nil
	}
	return []string{d.PrivateNetworkUuid.Data}
}

func loadBalancerVpcIDs(v any) []string {
	lb, ok := v.(*mqlDigitaloceanLoadBalancer)
	if !ok {
		return nil
	}
	return []string{lb.VpcUuid.Data}
}

func kubernetesClusterVpcIDs(v any) []string {
	c, ok := v.(*mqlDigitaloceanKubernetesCluster)
	if !ok {
		return nil
	}
	return []string{c.VpcUuid.Data}
}

func nfsShareVpcIDs(v any) []string {
	n, ok := v.(*mqlDigitaloceanNfs)
	if !ok {
		return nil
	}
	return stringsFromRawList(n.VpcIds.Data)
}

func natGatewayVpcIDs(v any) []string {
	g, ok := v.(*mqlDigitaloceanVpcNatGateway)
	if !ok {
		return nil
	}
	return g.vpcUUIDs
}

// --- Memoized reverse indexes on the account-level parent ---

func (r *mqlDigitalocean) dropletsByVpc() (map[string][]any, error) {
	r.vpcDropletIndexOnce.Do(func() {
		droplets := r.GetDroplets()
		if droplets.Error != nil {
			r.vpcDropletIndexErr = droplets.Error
			return
		}
		r.vpcDropletIndex = groupByVpcID(droplets.Data, dropletVpcIDs)
	})
	return r.vpcDropletIndex, r.vpcDropletIndexErr
}

func (r *mqlDigitalocean) databasesByVpc() (map[string][]any, error) {
	r.vpcDatabaseIndexOnce.Do(func() {
		databases := r.GetDatabases()
		if databases.Error != nil {
			r.vpcDatabaseIndexErr = databases.Error
			return
		}
		r.vpcDatabaseIndex = groupByVpcID(databases.Data, databaseVpcIDs)
	})
	return r.vpcDatabaseIndex, r.vpcDatabaseIndexErr
}

func (r *mqlDigitalocean) loadBalancersByVpc() (map[string][]any, error) {
	r.vpcLoadBalancerIndexOnce.Do(func() {
		lbs := r.GetLoadBalancers()
		if lbs.Error != nil {
			r.vpcLoadBalancerIndexErr = lbs.Error
			return
		}
		r.vpcLoadBalancerIndex = groupByVpcID(lbs.Data, loadBalancerVpcIDs)
	})
	return r.vpcLoadBalancerIndex, r.vpcLoadBalancerIndexErr
}

func (r *mqlDigitalocean) kubernetesClustersByVpc() (map[string][]any, error) {
	r.vpcK8sClusterIndexOnce.Do(func() {
		clusters := r.GetKubernetesClusters()
		if clusters.Error != nil {
			r.vpcK8sClusterIndexErr = clusters.Error
			return
		}
		r.vpcK8sClusterIndex = groupByVpcID(clusters.Data, kubernetesClusterVpcIDs)
	})
	return r.vpcK8sClusterIndex, r.vpcK8sClusterIndexErr
}

func (r *mqlDigitalocean) nfsSharesByVpc() (map[string][]any, error) {
	r.vpcNfsShareIndexOnce.Do(func() {
		shares := r.GetNfsShares()
		if shares.Error != nil {
			r.vpcNfsShareIndexErr = shares.Error
			return
		}
		r.vpcNfsShareIndex = groupByVpcID(shares.Data, nfsShareVpcIDs)
	})
	return r.vpcNfsShareIndex, r.vpcNfsShareIndexErr
}

func (r *mqlDigitalocean) natGatewaysByVpc() (map[string][]any, error) {
	r.vpcNatGatewayIndexOnce.Do(func() {
		gateways := r.GetVpcNatGateways()
		if gateways.Error != nil {
			r.vpcNatGatewayIndexErr = gateways.Error
			return
		}
		r.vpcNatGatewayIndex = groupByVpcID(gateways.Data, natGatewayVpcIDs)
	})
	return r.vpcNatGatewayIndex, r.vpcNatGatewayIndexErr
}

// --- Reverse-edge accessors ---

func (r *mqlDigitaloceanVpc) droplets() ([]any, error) {
	return vpcMembers(r, &r.Droplets, (*mqlDigitalocean).dropletsByVpc)
}

func (r *mqlDigitaloceanVpc) databases() ([]any, error) {
	return vpcMembers(r, &r.Databases, (*mqlDigitalocean).databasesByVpc)
}

func (r *mqlDigitaloceanVpc) loadBalancers() ([]any, error) {
	return vpcMembers(r, &r.LoadBalancers, (*mqlDigitalocean).loadBalancersByVpc)
}

func (r *mqlDigitaloceanVpc) kubernetesClusters() ([]any, error) {
	return vpcMembers(r, &r.KubernetesClusters, (*mqlDigitalocean).kubernetesClustersByVpc)
}

func (r *mqlDigitaloceanVpc) nfsShares() ([]any, error) {
	return vpcMembers(r, &r.NfsShares, (*mqlDigitalocean).nfsSharesByVpc)
}

func (r *mqlDigitaloceanVpc) natGateways() ([]any, error) {
	return vpcMembers(r, &r.NatGateways, (*mqlDigitalocean).natGatewaysByVpc)
}
