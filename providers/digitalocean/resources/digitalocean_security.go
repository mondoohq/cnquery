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

	projectIndexOnce sync.Once
	projectIndex     map[string]*mqlDigitaloceanProject
	projectIndexErr  error

	databaseIndexOnce sync.Once
	databaseIndex     map[string]*mqlDigitaloceanDatabase
	databaseIndexErr  error

	vectorDatabaseIndexOnce sync.Once
	vectorDatabaseIndex     map[string]*mqlDigitaloceanVectorDatabase
	vectorDatabaseIndexErr  error

	certificateIndexOnce sync.Once
	certificateIndex     map[string]*mqlDigitaloceanCertificate
	certificateIndexErr  error

	k8sClusterIndexOnce sync.Once
	k8sClusterIndex     map[string]*mqlDigitaloceanKubernetesCluster
	k8sClusterIndexErr  error

	loadBalancerIndexOnce sync.Once
	loadBalancerIndex     map[string]*mqlDigitaloceanLoadBalancer
	loadBalancerIndexErr  error

	volumeIndexOnce sync.Once
	volumeIndex     map[string]*mqlDigitaloceanVolume
	volumeIndexErr  error

	snapshotIndexOnce sync.Once
	snapshotIndex     map[string]*mqlDigitaloceanSnapshot
	snapshotIndexErr  error

	imageIndexOnce sync.Once
	imageIndex     map[int64]*mqlDigitaloceanImage
	imageIndexErr  error

	firewallIndexOnce sync.Once
	firewallByDroplet map[int64][]*mqlDigitaloceanFirewall
	firewallByTag     map[string][]*mqlDigitaloceanFirewall
	firewallIndexErr  error

	partnerAttachmentIndexOnce sync.Once
	partnerAttachmentIndex     map[string]*mqlDigitaloceanPartnerAttachment
	partnerAttachmentIndexErr  error
}

