// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"github.com/stackitcloud/stackit-sdk-go/services/serverbackup"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

// ------------------------- server backups -------------------------

func (r *mqlStackitServer) backups() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.ServerBackup()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListBackupsExecute(bgctx(), c.ProjectID(), r.Id.Data, c.Region())
	if err != nil {
		if isAccessDenied(err) || isNotFound(err) {
			// A 404 means the Server Backup service is not enabled for this
			// server, which is a legitimate "no backups" state rather than an
			// error.
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildServerBackup(r.MqlRuntime, r.Id.Data, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func buildServerBackup(runtime *plugin.Runtime, serverID string, b *serverbackup.Backup) (plugin.Resource, error) {
	volumeBackups := []any{}
	for _, vb := range b.GetVolumeBackups() {
		volumeBackups = append(volumeBackups, map[string]any{
			"id":                   vb.GetId(),
			"volumeId":             vb.GetVolumeId(),
			"size":                 vb.GetSize(),
			"status":               string(vb.GetStatus()),
			"lastRestoredAt":       vb.GetLastRestoredAt(),
			"lastRestoredVolumeId": vb.GetLastRestoredVolumeId(),
		})
	}
	args := map[string]*llx.RawData{
		"id":             llx.StringData(b.GetId()),
		"serverId":       llx.StringData(serverID),
		"name":           llx.StringData(b.GetName()),
		"status":         llx.StringData(string(b.GetStatus())),
		"size":           llx.IntData(b.GetSize()),
		"createdAt":      llx.TimeDataPtr(parseRFC3339(b.GetCreatedAt())),
		"expireAt":       llx.TimeDataPtr(parseRFC3339(b.GetExpireAt())),
		"lastRestoredAt": llx.TimeDataPtr(parseRFC3339(b.GetLastRestoredAt())),
		"volumeBackups":  llx.ArrayData(volumeBackups, types.Dict),
	}
	return CreateResource(runtime, "stackit.server.backup", args)
}

func (r *mqlStackitServerBackup) id() (string, error) {
	return "stackit.server.backup/" + r.ServerId.Data + "/" + r.Id.Data, nil
}

func (r *mqlStackitServerBackup) server() (*mqlStackitServer, error) {
	return serverRef(r.MqlRuntime, r.ServerId.Data, &r.Server)
}

// volumes resolves the volumes protected by the backup from the volume IDs
// carried in its per-volume backup entries.
func (r *mqlStackitServerBackup) volumes() ([]any, error) {
	ids := []string{}
	for _, raw := range r.VolumeBackups.Data {
		vb, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := vb["volumeId"].(string); ok {
			ids = append(ids, id)
		}
	}
	return volumeRefs(r.MqlRuntime, ids)
}

// ------------------------- server backup schedules -------------------------

func (r *mqlStackitServer) backupSchedules() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.ServerBackup()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListBackupSchedulesExecute(bgctx(), c.ProjectID(), r.Id.Data, c.Region())
	if err != nil {
		if isAccessDenied(err) || isNotFound(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildBackupSchedule(r.MqlRuntime, r.Id.Data, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func buildBackupSchedule(runtime *plugin.Runtime, serverID string, s *serverbackup.BackupSchedule) (plugin.Resource, error) {
	var (
		retention int64
		volumeIds []string
	)
	if props, ok := s.GetBackupPropertiesOk(); ok {
		retention = props.GetRetentionPeriod()
		volumeIds = props.GetVolumeIds()
	}
	args := map[string]*llx.RawData{
		"id":              llx.IntData(s.GetId()),
		"serverId":        llx.StringData(serverID),
		"name":            llx.StringData(s.GetName()),
		"enabled":         llx.BoolData(s.GetEnabled()),
		"rrule":           llx.StringData(s.GetRrule()),
		"retentionPeriod": llx.IntData(retention),
		"volumeIds":       strSliceData(volumeIds),
	}
	return CreateResource(runtime, "stackit.server.backupSchedule", args)
}

func (r *mqlStackitServerBackupSchedule) id() (string, error) {
	return "stackit.server.backupSchedule/" + r.ServerId.Data + "/" + strconv.FormatInt(r.Id.Data, 10), nil
}

func (r *mqlStackitServerBackupSchedule) server() (*mqlStackitServer, error) {
	return serverRef(r.MqlRuntime, r.ServerId.Data, &r.Server)
}

func (r *mqlStackitServerBackupSchedule) volumes() ([]any, error) {
	ids := make([]string, 0, len(r.VolumeIds.Data))
	for _, raw := range r.VolumeIds.Data {
		if id, ok := raw.(string); ok {
			ids = append(ids, id)
		}
	}
	return volumeRefs(r.MqlRuntime, ids)
}
