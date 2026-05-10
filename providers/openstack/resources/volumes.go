// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// ---- openstack.blockstorage.volume ----

type mqlOpenstackBlockstorageVolumeInternal struct {
	cacheProjectID        string
	cacheUserID           string
	cacheSourceVolumeID   string
	cacheSourceSnapshotID string
	cacheServerIDs        []string
}

func (r *mqlOpenstackBlockstorageVolume) id() (string, error) {
	return "openstack.blockstorage.volume/" + r.Id.Data, nil
}

func initOpenstackBlockstorageVolume(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := stringArg(args, "id")
	if !ok || id == "" {
		return args, nil, nil
	}
	root, err := CreateResource(runtime, "openstack", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	list := root.(*mqlOpenstack).GetVolumes()
	if list.Error == nil {
		for _, raw := range list.Data {
			v := raw.(*mqlOpenstackBlockstorageVolume)
			if v.Id.Data == id {
				return args, v, nil
			}
		}
	}
	initSyntheticID("openstack.blockstorage.volume", "id", args)
	return args, nil, nil
}

func (o *mqlOpenstack) volumes() ([]any, error) {
	c := conn(o.MqlRuntime)
	client, err := c.BlockStorageClient()
	if err != nil {
		return nil, err
	}
	pages, err := volumes.List(client, volumes.ListOpts{}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := volumes.ExtractVolumes(pages)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for i := range items {
		v := &items[i]
		res, err := newMqlOpenstackBlockstorageVolume(o.MqlRuntime, v)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func newMqlOpenstackBlockstorageVolume(runtime *plugin.Runtime, v *volumes.Volume) (*mqlOpenstackBlockstorageVolume, error) {
	res, err := CreateResource(runtime, "openstack.blockstorage.volume", map[string]*llx.RawData{
		"__id":              llx.StringData("openstack.blockstorage.volume/" + v.ID),
		"id":                llx.StringData(v.ID),
		"name":              llx.StringData(v.Name),
		"description":       llx.StringData(v.Description),
		"status":            llx.StringData(v.Status),
		"size":              llx.IntData(int64(v.Size)),
		"bootable":          llx.BoolData(parseBootable(v.Bootable)),
		"encrypted":         llx.BoolData(v.Encrypted),
		"multiAttach":       llx.BoolData(v.Multiattach),
		"volumeType":        llx.StringData(v.VolumeType),
		"availabilityZone":  llx.StringData(v.AvailabilityZone),
		"replicationStatus": llx.StringData(v.ReplicationStatus),
		"metadata":          stringMapData(v.Metadata),
		"imageMetadata":     stringMapData(v.VolumeImageMetadata),
		"attachments":       dictSliceData(volumeAttachmentsToDict(v.Attachments)),
		"createdAt":         llx.TimeDataPtr(timePtr(v.CreatedAt)),
		"updatedAt":         llx.TimeDataPtr(timePtr(v.UpdatedAt)),
	})
	if err != nil {
		return nil, err
	}
	mqlVol := res.(*mqlOpenstackBlockstorageVolume)
	mqlVol.cacheProjectID = v.TenantID
	mqlVol.cacheUserID = v.UserID
	mqlVol.cacheSourceVolumeID = v.SourceVolID
	mqlVol.cacheSourceSnapshotID = v.SnapshotID
	mqlVol.cacheServerIDs = volumeServerIDs(v.Attachments)
	return mqlVol, nil
}

func (r *mqlOpenstackBlockstorageVolume) project() (*mqlOpenstackProject, error) {
	if r.cacheProjectID == "" {
		r.Project.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.project", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheProjectID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackProject), nil
}

func (r *mqlOpenstackBlockstorageVolume) user() (*mqlOpenstackUser, error) {
	if r.cacheUserID == "" {
		r.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.user", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheUserID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackUser), nil
}

func (r *mqlOpenstackBlockstorageVolume) sourceVolume() (*mqlOpenstackBlockstorageVolume, error) {
	if r.cacheSourceVolumeID == "" {
		r.SourceVolume.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.blockstorage.volume", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheSourceVolumeID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackBlockstorageVolume), nil
}

func (r *mqlOpenstackBlockstorageVolume) sourceSnapshot() (*mqlOpenstackBlockstorageSnapshot, error) {
	if r.cacheSourceSnapshotID == "" {
		r.SourceSnapshot.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.blockstorage.snapshot", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheSourceSnapshotID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackBlockstorageSnapshot), nil
}

func (r *mqlOpenstackBlockstorageVolume) servers() ([]any, error) {
	if len(r.cacheServerIDs) == 0 {
		return []any{}, nil
	}
	out := make([]any, 0, len(r.cacheServerIDs))
	for _, id := range r.cacheServerIDs {
		if id == "" {
			continue
		}
		res, err := NewResource(r.MqlRuntime, "openstack.compute.server", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// parseBootable converts Cinder's "true"/"false" string into a bool.
// Cinder returns this field as a string for historical reasons.
func parseBootable(s string) bool {
	return strings.EqualFold(s, "true")
}

func volumeAttachmentsToDict(in []volumes.Attachment) []any {
	out := make([]any, 0, len(in))
	for _, a := range in {
		entry := map[string]any{
			"attachment_id": a.AttachmentID,
			"server_id":     a.ServerID,
			"device":        a.Device,
			"host_name":     a.HostName,
		}
		if !a.AttachedAt.IsZero() {
			entry["attached_at"] = a.AttachedAt
		}
		out = append(out, entry)
	}
	return out
}

func volumeServerIDs(in []volumes.Attachment) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, a := range in {
		if a.ServerID == "" {
			continue
		}
		if _, dup := seen[a.ServerID]; dup {
			continue
		}
		seen[a.ServerID] = struct{}{}
		out = append(out, a.ServerID)
	}
	return out
}

// ---- openstack.blockstorage.snapshot ----

type mqlOpenstackBlockstorageSnapshotInternal struct {
	cacheVolumeID  string
	cacheProjectID string
	cacheUserID    string
}

func (r *mqlOpenstackBlockstorageSnapshot) id() (string, error) {
	return "openstack.blockstorage.snapshot/" + r.Id.Data, nil
}

func initOpenstackBlockstorageSnapshot(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := stringArg(args, "id")
	if !ok || id == "" {
		return args, nil, nil
	}
	root, err := CreateResource(runtime, "openstack", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	list := root.(*mqlOpenstack).GetSnapshots()
	if list.Error == nil {
		for _, raw := range list.Data {
			s := raw.(*mqlOpenstackBlockstorageSnapshot)
			if s.Id.Data == id {
				return args, s, nil
			}
		}
	}
	initSyntheticID("openstack.blockstorage.snapshot", "id", args)
	return args, nil, nil
}

func (o *mqlOpenstack) snapshots() ([]any, error) {
	c := conn(o.MqlRuntime)
	client, err := c.BlockStorageClient()
	if err != nil {
		return nil, err
	}
	pages, err := snapshots.List(client, snapshots.ListOpts{}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := snapshots.ExtractSnapshots(pages)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for i := range items {
		s := &items[i]
		res, err := CreateResource(o.MqlRuntime, "openstack.blockstorage.snapshot", map[string]*llx.RawData{
			"__id":            llx.StringData("openstack.blockstorage.snapshot/" + s.ID),
			"id":              llx.StringData(s.ID),
			"name":            llx.StringData(s.Name),
			"description":     llx.StringData(s.Description),
			"status":          llx.StringData(s.Status),
			"size":            llx.IntData(int64(s.Size)),
			"progress":        llx.StringData(s.Progress),
			"groupSnapshotId": llx.StringData(s.GroupSnapshotID),
			"consumesQuota":   llx.BoolData(s.ConsumesQuota),
			"metadata":        stringMapData(s.Metadata),
			"createdAt":       llx.TimeDataPtr(timePtr(s.CreatedAt)),
			"updatedAt":       llx.TimeDataPtr(timePtr(s.UpdatedAt)),
		})
		if err != nil {
			return nil, err
		}
		mqlSnap := res.(*mqlOpenstackBlockstorageSnapshot)
		mqlSnap.cacheVolumeID = s.VolumeID
		mqlSnap.cacheProjectID = s.ProjectID
		mqlSnap.cacheUserID = s.UserID
		out = append(out, mqlSnap)
	}
	return out, nil
}

func (r *mqlOpenstackBlockstorageSnapshot) volume() (*mqlOpenstackBlockstorageVolume, error) {
	if r.cacheVolumeID == "" {
		r.Volume.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.blockstorage.volume", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheVolumeID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackBlockstorageVolume), nil
}

func (r *mqlOpenstackBlockstorageSnapshot) project() (*mqlOpenstackProject, error) {
	if r.cacheProjectID == "" {
		r.Project.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.project", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheProjectID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackProject), nil
}

func (r *mqlOpenstackBlockstorageSnapshot) user() (*mqlOpenstackUser, error) {
	if r.cacheUserID == "" {
		r.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.user", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheUserID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackUser), nil
}
