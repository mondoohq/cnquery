// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/digitalocean/godo"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/digitalocean/connection"
)

func (r *mqlDigitalocean) dropletAutoscalePools() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		pools, resp, err := client.DropletAutoscale.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, p := range pools {
			if p == nil {
				continue
			}

			var minInstances, maxInstances, targetNumberInstances, cooldownMinutes int64
			var targetCPU, targetMem float64
			if p.Config != nil {
				minInstances = int64(p.Config.MinInstances)
				maxInstances = int64(p.Config.MaxInstances)
				targetNumberInstances = int64(p.Config.TargetNumberInstances)
				cooldownMinutes = int64(p.Config.CooldownMinutes)
				targetCPU = p.Config.TargetCPUUtilization
				targetMem = p.Config.TargetMemoryUtilization
			}

			var currentCPU, currentMem float64
			if p.CurrentUtilization != nil {
				currentCPU = p.CurrentUtilization.CPU
				currentMem = p.CurrentUtilization.Memory
			}

			template := map[string]interface{}{}
			if t := p.DropletTemplate; t != nil {
				tags := make([]interface{}, len(t.Tags))
				for i, tag := range t.Tags {
					tags[i] = tag
				}
				sshKeys := make([]interface{}, len(t.SSHKeys))
				for i, k := range t.SSHKeys {
					sshKeys[i] = k
				}
				template = map[string]interface{}{
					"size":             t.Size,
					"region":           t.Region,
					"image":            t.Image,
					"tags":             tags,
					"sshKeys":          sshKeys,
					"vpcUuid":          t.VpcUUID,
					"projectId":        t.ProjectID,
					"ipv6":             t.IPV6,
					"withDropletAgent": t.WithDropletAgent,
					"userData":         t.UserData,
				}
				if t.PublicNetworking != nil {
					template["publicNetworking"] = *t.PublicNetworking
				}
			}

			res, err := CreateResource(r.MqlRuntime, "digitalocean.dropletAutoscalePool", map[string]*llx.RawData{
				"id":                       llx.StringData(p.ID),
				"name":                     llx.StringData(p.Name),
				"status":                   llx.StringData(p.Status),
				"minInstances":             llx.IntData(minInstances),
				"maxInstances":             llx.IntData(maxInstances),
				"targetCpuUtilization":     llx.FloatData(targetCPU),
				"targetMemoryUtilization":  llx.FloatData(targetMem),
				"cooldownMinutes":          llx.IntData(cooldownMinutes),
				"targetNumberInstances":    llx.IntData(targetNumberInstances),
				"currentCpuUtilization":    llx.FloatData(currentCPU),
				"currentMemoryUtilization": llx.FloatData(currentMem),
				"dropletTemplate":          llx.DictData(template),
				"createdAt":                llx.TimeData(p.CreatedAt),
				"updatedAt":                llx.TimeData(p.UpdatedAt),
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
	return all, nil
}