// partnerAttachmentByID resolves a partner attachment by its ID from the
// account-wide list, caching the index. Partner attachments reference
// their parent and children by UUID, which matches the attachment ID.
func (r *mqlDigitalocean) partnerAttachmentByID(id string) (*mqlDigitaloceanPartnerAttachment, error) {
	r.partnerAttachmentIndexOnce.Do(func() {
		attachments := r.GetPartnerAttachments()
		if attachments.Error != nil {
			r.partnerAttachmentIndexErr = attachments.Error
			return
		}
		idx := make(map[string]*mqlDigitaloceanPartnerAttachment, len(attachments.Data))
		for _, a := range attachments.Data {
			ma := a.(*mqlDigitaloceanPartnerAttachment)
			idx[ma.Id.Data] = ma
		}
		r.partnerAttachmentIndex = idx
	})
	if r.partnerAttachmentIndexErr != nil {
		return nil, r.partnerAttachmentIndexErr
	}
	return r.partnerAttachmentIndex[id], nil
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

func (r *mqlDigitalocean) projectByID(id string) (*mqlDigitaloceanProject, error) {
	r.projectIndexOnce.Do(func() {
		projects := r.GetProjects()
		if projects.Error != nil {
			r.projectIndexErr = projects.Error
			return
		}
		idx := make(map[string]*mqlDigitaloceanProject, len(projects.Data))
		for _, p := range projects.Data {
			mp := p.(*mqlDigitaloceanProject)
			idx[mp.Id.Data] = mp
		}
		r.projectIndex = idx
	})
	if r.projectIndexErr != nil {
		return nil, r.projectIndexErr
	}
	return r.projectIndex[id], nil
}

func (r *mqlDigitalocean) databaseByID(id string) (*mqlDigitaloceanDatabase, error) {
	r.databaseIndexOnce.Do(func() {
		databases := r.GetDatabases()
		if databases.Error != nil {
			r.databaseIndexErr = databases.Error
			return
		}
		idx := make(map[string]*mqlDigitaloceanDatabase, len(databases.Data))
		for _, d := range databases.Data {
			md := d.(*mqlDigitaloceanDatabase)
			idx[md.Id.Data] = md
		}
		r.databaseIndex = idx
	})
	if r.databaseIndexErr != nil {
		return nil, r.databaseIndexErr
	}
	return r.databaseIndex[id], nil
}

func (r *mqlDigitalocean) vectorDatabaseByID(id string) (*mqlDigitaloceanVectorDatabase, error) {
	r.vectorDatabaseIndexOnce.Do(func() {
		vdbs := r.GetVectorDatabases()
		if vdbs.Error != nil {
			r.vectorDatabaseIndexErr = vdbs.Error
			return
		}
		idx := make(map[string]*mqlDigitaloceanVectorDatabase, len(vdbs.Data))
		for _, v := range vdbs.Data {
			mv := v.(*mqlDigitaloceanVectorDatabase)
			idx[mv.Id.Data] = mv
		}
		r.vectorDatabaseIndex = idx
	})
	if r.vectorDatabaseIndexErr != nil {
		return nil, r.vectorDatabaseIndexErr
	}
	return r.vectorDatabaseIndex[id], nil
}

func (r *mqlDigitalocean) certificateByID(id string) (*mqlDigitaloceanCertificate, error) {
	r.certificateIndexOnce.Do(func() {
		certs := r.GetCertificates()
		if certs.Error != nil {
			r.certificateIndexErr = certs.Error
			return
		}
		idx := make(map[string]*mqlDigitaloceanCertificate, len(certs.Data))
		for _, c := range certs.Data {
			mc := c.(*mqlDigitaloceanCertificate)
			idx[mc.Id.Data] = mc
		}
		r.certificateIndex = idx
	})
	if r.certificateIndexErr != nil {
		return nil, r.certificateIndexErr
	}
	return r.certificateIndex[id], nil
}

func (r *mqlDigitalocean) ensureK8sClusterIndex() error {
	r.k8sClusterIndexOnce.Do(func() {
		clusters := r.GetKubernetesClusters()
		if clusters.Error != nil {
			r.k8sClusterIndexErr = clusters.Error
			return
		}
		idx := make(map[string]*mqlDigitaloceanKubernetesCluster, len(clusters.Data))
		for _, c := range clusters.Data {
			mc := c.(*mqlDigitaloceanKubernetesCluster)
			idx[mc.Id.Data] = mc
		}
		r.k8sClusterIndex = idx
	})
	return r.k8sClusterIndexErr
}

func (r *mqlDigitalocean) kubernetesClusterByID(id string) (*mqlDigitaloceanKubernetesCluster, error) {
	if err := r.ensureK8sClusterIndex(); err != nil {
		return nil, err
	}
	return r.k8sClusterIndex[id], nil
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

func (r *mqlDigitalocean) loadBalancerByUIDs(uids []any) ([]any, error) {
	if len(uids) == 0 {
		return []any{}, nil
	}
	r.loadBalancerIndexOnce.Do(func() {
		lbs := r.GetLoadBalancers()
		if lbs.Error != nil {
			r.loadBalancerIndexErr = lbs.Error
			return
		}
		idx := make(map[string]*mqlDigitaloceanLoadBalancer, len(lbs.Data))
		for _, lb := range lbs.Data {
			mlb := lb.(*mqlDigitaloceanLoadBalancer)
			idx[mlb.Id.Data] = mlb
		}
		r.loadBalancerIndex = idx
	})
	if r.loadBalancerIndexErr != nil {
		return nil, r.loadBalancerIndexErr
	}
	out := make([]any, 0, len(uids))
	for _, uid := range uids {
		s, ok := uid.(string)
		if !ok {
			continue
		}
		if lb, ok := r.loadBalancerIndex[s]; ok {
			out = append(out, lb)
		}
	}
	return out, nil
}

func (r *mqlDigitalocean) volumesByIDs(ids []string) ([]any, error) {
	if len(ids) == 0 {
		return []any{}, nil
	}
	r.volumeIndexOnce.Do(func() {
		volumes := r.GetVolumes()
		if volumes.Error != nil {
			r.volumeIndexErr = volumes.Error
			return
		}
		idx := make(map[string]*mqlDigitaloceanVolume, len(volumes.Data))
		for _, v := range volumes.Data {
			mv := v.(*mqlDigitaloceanVolume)
			idx[mv.Id.Data] = mv
		}
		r.volumeIndex = idx
	})
	if r.volumeIndexErr != nil {
		return nil, r.volumeIndexErr
	}
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		if v, ok := r.volumeIndex[id]; ok {
			out = append(out, v)
		}
	}
	return out, nil
}

// snapshotsByIDs resolves snapshot string IDs to digitalocean.snapshot
// resources from the account-wide snapshot list, skipping any not found.
// A droplet snapshot's string id equals the image id it was saved as.
func (r *mqlDigitalocean) snapshotsByIDs(ids []string) ([]any, error) {
	if len(ids) == 0 {
		return []any{}, nil
	}
	r.snapshotIndexOnce.Do(func() {
		snapshots := r.GetSnapshots()
		if snapshots.Error != nil {
			r.snapshotIndexErr = snapshots.Error
			return
		}
		idx := make(map[string]*mqlDigitaloceanSnapshot, len(snapshots.Data))
		for _, s := range snapshots.Data {
			ms := s.(*mqlDigitaloceanSnapshot)
			idx[ms.Id.Data] = ms
		}
		r.snapshotIndex = idx
	})
	if r.snapshotIndexErr != nil {
		return nil, r.snapshotIndexErr
	}
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		if s, ok := r.snapshotIndex[id]; ok {
			out = append(out, s)
		}
	}
	return out, nil
}

