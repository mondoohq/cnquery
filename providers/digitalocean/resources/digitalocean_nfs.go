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

// nfsShares enumerates managed NFS shares. The DigitalOcean NFS list API
// is region-scoped, so we fan out over the account's available regions
// and aggregate the results.
func (r *mqlDigitalocean) nfsShares() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()
	ctx := context.Background()

	regions, _, err := client.Regions.List(ctx, &godo.ListOptions{PerPage: 200})
	if err != nil {
		return nil, err
	}

	var all []interface{}
	for _, region := range regions {
		opt := &godo.ListOptions{PerPage: 200}
		for {
			shares, resp, err := client.Nfs.List(ctx, opt, region.Slug)
			if err != nil {
				return nil, err
			}
			for _, s := range shares {
				if s == nil {
					continue
				}

				vpcIDs := make([]interface{}, len(s.VpcIDs))
				for i, v := range s.VpcIDs {
					vpcIDs[i] = v
				}

				var createdAt *time.Time
				if t, perr := time.Parse(time.RFC3339, s.CreatedAt); perr == nil {
					createdAt = &t
				}

				res, err := CreateResource(r.MqlRuntime, "digitalocean.nfs", map[string]*llx.RawData{
					"id":              llx.StringData(s.ID),
					"name":            llx.StringData(s.Name),
					"sizeGib":         llx.IntData(int64(s.SizeGib)),
					"region":          llx.StringData(s.Region),
					"status":          llx.StringData(string(s.Status)),
					"performanceTier": llx.StringData(s.PerformanceTier),
					"host":            llx.StringData(s.Host),
					"mountPath":       llx.StringData(s.MountPath),
					"vpcIds":          llx.ArrayData(vpcIDs, "\x02"),
					"createdAt":       llx.TimeDataPtr(createdAt),
				})
				if err != nil {
					return nil, err
				}
				all = append(all, res)
			}
			if resp == nil || resp.Links == nil || resp.Links.IsLastPage() {
				break
			}
			page, err := resp.Links.CurrentPage()
			if err != nil {
				return nil, err
			}
			opt.Page = page + 1
		}
	}
	return all, nil
}

func (r *mqlDigitaloceanNfs) vpcs() ([]interface{}, error) {
	uuids := make([]string, 0, len(r.VpcIds.Data))
	for _, v := range r.VpcIds.Data {
		if s, ok := v.(string); ok {
			uuids = append(uuids, s)
		}
	}
	return vpcRefsByUUIDs(r.MqlRuntime, uuids)
}
