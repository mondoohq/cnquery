// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/proxmox/connection"
)

type mqlProxmoxBackupJobInternal struct {
	configFetched bool
	cfg           map[string]any
	cfgErr        error
	lock          sync.Mutex
}

func (r *mqlProxmox) backupJobs() ([]any, error) {
	conn := proxmoxConn(r)
	jobs, err := conn.GetBackupJobs()
	if err != nil {
		return nil, err
	}
	list := make([]any, len(jobs))
	for i, j := range jobs {
		res, err := CreateResource(r.MqlRuntime, "proxmox.backup.job", map[string]*llx.RawData{
			"id":               llx.StringData(j.ID),
			"enabled":          llx.BoolData(j.Enabled != 0),
			"schedule":         llx.StringData(j.Schedule),
			"storage":          llx.StringData(j.Storage),
			"mode":             llx.StringData(j.Mode),
			"comment":          llx.StringData(j.Comment),
			"vmids":            llx.StringData(j.VMID),
			"pool":             llx.StringData(j.Pool),
			"all":              llx.BoolData(j.All != 0),
			"exclude":          llx.StringData(j.Exclude),
			"compress":         llx.StringData(j.Compress),
			"mailto":           llx.StringData(j.Mailto),
			"notificationMode": llx.StringData(j.NotificationMode),
			"node":             llx.StringData(j.Node),
			"prune":            llx.StringData(j.Prune),
			"fleecing":         llx.StringData(j.Fleecing),
			"notesTemplate":    llx.StringData(j.NotesTemplate),
			"protected":        llx.BoolData(j.Protected != 0),
			"nextRun":          llx.IntData(j.NextRun),
		})
		if err != nil {
			return nil, err
		}
		list[i] = res
	}
	return list, nil
}

func (r *mqlProxmoxBackupJob) targetStorage() (*mqlProxmoxStorage, error) {
	if r.Storage.Data == "" {
		r.TargetStorage.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "proxmox.storage", map[string]*llx.RawData{
		"id": llx.StringData(r.Storage.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlProxmoxStorage), nil
}

func (r *mqlProxmoxBackupJob) config() (any, error) {
	if r.configFetched {
		return r.cfg, r.cfgErr
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.configFetched {
		return r.cfg, r.cfgErr
	}
	conn := r.MqlRuntime.Connection.(*connection.PveConnection)
	r.cfg, r.cfgErr = conn.GetBackupJob(r.Id.Data)
	r.configFetched = true
	return r.cfg, r.cfgErr
}