// imagesByIDs resolves image IDs to digitalocean.image resources from the
// account's own image list (which includes backups and snapshots saved as
// images), skipping any not found.
func (r *mqlDigitalocean) imagesByIDs(ids []int) ([]any, error) {
	if len(ids) == 0 {
		return []any{}, nil
	}
	r.imageIndexOnce.Do(func() {
		images := r.GetImages()
		if images.Error != nil {
			r.imageIndexErr = images.Error
			return
		}
		idx := make(map[int64]*mqlDigitaloceanImage, len(images.Data))
		for _, img := range images.Data {
			mi := img.(*mqlDigitaloceanImage)
			idx[mi.Id.Data] = mi
		}
		r.imageIndex = idx
	})
	if r.imageIndexErr != nil {
		return nil, r.imageIndexErr
	}
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		if img, ok := r.imageIndex[int64(id)]; ok {
			out = append(out, img)
		}
	}
	return out, nil
}

func (r *mqlDigitalocean) kubernetesClustersByIDs(ids []any) ([]any, error) {
	if len(ids) == 0 {
		return []any{}, nil
	}
	if err := r.ensureK8sClusterIndex(); err != nil {
		return nil, err
	}
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		s, ok := id.(string)
		if !ok {
			continue
		}
		if c, ok := r.k8sClusterIndex[s]; ok {
			out = append(out, c)
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

// projectRef sets the StateIsSet|StateIsNull bookkeeping on the target
// field and resolves the project from the parent resource's cached index.
func projectRef(runtime *plugin.Runtime, projectID string, target *plugin.TValue[*mqlDigitaloceanProject]) (*mqlDigitaloceanProject, error) {
	if strings.TrimSpace(projectID) == "" {
		target.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	parent, err := parentDigitalocean(runtime)
	if err != nil {
		return nil, err
	}
	project, err := parent.projectByID(projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		target.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return project, nil
}

// dropletRef resolves a single droplet by id from the parent's cached
// index, setting StateIsSet|StateIsNull when the id is unset or unknown.
func dropletRef(runtime *plugin.Runtime, dropletID int64, target *plugin.TValue[*mqlDigitaloceanDroplet]) (*mqlDigitaloceanDroplet, error) {
	if dropletID == 0 {
		target.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	parent, err := parentDigitalocean(runtime)
	if err != nil {
		return nil, err
	}
	droplets, err := parent.dropletByIDs([]any{dropletID})
	if err != nil {
		return nil, err
	}
	if len(droplets) == 0 {
		target.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return droplets[0].(*mqlDigitaloceanDroplet), nil
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

// sqlMode returns the cluster's SQL mode flags. It is a MySQL-only setting,
// so for any other engine the field is resolved to null without an API call.
func (r *mqlDigitaloceanDatabase) sqlMode() (string, error) {
	if r.Engine.Data != "mysql" {
		r.SqlMode.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	mode, _, err := conn.Client().Databases.GetSQLMode(context.Background(), r.Id.Data)
	if err != nil {
		return "", err
	}
	return mode, nil
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
