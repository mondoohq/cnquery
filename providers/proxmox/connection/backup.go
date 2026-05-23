// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import "fmt"

// BackupJob describes a cluster-wide scheduled vzdump job from /cluster/backup.
// Fields use json.RawMessage-friendly types where the Proxmox API is loose
// (int vs string vs bool depending on version).
type BackupJob struct {
	ID               string `json:"id"`
	Enabled          int    `json:"enabled"`
	Schedule         string `json:"schedule"`
	Storage          string `json:"storage"`
	Mode             string `json:"mode"`
	Comment          string `json:"comment"`
	VMID             string `json:"vmid"`
	Pool             string `json:"pool"`
	All              int    `json:"all"`
	Exclude          string `json:"exclude"`
	Compress         string `json:"compress"`
	Mailto           string `json:"mailto"`
	NotificationMode string `json:"notification-mode"`
	Node             string `json:"node"`
	Prune            string `json:"prune-backups"`
	Fleecing         string `json:"fleecing"`
	NotesTemplate    string `json:"notes-template"`
	Protected        int    `json:"protected"`
	NextRun          int64  `json:"next-run"`
	Type             string `json:"type"`
	Repeat           int    `json:"repeat-missed"`
	Remove           int    `json:"remove"`
}

func (c *PveConnection) GetBackupJobs() ([]BackupJob, error) {
	var jobs []BackupJob
	if err := c.apiGet("/cluster/backup", &jobs); err != nil {
		return nil, fmt.Errorf("failed to get backup jobs: %w", err)
	}
	return jobs, nil
}

// GetBackupJob returns the full raw config for a specific backup job, used
// to expose less-common fields via the `config` dict on the resource.
func (c *PveConnection) GetBackupJob(id string) (map[string]any, error) {
	var cfg map[string]any
	path := fmt.Sprintf("/cluster/backup/%s", id)
	if err := c.apiGet(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to get backup job %s: %w", id, err)
	}
	return cfg, nil
}
