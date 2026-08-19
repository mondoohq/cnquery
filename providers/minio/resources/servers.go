// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	madmin "github.com/minio/madmin-go/v4"
	"go.mondoo.com/mql/v13/llx"
)

// driveCounts reports how many drives a server has and how many of them are
// healthy. MinIO reports a per-drive state string, and "ok" is the only value
// that means the drive is usable.
func driveCounts(drives []madmin.Disk) (total int64, online int64) {
	for _, drive := range drives {
		total++
		if drive.State == "ok" {
			online++
		}
	}
	return total, online
}

func (r *mqlMinio) servers() ([]any, error) {
	info, err := r.info()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(info.Servers))
	for i := range info.Servers {
		server := info.Servers[i]
		total, online := driveCounts(server.Disks)

		resource, err := CreateResource(r.MqlRuntime, "minio.server", map[string]*llx.RawData{
			"__id":         llx.StringData("server/" + server.Endpoint),
			"endpoint":     llx.StringData(server.Endpoint),
			"state":        llx.StringData(server.State),
			"version":      llx.StringData(server.Version),
			"commitId":     llx.StringData(server.CommitID),
			"uptime":       llx.IntData(server.Uptime),
			"poolNumber":   llx.IntData(int64(server.PoolNumber)),
			"totalDrives":  llx.IntData(total),
			"onlineDrives": llx.IntData(online),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, resource)
	}
	return res, nil
}
