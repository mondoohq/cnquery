// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers/cloudflare/connection"
)

func (c *mqlCloudflareZone) workers() (*mqlCloudflareWorkers, error) {
	res, err := CreateResource(c.MqlRuntime, "cloudflare.workers", map[string]*llx.RawData{
		"__id": llx.StringData("cloudflare.workers"),
	})
	if err != nil {
		return nil, err
	}

	workers := res.(*mqlCloudflareWorkers)
	workers.AccountID = c.GetAccount().Data.GetId().Data

	return workers, nil
}

type mqlCloudflareWorkersInternal struct {
	AccountID string
}

func (c *mqlCloudflareWorkers) workers() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	resp, _, err := conn.Cf.ListWorkers(context.TODO(), &cloudflare.ResourceContainer{
		Identifier: c.mqlCloudflareWorkersInternal.AccountID,
		Level:      cloudflare.AccountRouteLevel,
	}, cloudflare.ListWorkersParams{})
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range resp.WorkerList {
		w := resp.WorkerList[i]

		placementMode := ""
		if w.PlacementMode != nil {
			placementMode = string(*w.PlacementMode)
		}

		res, err := NewResource(c.MqlRuntime, "cloudflare.workers.worker", map[string]*llx.RawData{
			"id":               llx.StringData(w.ID),
			"etag":             llx.StringData(w.ETAG),
			"size":             llx.IntData(w.Size),
			"deploymentId":     llx.StringDataPtr(w.DeploymentId),
			"pipelineHash":     llx.StringDataPtr(w.PipelineHash),
			"placementMode":    llx.StringData(placementMode),
			"lastDeployedFrom": llx.StringDataPtr(w.LastDeployedFrom),
			"logPush":          llx.BoolDataPtr(w.Logpush),
			"createdOn":        llx.TimeData(w.CreatedOn),
			"modifiedOn":       llx.TimeData(w.ModifiedOn),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, res)
	}

	return result, nil
}

func (c *mqlCloudflareWorkers) pages() ([]any, error) {
	return nil, nil
}
