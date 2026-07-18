// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// ------------------------- volume backups -------------------------

func (r *mqlStackit) backups() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.IaaS()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListBackupsExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildBackup(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func buildBackup(runtime *plugin.Runtime, b *iaas.Backup) (plugin.Resource, error) {
	createdAt, ok1 := b.GetCreatedAtOk()
	updatedAt, ok2 := b.GetUpdatedAtOk()
	args := map[string]*llx.RawData{
		"id":               llx.StringData(b.GetId()),
		"name":             llx.StringData(b.GetName()),
		"description":      llx.StringData(b.GetDescription()),
		"status":           llx.StringData(b.GetStatus()),
		"size":             llx.IntData(b.GetSize()),
		"availabilityZone": llx.StringData(b.GetAvailabilityZone()),
		"encrypted":        llx.BoolData(b.GetEncrypted()),
		"volumeId":         llx.StringData(b.GetVolumeId()),
		"snapshotId":       llx.StringData(b.GetSnapshotId()),
		"createdAt":        llx.TimeDataPtr(timeOrNil(createdAt, ok1)),
		"updatedAt":        llx.TimeDataPtr(timeOrNil(updatedAt, ok2)),
		"labels":           labelData(b.GetLabels()),
	}
	return CreateResource(runtime, "stackit.backup", args)
}

func (r *mqlStackitBackup) id() (string, error) {
	return "stackit.backup/" + r.Id.Data, nil
}

func initStackitBackup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.IaaS()
	if err != nil {
		return nil, nil, err
	}
	b, err := client.GetBackupExecute(bgctx(), c.ProjectID(), c.Region(), id)
	if err != nil {
		return nil, nil, err
	}
	res, err := buildBackup(runtime, b)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func (r *mqlStackitBackup) volume() (*mqlStackitVolume, error) {
	return volumeRef(r.MqlRuntime, r.VolumeId.Data, &r.Volume)
}

func (r *mqlStackitBackup) snapshot() (*mqlStackitSnapshot, error) {
	if r.SnapshotId.Data == "" {
		return markNull[mqlStackitSnapshot](&r.Snapshot)
	}
	res, err := NewResource(r.MqlRuntime, "stackit.snapshot", map[string]*llx.RawData{
		"id": llx.StringData(r.SnapshotId.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitSnapshot), nil
}

type mqlStackitAffinityGroupInternal struct {
	// cacheMemberIds holds the member server UUIDs, captured when the group is
	// built so servers() can resolve them. It is not exposed as a field
	// because servers() already carries the same information.
	cacheMemberIds []string
}

// ------------------------- affinity groups -------------------------

func (r *mqlStackit) affinityGroups() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.IaaS()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListAffinityGroupsExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildAffinityGroup(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func buildAffinityGroup(runtime *plugin.Runtime, ag *iaas.AffinityGroup) (plugin.Resource, error) {
	args := map[string]*llx.RawData{
		"id":     llx.StringData(ag.GetId()),
		"name":   llx.StringData(ag.GetName()),
		"policy": llx.StringData(ag.GetPolicy()),
	}
	res, err := CreateResource(runtime, "stackit.affinityGroup", args)
	if err != nil {
		return nil, err
	}
	res.(*mqlStackitAffinityGroup).cacheMemberIds = ag.GetMembers()
	return res, nil
}

func (r *mqlStackitAffinityGroup) id() (string, error) {
	return "stackit.affinityGroup/" + r.Id.Data, nil
}

func initStackitAffinityGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.IaaS()
	if err != nil {
		return nil, nil, err
	}
	ag, err := client.GetAffinityGroupExecute(bgctx(), c.ProjectID(), c.Region(), id)
	if err != nil {
		return nil, nil, err
	}
	res, err := buildAffinityGroup(runtime, ag)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

// servers resolves the affinity group's member servers from their UUIDs.
func (r *mqlStackitAffinityGroup) servers() ([]any, error) {
	out := make([]any, 0, len(r.cacheMemberIds))
	for _, id := range r.cacheMemberIds {
		if id == "" {
			continue
		}
		res, err := NewResource(r.MqlRuntime, "stackit.server", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}
