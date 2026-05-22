// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"github.com/digitalocean/godo"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/digitalocean/connection"
)

type mqlDigitaloceanDatabaseInternal struct {
	cachedBackups []godo.DatabaseBackup
}

func (r *mqlDigitaloceanDatabaseBackup) id() (string, error) {
	return "digitalocean.database.backup/" + r.DatabaseId.Data + "/" +
		r.CreatedAt.Data.UTC().Format(time.RFC3339Nano), nil
}

// listBackups fetches all retained backups for the cluster once and
// caches them; the same call feeds backups(), latestBackupAt(), and
// backupCount() so we don't hit the API three times.
func (r *mqlDigitaloceanDatabase) fetchBackups() ([]godo.DatabaseBackup, error) {
	if r.cachedBackups != nil {
		return r.cachedBackups, nil
	}
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []godo.DatabaseBackup
	opt := &godo.ListOptions{PerPage: 200}
	for {
		backups, resp, err := client.Databases.ListBackups(context.Background(), r.Id.Data, opt)
		if err != nil {
			return nil, err
		}
		all = append(all, backups...)
		if resp == nil || resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, perr := resp.Links.CurrentPage()
		if perr != nil {
			break
		}
		opt.Page = page + 1
	}
	r.cachedBackups = all
	return all, nil
}

func (r *mqlDigitaloceanDatabase) backups() ([]interface{}, error) {
	backups, err := r.fetchBackups()
	if err != nil {
		return nil, err
	}
	all := make([]interface{}, 0, len(backups))
	for i := range backups {
		b := backups[i]
		res, err := CreateResource(r.MqlRuntime, "digitalocean.database.backup", map[string]*llx.RawData{
			"databaseId":    llx.StringData(r.Id.Data),
			"createdAt":     llx.TimeData(b.CreatedAt),
			"sizeGigabytes": llx.FloatData(b.SizeGigabytes),
		})
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}

func (r *mqlDigitaloceanDatabase) latestBackupAt() (*time.Time, error) {
	backups, err := r.fetchBackups()
	if err != nil {
		return nil, err
	}
	if len(backups) == 0 {
		return nil, nil
	}
	latest := backups[0].CreatedAt
	for i := range backups {
		if backups[i].CreatedAt.After(latest) {
			latest = backups[i].CreatedAt
		}
	}
	return &latest, nil
}

func (r *mqlDigitaloceanDatabase) backupCount() (int64, error) {
	backups, err := r.fetchBackups()
	if err != nil {
		return 0, err
	}
	return int64(len(backups)), nil
}

func (r *mqlDigitaloceanDatabase) caCertificate() (string, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()
	ca, _, err := client.Databases.GetCA(context.Background(), r.Id.Data)
	if err != nil {
		return "", err
	}
	if ca == nil {
		return "", nil
	}
	return string(ca.Certificate), nil
}
