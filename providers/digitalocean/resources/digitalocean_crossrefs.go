// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"strings"

	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// parseDropletID extracts a droplet id from either a bare numeric id
// ("12345") or a DigitalOcean droplet URN ("do:droplet:12345"). The
// second return is false when the string names something other than a
// droplet.
func parseDropletID(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if strings.HasPrefix(s, "do:") {
		parts := strings.Split(s, ":")
		if len(parts) != 3 || parts[1] != "droplet" {
			return 0, false
		}
		s = parts[2]
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

// volumeRef resolves a single volume by id from the parent's cached
// index, setting StateIsSet|StateIsNull when the id is unset or unknown.
func volumeRef(runtime *plugin.Runtime, id string, target *plugin.TValue[*mqlDigitaloceanVolume]) (*mqlDigitaloceanVolume, error) {
	if id == "" {
		target.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	parent, err := parentDigitalocean(runtime)
	if err != nil {
		return nil, err
	}
	volumes, err := parent.volumesByIDs([]string{id})
	if err != nil {
		return nil, err
	}
	if len(volumes) == 0 {
		target.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return volumes[0].(*mqlDigitaloceanVolume), nil
}

// ----- Database cluster project -----

func (r *mqlDigitaloceanDatabase) project() (*mqlDigitaloceanProject, error) {
	return projectRef(r.MqlRuntime, r.ProjectId.Data, &r.Project)
}

// ----- Snapshot source -----

func (r *mqlDigitaloceanSnapshot) droplet() (*mqlDigitaloceanDroplet, error) {
	if r.ResourceType.Data != "droplet" {
		r.Droplet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	id, ok := parseDropletID(r.ResourceId.Data)
	if !ok {
		r.Droplet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return dropletRef(r.MqlRuntime, id, &r.Droplet)
}

func (r *mqlDigitaloceanSnapshot) volume() (*mqlDigitaloceanVolume, error) {
	if r.ResourceType.Data != "volume" {
		r.Volume.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return volumeRef(r.MqlRuntime, r.ResourceId.Data, &r.Volume)
}

// ----- Alert policy targets -----

// droplets resolves the policy's entity IDs to droplets. Droplet alert
// types carry droplet IDs (bare or as a do:droplet: URN); policies that
// target other entity kinds resolve to an empty list.
func (r *mqlDigitaloceanAlertPolicy) droplets() ([]any, error) {
	ids := make([]any, 0, len(r.Entities.Data))
	for _, e := range r.Entities.Data {
		s, ok := e.(string)
		if !ok {
			continue
		}
		if id, ok := parseDropletID(s); ok {
			ids = append(ids, id)
		}
	}
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return parent.dropletByIDs(ids)
}

// ----- BYOIP prefix resource assignment -----

func (r *mqlDigitaloceanByoipPrefixResource) droplet() (*mqlDigitaloceanDroplet, error) {
	id, ok := parseDropletID(r.Resource.Data)
	if !ok {
		r.Droplet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return dropletRef(r.MqlRuntime, id, &r.Droplet)
}

// ----- VPC peering -----

func (r *mqlDigitaloceanVpcPeering) vpcs() ([]interface{}, error) {
	uuids := make([]string, 0, len(r.VpcIds.Data))
	for _, v := range r.VpcIds.Data {
		if s, ok := v.(string); ok {
			uuids = append(uuids, s)
		}
	}
	return vpcRefsByUUIDs(r.MqlRuntime, uuids)
}

// ----- Kubernetes node pool -----

func (r *mqlDigitaloceanKubernetesNode) droplet() (*mqlDigitaloceanDroplet, error) {
	// The API returns the backing droplet's ID as a numeric string; it is
	// empty until the node's droplet has been provisioned.
	dropletID, err := strconv.ParseInt(r.cacheDropletID, 10, 64)
	if err != nil || dropletID == 0 {
		r.Droplet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return dropletRef(r.MqlRuntime, dropletID, &r.Droplet)
}

func (r *mqlDigitaloceanKubernetesNodePool) cluster() (*mqlDigitaloceanKubernetesCluster, error) {
	if r.ClusterId.Data == "" {
		r.Cluster.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	cluster, err := parent.kubernetesClusterByID(r.ClusterId.Data)
	if err != nil {
		return nil, err
	}
	if cluster == nil {
		r.Cluster.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return cluster, nil
}

// ----- Droplet -----

func (r *mqlDigitaloceanDroplet) volumes() ([]interface{}, error) {
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return parent.volumesByIDs(r.cacheVolumeIDs)
}

func (r *mqlDigitaloceanDroplet) snapshots() ([]interface{}, error) {
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(r.cacheSnapshotIDs))
	for i, id := range r.cacheSnapshotIDs {
		ids[i] = strconv.Itoa(id)
	}
	return parent.snapshotsByIDs(ids)
}

func (r *mqlDigitaloceanDroplet) backups() ([]interface{}, error) {
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return parent.imagesByIDs(r.cacheBackupIDs)
}

// ----- Load balancer -----

func (r *mqlDigitaloceanLoadBalancer) project() (*mqlDigitaloceanProject, error) {
	return projectRef(r.MqlRuntime, r.ProjectId.Data, &r.Project)
}

func (r *mqlDigitaloceanLoadBalancer) targetLoadBalancers() ([]interface{}, error) {
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	ids := make([]any, len(r.cacheTargetLoadBalancerIDs))
	for i, id := range r.cacheTargetLoadBalancerIDs {
		ids[i] = id
	}
	return parent.loadBalancerByUIDs(ids)
}

// ----- App Platform -----

func (r *mqlDigitaloceanApp) project() (*mqlDigitaloceanProject, error) {
	return projectRef(r.MqlRuntime, r.ProjectId.Data, &r.Project)
}

// ----- Reserved IP (v4) -----

func (r *mqlDigitaloceanReservedIp) project() (*mqlDigitaloceanProject, error) {
	return projectRef(r.MqlRuntime, r.ProjectId.Data, &r.Project)
}

func (r *mqlDigitaloceanReservedIp) droplet() (*mqlDigitaloceanDroplet, error) {
	return dropletRef(r.MqlRuntime, r.DropletId.Data, &r.Droplet)
}

// ----- CDN -----

func (r *mqlDigitaloceanCdn) certificate() (*mqlDigitaloceanCertificate, error) {
	if r.CertificateId.Data == "" {
		r.Certificate.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	cert, err := parent.certificateByID(r.CertificateId.Data)
	if err != nil {
		return nil, err
	}
	if cert == nil {
		r.Certificate.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return cert, nil
}

// ----- Database sub-resource back-references -----

func databaseClusterRef(runtime *plugin.Runtime, id string, target *plugin.TValue[*mqlDigitaloceanDatabase]) (*mqlDigitaloceanDatabase, error) {
	if id == "" {
		target.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	parent, err := parentDigitalocean(runtime)
	if err != nil {
		return nil, err
	}
	db, err := parent.databaseByID(id)
	if err != nil {
		return nil, err
	}
	if db == nil {
		target.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return db, nil
}

func (r *mqlDigitaloceanDatabaseBackup) database() (*mqlDigitaloceanDatabase, error) {
	return databaseClusterRef(r.MqlRuntime, r.DatabaseId.Data, &r.Database)
}

func (r *mqlDigitaloceanDatabaseUser) database() (*mqlDigitaloceanDatabase, error) {
	return databaseClusterRef(r.MqlRuntime, r.DatabaseId.Data, &r.Database)
}

func (r *mqlDigitaloceanDatabaseReplica) database() (*mqlDigitaloceanDatabase, error) {
	return databaseClusterRef(r.MqlRuntime, r.DatabaseId.Data, &r.Database)
}

// pool exposes the parent cluster as databaseCluster() because its
// `database` field already names the logical database the pool connects to.
func (r *mqlDigitaloceanDatabasePool) databaseCluster() (*mqlDigitaloceanDatabase, error) {
	return databaseClusterRef(r.MqlRuntime, r.DatabaseId.Data, &r.DatabaseCluster)
}

// ----- Vector database backup back-reference -----

func (r *mqlDigitaloceanVectorDatabaseBackup) vectorDatabase() (*mqlDigitaloceanVectorDatabase, error) {
	if r.VectorDatabaseId.Data == "" {
		r.VectorDatabase.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	vdb, err := parent.vectorDatabaseByID(r.VectorDatabaseId.Data)
	if err != nil {
		return nil, err
	}
	if vdb == nil {
		r.VectorDatabase.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return vdb, nil
}
