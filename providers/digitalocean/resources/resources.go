// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"github.com/digitalocean/godo"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/digitalocean/connection"
)

// --- Database sub-resources ---

func (r *mqlDigitaloceanDatabase) users() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		users, resp, err := client.Databases.ListUsers(context.Background(), r.Id.Data, opt)
		if err != nil {
			return nil, err
		}
		for _, u := range users {
			res, err := CreateResource(r.MqlRuntime, "digitalocean.database.user", map[string]*llx.RawData{
				"databaseId": llx.StringData(r.Id.Data),
				"name":       llx.StringData(u.Name),
				"role":       llx.StringData(u.Role),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, _ := resp.Links.CurrentPage()
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanDatabaseUser) id() (string, error) {
	return "digitalocean.database.user/" + r.DatabaseId.Data + "/" + r.Name.Data, nil
}

func (r *mqlDigitaloceanDatabase) replicas() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		replicas, resp, err := client.Databases.ListReplicas(context.Background(), r.Id.Data, opt)
		if err != nil {
			return nil, err
		}
		for _, rep := range replicas {
			tags := make([]interface{}, len(rep.Tags))
			for i, t := range rep.Tags {
				tags[i] = t
			}
			res, err := CreateResource(r.MqlRuntime, "digitalocean.database.replica", map[string]*llx.RawData{
				"databaseId": llx.StringData(r.Id.Data),
				"name":       llx.StringData(rep.Name),
				"region":     llx.StringData(rep.Region),
				"status":     llx.StringData(rep.Status),
				"createdAt":  llx.TimeData(rep.CreatedAt),
				"size":       llx.StringData(rep.Size),
				"tags":       llx.ArrayData(tags, "\x02"),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, _ := resp.Links.CurrentPage()
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanDatabaseReplica) id() (string, error) {
	return "digitalocean.database.replica/" + r.DatabaseId.Data + "/" + r.Name.Data, nil
}

func (r *mqlDigitaloceanDatabase) pools() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		pools, resp, err := client.Databases.ListPools(context.Background(), r.Id.Data, opt)
		if err != nil {
			return nil, err
		}
		for _, p := range pools {
			res, err := CreateResource(r.MqlRuntime, "digitalocean.database.pool", map[string]*llx.RawData{
				"databaseId": llx.StringData(r.Id.Data),
				"name":       llx.StringData(p.Name),
				"database":   llx.StringData(p.Database),
				"user":       llx.StringData(p.User),
				"size":       llx.IntData(int64(p.Size)),
				"mode":       llx.StringData(p.Mode),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, _ := resp.Links.CurrentPage()
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanDatabasePool) id() (string, error) {
	return "digitalocean.database.pool/" + r.DatabaseId.Data + "/" + r.Name.Data, nil
}

// --- VPC Peering ---

func (r *mqlDigitalocean) vpcPeerings() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		peerings, resp, err := client.VPCs.ListVPCPeerings(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, p := range peerings {
			vpcIds := make([]interface{}, len(p.VPCIDs))
			for i, id := range p.VPCIDs {
				vpcIds[i] = id
			}
			res, err := CreateResource(r.MqlRuntime, "digitalocean.vpcPeering", map[string]*llx.RawData{
				"id":        llx.StringData(p.ID),
				"name":      llx.StringData(p.Name),
				"vpcIds":    llx.ArrayData(vpcIds, "\x02"),
				"status":    llx.StringData(string(p.Status)),
				"createdAt": llx.TimeData(p.CreatedAt),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, _ := resp.Links.CurrentPage()
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanVpcPeering) id() (string, error) {
	return "digitalocean.vpcPeering/" + r.Id.Data, nil
}

// --- Kubernetes node pools (typed) ---

func (r *mqlDigitaloceanKubernetesCluster) nodePools() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		pools, resp, err := client.Kubernetes.ListNodePools(context.Background(), r.Id.Data, opt)
		if err != nil {
			return nil, err
		}
		for _, np := range pools {
			npTags := make([]interface{}, len(np.Tags))
			for i, t := range np.Tags {
				npTags[i] = t
			}

			labels := map[string]interface{}{}
			for k, v := range np.Labels {
				labels[k] = v
			}

			taints := make([]interface{}, len(np.Taints))
			for i, t := range np.Taints {
				taints[i] = map[string]interface{}{
					"key":    t.Key,
					"value":  t.Value,
					"effect": t.Effect,
				}
			}

			res, err := CreateResource(r.MqlRuntime, "digitalocean.kubernetes.nodePool", map[string]*llx.RawData{
				"id":        llx.StringData(np.ID),
				"clusterId": llx.StringData(r.Id.Data),
				"name":      llx.StringData(np.Name),
				"size":      llx.StringData(np.Size),
				"count":     llx.IntData(int64(np.Count)),
				"autoScale": llx.BoolData(np.AutoScale),
				"minNodes":  llx.IntData(int64(np.MinNodes)),
				"maxNodes":  llx.IntData(int64(np.MaxNodes)),
				"tags":      llx.ArrayData(npTags, "\x02"),
				"labels":    llx.DictData(labels),
				"taints":    llx.ArrayData(taints, "\x13"),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, _ := resp.Links.CurrentPage()
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanKubernetesNodePool) id() (string, error) {
	return "digitalocean.kubernetes.nodePool/" + r.ClusterId.Data + "/" + r.Id.Data, nil
}

// --- Container Registry ---

func initDigitaloceanRegistry(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.DigitaloceanConnection)
	reg, _, err := conn.Client().Registry.Get(context.Background())
	if err != nil {
		// No registry configured -- return empty sentinel
		args["name"] = llx.StringData("")
		args["storageUsageBytes"] = llx.IntData(0)
		args["region"] = llx.StringData("")
		args["createdAt"] = llx.TimeData(time.Time{})
		args["subscriptionTier"] = llx.StringData("")
		return args, nil, nil
	}
	args["name"] = llx.StringData(reg.Name)
	args["storageUsageBytes"] = llx.IntData(int64(reg.StorageUsageBytes))
	args["region"] = llx.StringData(reg.Region)
	args["createdAt"] = llx.TimeData(reg.CreatedAt)

	sub, _, err := conn.Client().Registry.GetSubscription(context.Background())
	tier := ""
	if err == nil && sub.Tier != nil {
		tier = sub.Tier.Slug
	}
	args["subscriptionTier"] = llx.StringData(tier)
	return args, nil, nil
}

func (r *mqlDigitaloceanRegistry) id() (string, error) {
	return "digitalocean.registry", nil
}

func (r *mqlDigitalocean) registryRepositories() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	reg, _, err := client.Registry.Get(context.Background())
	if err != nil {
		return []interface{}{}, nil
	}

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		repos, resp, err := client.Registry.ListRepositoriesV2(context.Background(), reg.Name, &godo.TokenListOptions{
			PerPage: opt.PerPage,
			Page:    opt.Page,
		})
		if err != nil {
			return nil, err
		}
		for _, repo := range repos {
			res, err := CreateResource(r.MqlRuntime, "digitalocean.registry.repository", map[string]*llx.RawData{
				"registryName":  llx.StringData(repo.RegistryName),
				"name":          llx.StringData(repo.Name),
				"tagCount":      llx.IntData(int64(repo.TagCount)),
				"manifestCount": llx.IntData(int64(repo.ManifestCount)),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, _ := resp.Links.CurrentPage()
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanRegistryRepository) id() (string, error) {
	return "digitalocean.registry.repository/" + r.RegistryName.Data + "/" + r.Name.Data, nil
}

// --- Reserved IPs ---

func (r *mqlDigitalocean) reservedIPs() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		ips, resp, err := client.ReservedIPs.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			regionSlug := ""
			if ip.Region != nil {
				regionSlug = ip.Region.Slug
			}
			dropletId := int64(0)
			if ip.Droplet != nil {
				dropletId = int64(ip.Droplet.ID)
			}
			res, err := CreateResource(r.MqlRuntime, "digitalocean.reservedIp", map[string]*llx.RawData{
				"ip":        llx.StringData(ip.IP),
				"region":    llx.StringData(regionSlug),
				"projectId": llx.StringData(ip.ProjectID),
				"locked":    llx.BoolData(ip.Locked),
				"dropletId": llx.IntData(dropletId),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, _ := resp.Links.CurrentPage()
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanReservedIp) id() (string, error) {
	return "digitalocean.reservedIp/" + r.Ip.Data, nil
}

// --- App Platform ---

func (r *mqlDigitalocean) apps() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		apps, resp, err := client.Apps.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, app := range apps {
			status := ""
			if !app.LastDeploymentActiveAt.IsZero() {
				status = "active"
			}
			if app.InProgressDeployment != nil {
				status = "deploying"
			}

			spec := map[string]interface{}{}
			if app.Spec != nil {
				spec["name"] = app.Spec.Name
				svcNames := make([]interface{}, len(app.Spec.Services))
				for i, s := range app.Spec.Services {
					svcNames[i] = s.Name
				}
				spec["services"] = svcNames
				workerNames := make([]interface{}, len(app.Spec.Workers))
				for i, w := range app.Spec.Workers {
					workerNames[i] = w.Name
				}
				spec["workers"] = workerNames
				jobNames := make([]interface{}, len(app.Spec.Jobs))
				for i, j := range app.Spec.Jobs {
					jobNames[i] = j.Name
				}
				spec["jobs"] = jobNames
				staticNames := make([]interface{}, len(app.Spec.StaticSites))
				for i, s := range app.Spec.StaticSites {
					staticNames[i] = s.Name
				}
				spec["staticSites"] = staticNames
				fnNames := make([]interface{}, len(app.Spec.Functions))
				for i, f := range app.Spec.Functions {
					fnNames[i] = f.Name
				}
				spec["functions"] = fnNames
			}

			name := ""
			if app.Spec != nil {
				name = app.Spec.Name
			}

			res, err := CreateResource(r.MqlRuntime, "digitalocean.app", map[string]*llx.RawData{
				"id":                     llx.StringData(app.ID),
				"name":                   llx.StringData(name),
				"liveUrl":                llx.StringData(app.LiveURL),
				"createdAt":              llx.TimeData(app.CreatedAt),
				"updatedAt":              llx.TimeData(app.UpdatedAt),
				"activeDeploymentStatus": llx.StringData(status),
				"spec":                   llx.DictData(spec),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, _ := resp.Links.CurrentPage()
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanApp) id() (string, error) {
	return "digitalocean.app/" + r.Id.Data, nil
}

// --- Monitoring Alert Policies ---

func (r *mqlDigitalocean) alertPolicies() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		policies, resp, err := client.Monitoring.ListAlertPolicies(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, p := range policies {
			entities := make([]interface{}, len(p.Entities))
			for i, e := range p.Entities {
				entities[i] = e
			}
			pTags := make([]interface{}, len(p.Tags))
			for i, t := range p.Tags {
				pTags[i] = t
			}

			emails := make([]interface{}, 0)
			slacks := make([]interface{}, 0)
			if p.Alerts.Email != nil {
				for _, e := range p.Alerts.Email {
					emails = append(emails, e)
				}
			}
			if p.Alerts.Slack != nil {
				for _, s := range p.Alerts.Slack {
					slacks = append(slacks, map[string]interface{}{
						"channel": s.Channel,
						"url":     s.URL,
					})
				}
			}

			res, err := CreateResource(r.MqlRuntime, "digitalocean.alertPolicy", map[string]*llx.RawData{
				"uuid":        llx.StringData(p.UUID),
				"type":        llx.StringData(p.Type),
				"description": llx.StringData(p.Description),
				"compare":     llx.StringData(string(p.Compare)),
				"value":       llx.FloatData(float64(p.Value)),
				"window":      llx.StringData(p.Window),
				"enabled":     llx.BoolData(p.Enabled),
				"entities":    llx.ArrayData(entities, "\x02"),
				"tags":        llx.ArrayData(pTags, "\x02"),
				"alertEmails": llx.ArrayData(emails, "\x02"),
				"alertSlack":  llx.ArrayData(slacks, "\x13"),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, _ := resp.Links.CurrentPage()
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanAlertPolicy) id() (string, error) {
	return "digitalocean.alertPolicy/" + r.Uuid.Data, nil
}

// --- Uptime Checks ---

func (r *mqlDigitalocean) uptimeChecks() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		checks, resp, err := client.UptimeChecks.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, c := range checks {
			regions := make([]interface{}, len(c.Regions))
			for i, r := range c.Regions {
				regions[i] = r
			}
			res, err := CreateResource(r.MqlRuntime, "digitalocean.uptimeCheck", map[string]*llx.RawData{
				"id":      llx.StringData(c.ID),
				"name":    llx.StringData(c.Name),
				"type":    llx.StringData(c.Type),
				"target":  llx.StringData(c.Target),
				"regions": llx.ArrayData(regions, "\x02"),
				"enabled": llx.BoolData(c.Enabled),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, _ := resp.Links.CurrentPage()
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanUptimeCheck) id() (string, error) {
	return "digitalocean.uptimeCheck/" + r.Id.Data, nil
}

// --- CDN ---

func (r *mqlDigitalocean) cdnEndpoints() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		cdns, resp, err := client.CDNs.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, c := range cdns {
			res, err := CreateResource(r.MqlRuntime, "digitalocean.cdn", map[string]*llx.RawData{
				"id":            llx.StringData(c.ID),
				"origin":        llx.StringData(c.Origin),
				"endpoint":      llx.StringData(c.Endpoint),
				"ttl":           llx.IntData(int64(c.TTL)),
				"certificateId": llx.StringData(c.CertificateID),
				"customDomain":  llx.StringData(c.CustomDomain),
				"createdAt":     llx.TimeData(c.CreatedAt),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, _ := resp.Links.CurrentPage()
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanCdn) id() (string, error) {
	return "digitalocean.cdn/" + r.Id.Data, nil
}

// --- Tags ---

func (r *mqlDigitalocean) tags() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		tags, resp, err := client.Tags.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, t := range tags {
			count := 0
			if t.Resources != nil {
				count = t.Resources.Count
			}
			res, err := CreateResource(r.MqlRuntime, "digitalocean.tag", map[string]*llx.RawData{
				"name":          llx.StringData(t.Name),
				"resourceCount": llx.IntData(int64(count)),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, _ := resp.Links.CurrentPage()
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanTag) id() (string, error) {
	return "digitalocean.tag/" + r.Name.Data, nil
}

// --- Spaces Keys ---

func (r *mqlDigitalocean) spacesKeys() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		keys, resp, err := client.SpacesKeys.List(context.Background(), opt)
		if err != nil {
			// Spaces Keys API may not be available if Spaces is not enabled
			return []interface{}{}, nil
		}
		for _, k := range keys {
			grants := make([]interface{}, len(k.Grants))
			for i, g := range k.Grants {
				grants[i] = map[string]interface{}{
					"bucket":     g.Bucket,
					"permission": string(g.Permission),
				}
			}
			res, err := CreateResource(r.MqlRuntime, "digitalocean.spacesKey", map[string]*llx.RawData{
				"name":      llx.StringData(k.Name),
				"accessKey": llx.StringData(k.AccessKey),
				"grants":    llx.ArrayData(grants, "\x13"),
				"createdAt": llx.TimeDataPtr(parseDoTime(k.CreatedAt)),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, _ := resp.Links.CurrentPage()
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanSpacesKey) id() (string, error) {
	return "digitalocean.spacesKey/" + r.AccessKey.Data, nil
}
