// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strconv"
	"time"

	"github.com/digitalocean/godo"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/digitalocean/connection"
)

func (r *mqlDigitalocean) id() (string, error) {
	return "digitalocean", nil
}

func (r *mqlDigitaloceanAccount) id() (string, error) {
	return "digitalocean.account", nil
}

func initDigitaloceanAccount(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.DigitaloceanConnection)
	acct, _, err := conn.Client().Account.Get(context.Background())
	if err != nil {
		return nil, nil, err
	}
	args["email"] = llx.StringData(acct.Email)
	args["uuid"] = llx.StringData(acct.UUID)
	args["dropletLimit"] = llx.IntData(int64(acct.DropletLimit))
	args["floatingIpLimit"] = llx.IntData(int64(acct.FloatingIPLimit))
	args["volumeLimit"] = llx.IntData(int64(acct.VolumeLimit))
	args["emailVerified"] = llx.BoolData(acct.EmailVerified)
	args["status"] = llx.StringData(acct.Status)
	args["statusMessage"] = llx.StringData(acct.StatusMessage)
	return args, nil, nil
}

func toStringSlice(s []string) []interface{} {
	r := make([]interface{}, len(s))
	for i, v := range s {
		r[i] = v
	}
	return r
}

func toIntSlice(s []int) []interface{} {
	r := make([]interface{}, len(s))
	for i, v := range s {
		r[i] = int64(v)
	}
	return r
}

// helper to parse DigitalOcean time strings
func parseDoTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}

type mqlDigitaloceanDropletInternal struct {
	// image caches the godo image embedded in the droplet list response so the
	// typed baseImage() accessor can build a digitalocean.image without a refetch.
	image *godo.Image
}

func (r *mqlDigitalocean) droplets() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		droplets, resp, err := client.Droplets.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, d := range droplets {
			publicIPv4 := ""
			privateIPv4 := ""
			publicIPv6 := ""
			if d.Networks != nil {
				for _, v4 := range d.Networks.V4 {
					if v4.Type == "public" && publicIPv4 == "" {
						publicIPv4 = v4.IPAddress
					}
					if v4.Type == "private" && privateIPv4 == "" {
						privateIPv4 = v4.IPAddress
					}
				}
				for _, v6 := range d.Networks.V6 {
					if v6.Type == "public" && publicIPv6 == "" {
						publicIPv6 = v6.IPAddress
					}
				}
			}

			tags := make([]interface{}, len(d.Tags))
			for i, t := range d.Tags {
				tags[i] = t
			}

			features := make([]interface{}, len(d.Features))
			for i, f := range d.Features {
				features[i] = f
			}

			backupsEnabled := false
			monitoringEnabled := false
			for _, f := range d.Features {
				if f == "backups" {
					backupsEnabled = true
				}
				if f == "monitoring" {
					monitoringEnabled = true
				}
			}

			imageDict := map[string]interface{}{}
			if d.Image != nil {
				imageDict["id"] = float64(d.Image.ID)
				imageDict["name"] = d.Image.Name
				imageDict["distribution"] = d.Image.Distribution
				imageDict["slug"] = d.Image.Slug
			}

			regionSlug := ""
			if d.Region != nil {
				regionSlug = d.Region.Slug
			}
			sizeSlug := ""
			if d.Size != nil {
				sizeSlug = d.Size.Slug
			}

			res, err := CreateResource(r.MqlRuntime, "digitalocean.droplet", map[string]*llx.RawData{
				"id":                llx.IntData(int64(d.ID)),
				"name":              llx.StringData(d.Name),
				"memory":            llx.IntData(int64(d.Memory)),
				"vcpus":             llx.IntData(int64(d.Vcpus)),
				"disk":              llx.IntData(int64(d.Disk)),
				"region":            llx.StringData(regionSlug),
				"size":              llx.StringData(sizeSlug),
				"status":            llx.StringData(d.Status),
				"locked":            llx.BoolData(d.Locked),
				"createdAt":         llx.TimeDataPtr(parseDoTime(d.Created)),
				"publicIpv4":        llx.StringData(publicIPv4),
				"privateIpv4":       llx.StringData(privateIPv4),
				"publicIpv6":        llx.StringData(publicIPv6),
				"tags":              llx.ArrayData(tags, "\x02"),
				"vpcUuid":           llx.StringData(d.VPCUUID),
				"features":          llx.ArrayData(features, "\x02"),
				"backupsEnabled":    llx.BoolData(backupsEnabled),
				"monitoringEnabled": llx.BoolData(monitoringEnabled),
				"image":             llx.DictData(imageDict),
			})
			if err != nil {
				return nil, err
			}
			// Cache the droplet's image for the typed baseImage() accessor.
			res.(*mqlDigitaloceanDroplet).image = d.Image
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			break
		}
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanDroplet) id() (string, error) {
	return "digitalocean.droplet/" + strconv.FormatInt(r.Id.Data, 10), nil
}

func (r *mqlDigitalocean) firewalls() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		firewalls, resp, err := client.Firewalls.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, fw := range firewalls {
			inbound := make([]interface{}, len(fw.InboundRules))
			for i, rule := range fw.InboundRules {
				m := map[string]interface{}{
					"protocol": rule.Protocol,
					"ports":    rule.PortRange,
				}
				if rule.Sources != nil {
					m["sourceAddresses"] = toStringSlice(rule.Sources.Addresses)
					m["sourceDropletIds"] = toIntSlice(rule.Sources.DropletIDs)
					m["sourceLoadBalancerUids"] = toStringSlice(rule.Sources.LoadBalancerUIDs)
					m["sourceKubernetesIds"] = toStringSlice(rule.Sources.KubernetesIDs)
					m["sourceTags"] = toStringSlice(rule.Sources.Tags)
				}
				inbound[i] = m
			}
			outbound := make([]interface{}, len(fw.OutboundRules))
			for i, rule := range fw.OutboundRules {
				m := map[string]interface{}{
					"protocol": rule.Protocol,
					"ports":    rule.PortRange,
				}
				if rule.Destinations != nil {
					m["destinationAddresses"] = toStringSlice(rule.Destinations.Addresses)
					m["destinationDropletIds"] = toIntSlice(rule.Destinations.DropletIDs)
					m["destinationLoadBalancerUids"] = toStringSlice(rule.Destinations.LoadBalancerUIDs)
					m["destinationKubernetesIds"] = toStringSlice(rule.Destinations.KubernetesIDs)
					m["destinationTags"] = toStringSlice(rule.Destinations.Tags)
				}
				outbound[i] = m
			}

			dropletIds := make([]interface{}, len(fw.DropletIDs))
			for i, id := range fw.DropletIDs {
				dropletIds[i] = int64(id)
			}
			tags := make([]interface{}, len(fw.Tags))
			for i, t := range fw.Tags {
				tags[i] = t
			}

			res, err := CreateResource(r.MqlRuntime, "digitalocean.firewall", map[string]*llx.RawData{
				"id":            llx.StringData(fw.ID),
				"name":          llx.StringData(fw.Name),
				"status":        llx.StringData(fw.Status),
				"createdAt":     llx.TimeDataPtr(parseDoTime(fw.Created)),
				"inboundRules":  llx.ArrayData(inbound, "\x13"),
				"outboundRules": llx.ArrayData(outbound, "\x13"),
				"dropletIds":    llx.ArrayData(dropletIds, "\x05"),
				"tags":          llx.ArrayData(tags, "\x02"),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			break
		}
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanFirewall) id() (string, error) {
	return "digitalocean.firewall/" + r.Id.Data, nil
}

func (r *mqlDigitalocean) databases() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		dbs, resp, err := client.Databases.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, db := range dbs {
			tags := make([]interface{}, len(db.Tags))
			for i, t := range db.Tags {
				tags[i] = t
			}

			mw := map[string]interface{}{}
			if db.MaintenanceWindow != nil {
				mw["day"] = db.MaintenanceWindow.Day
				mw["hour"] = db.MaintenanceWindow.Hour
				mw["pending"] = db.MaintenanceWindow.Pending
			}

			// The DigitalOcean API returns connection URIs that embed the admin
			// password, so we deliberately do not surface them on the resource.
			// Host/port are exposed separately for connectivity checks.
			connHost := ""
			connPort := int64(0)
			if db.Connection != nil {
				connHost = db.Connection.Host
				connPort = int64(db.Connection.Port)
			}
			privConnHost := ""
			privConnPort := int64(0)
			if db.PrivateConnection != nil {
				privConnHost = db.PrivateConnection.Host
				privConnPort = int64(db.PrivateConnection.Port)
			}

			dbNames := make([]interface{}, len(db.DBNames))
			for i, n := range db.DBNames {
				dbNames[i] = n
			}

			res, err := CreateResource(r.MqlRuntime, "digitalocean.database", map[string]*llx.RawData{
				"id":                    llx.StringData(db.ID),
				"name":                  llx.StringData(db.Name),
				"engine":                llx.StringData(db.EngineSlug),
				"version":               llx.StringData(db.VersionSlug),
				"numNodes":              llx.IntData(int64(db.NumNodes)),
				"size":                  llx.StringData(db.SizeSlug),
				"region":                llx.StringData(db.RegionSlug),
				"status":                llx.StringData(db.Status),
				"storageSizeMib":        llx.IntData(int64(db.StorageSizeMib)),
				"dbNames":               llx.ArrayData(dbNames, "\x02"),
				"createdAt":             llx.TimeData(db.CreatedAt),
				"projectId":             llx.StringData(db.ProjectID),
				"privateNetworkUuid":    llx.StringData(db.PrivateNetworkUUID),
				"tags":                  llx.ArrayData(tags, "\x02"),
				"maintenanceWindow":     llx.DictData(mw),
				"connectionHost":        llx.StringData(connHost),
				"connectionPort":        llx.IntData(connPort),
				"privateConnectionHost": llx.StringData(privConnHost),
				"privateConnectionPort": llx.IntData(privConnPort),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			break
		}
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanDatabase) id() (string, error) {
	return "digitalocean.database/" + r.Id.Data, nil
}

func (r *mqlDigitaloceanDatabase) firewallRules() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	rules, _, err := client.Databases.GetFirewallRules(context.Background(), r.Id.Data)
	if err != nil {
		return nil, err
	}

	var all []interface{}
	for _, rule := range rules {
		all = append(all, map[string]interface{}{
			"uuid":      rule.UUID,
			"type":      rule.Type,
			"value":     rule.Value,
			"createdAt": rule.CreatedAt.Format(time.RFC3339),
		})
	}
	return all, nil
}

func (r *mqlDigitalocean) domains() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		domains, resp, err := client.Domains.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, d := range domains {
			res, err := CreateResource(r.MqlRuntime, "digitalocean.domain", map[string]*llx.RawData{
				"name":     llx.StringData(d.Name),
				"ttl":      llx.IntData(int64(d.TTL)),
				"zoneFile": llx.StringData(d.ZoneFile),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			break
		}
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanDomain) id() (string, error) {
	return "digitalocean.domain/" + r.Name.Data, nil
}

func (r *mqlDigitaloceanDomain) records() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		records, resp, err := client.Domains.Records(context.Background(), r.Name.Data, opt)
		if err != nil {
			return nil, err
		}
		for _, rec := range records {
			res, err := CreateResource(r.MqlRuntime, "digitalocean.domain.record", map[string]*llx.RawData{
				"domainName": llx.StringData(r.Name.Data),
				"id":         llx.IntData(int64(rec.ID)),
				"type":       llx.StringData(rec.Type),
				"name":       llx.StringData(rec.Name),
				"data":       llx.StringData(rec.Data),
				"ttl":        llx.IntData(int64(rec.TTL)),
				"priority":   llx.IntData(int64(rec.Priority)),
				"port":       llx.IntData(int64(rec.Port)),
				"weight":     llx.IntData(int64(rec.Weight)),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			break
		}
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanDomainRecord) id() (string, error) {
	return "digitalocean.domain.record/" + r.DomainName.Data + "/" + strconv.FormatInt(r.Id.Data, 10), nil
}

func (r *mqlDigitalocean) volumes() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListVolumeParams{ListOptions: &godo.ListOptions{PerPage: 200}}
	for {
		volumes, resp, err := client.Storage.ListVolumes(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, v := range volumes {
			dropletIds := make([]interface{}, len(v.DropletIDs))
			for i, id := range v.DropletIDs {
				dropletIds[i] = int64(id)
			}
			tags := make([]interface{}, len(v.Tags))
			for i, t := range v.Tags {
				tags[i] = t
			}

			volRegion := ""
			if v.Region != nil {
				volRegion = v.Region.Slug
			}
			res, err := CreateResource(r.MqlRuntime, "digitalocean.volume", map[string]*llx.RawData{
				"id":              llx.StringData(v.ID),
				"name":            llx.StringData(v.Name),
				"sizeGigabytes":   llx.IntData(v.SizeGigaBytes),
				"region":          llx.StringData(volRegion),
				"description":     llx.StringData(v.Description),
				"filesystemType":  llx.StringData(v.FilesystemType),
				"filesystemLabel": llx.StringData(v.FilesystemLabel),
				"createdAt":       llx.TimeData(v.CreatedAt),
				"tags":            llx.ArrayData(tags, "\x02"),
				"dropletIds":      llx.ArrayData(dropletIds, "\x05"),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			break
		}
		opt.ListOptions.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanVolume) id() (string, error) {
	return "digitalocean.volume/" + r.Id.Data, nil
}

func (r *mqlDigitalocean) loadBalancers() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		lbs, resp, err := client.LoadBalancers.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, lb := range lbs {
			dropletIds := make([]interface{}, len(lb.DropletIDs))
			for i, id := range lb.DropletIDs {
				dropletIds[i] = int64(id)
			}
			tags := make([]interface{}, len(lb.Tags))
			for i, t := range lb.Tags {
				tags[i] = t
			}

			fwdRules := make([]interface{}, len(lb.ForwardingRules))
			for i, rule := range lb.ForwardingRules {
				fwdRules[i] = map[string]interface{}{
					"entryProtocol":  rule.EntryProtocol,
					"entryPort":      float64(rule.EntryPort),
					"targetProtocol": rule.TargetProtocol,
					"targetPort":     float64(rule.TargetPort),
					"certificateId":  rule.CertificateID,
					"tlsPassthrough": rule.TlsPassthrough,
				}
			}

			hc := map[string]interface{}{}
			if lb.HealthCheck != nil {
				hc["protocol"] = lb.HealthCheck.Protocol
				hc["port"] = float64(lb.HealthCheck.Port)
				hc["path"] = lb.HealthCheck.Path
				hc["checkIntervalSeconds"] = float64(lb.HealthCheck.CheckIntervalSeconds)
				hc["responseTimeoutSeconds"] = float64(lb.HealthCheck.ResponseTimeoutSeconds)
				hc["unhealthyThreshold"] = float64(lb.HealthCheck.UnhealthyThreshold)
				hc["healthyThreshold"] = float64(lb.HealthCheck.HealthyThreshold)
			}

			ss := map[string]interface{}{}
			if lb.StickySessions != nil {
				ss["type"] = lb.StickySessions.Type
				ss["cookieName"] = lb.StickySessions.CookieName
				ss["cookieTtlSeconds"] = float64(lb.StickySessions.CookieTtlSeconds)
			}

			lbRegion := ""
			if lb.Region != nil {
				lbRegion = lb.Region.Slug
			}
			res, err := CreateResource(r.MqlRuntime, "digitalocean.loadBalancer", map[string]*llx.RawData{
				"id":                           llx.StringData(lb.ID),
				"name":                         llx.StringData(lb.Name),
				"ip":                           llx.StringData(lb.IP),
				"status":                       llx.StringData(lb.Status),
				"region":                       llx.StringData(lbRegion),
				"createdAt":                    llx.TimeDataPtr(parseDoTime(lb.Created)),
				"algorithm":                    llx.StringData(lb.Algorithm),
				"redirectHttpToHttps":          llx.BoolData(lb.RedirectHttpToHttps),
				"enableProxyProtocol":          llx.BoolData(lb.EnableProxyProtocol),
				"enableBackendKeepalive":       llx.BoolData(lb.EnableBackendKeepalive),
				"vpcUuid":                      llx.StringData(lb.VPCUUID),
				"dropletIds":                   llx.ArrayData(dropletIds, "\x05"),
				"tags":                         llx.ArrayData(tags, "\x02"),
				"forwardingRules":              llx.ArrayData(fwdRules, "\x13"),
				"healthCheck":                  llx.DictData(hc),
				"stickySessions":               llx.DictData(ss),
				"disableLetsEncryptDnsRecords": llx.BoolDataPtr(lb.DisableLetsEncryptDNSRecords),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			break
		}
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanLoadBalancer) id() (string, error) {
	return "digitalocean.loadBalancer/" + r.Id.Data, nil
}

func (r *mqlDigitalocean) vpcs() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		vpcs, resp, err := client.VPCs.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, v := range vpcs {
			res, err := CreateResource(r.MqlRuntime, "digitalocean.vpc", map[string]*llx.RawData{
				"id":          llx.StringData(v.ID),
				"name":        llx.StringData(v.Name),
				"description": llx.StringData(v.Description),
				"ipRange":     llx.StringData(v.IPRange),
				"region":      llx.StringData(v.RegionSlug),
				"createdAt":   llx.TimeData(v.CreatedAt),
				"default":     llx.BoolData(v.Default),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			break
		}
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanVpc) id() (string, error) {
	return "digitalocean.vpc/" + r.Id.Data, nil
}

func (r *mqlDigitalocean) kubernetesClusters() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		clusters, resp, err := client.Kubernetes.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, c := range clusters {
			tags := make([]interface{}, len(c.Tags))
			for i, t := range c.Tags {
				tags[i] = t
			}

			mp := map[string]interface{}{}
			if c.MaintenancePolicy != nil {
				mp["startTime"] = c.MaintenancePolicy.StartTime
				mp["day"] = float64(c.MaintenancePolicy.Day)
			}

			status := ""
			if c.Status != nil {
				status = string(c.Status.State)
			}

			var ssoEnabled, ssoRequired bool
			var ssoIssuerURL, ssoClientID string
			if c.SSO != nil {
				ssoEnabled = c.SSO.Enabled
				ssoRequired = c.SSO.Required
				ssoIssuerURL = c.SSO.IssuerURL
				ssoClientID = c.SSO.ClientID
			}

			res, err := CreateResource(r.MqlRuntime, "digitalocean.kubernetes.cluster", map[string]*llx.RawData{
				"id":                llx.StringData(c.ID),
				"name":              llx.StringData(c.Name),
				"version":           llx.StringData(c.VersionSlug),
				"region":            llx.StringData(c.RegionSlug),
				"status":            llx.StringData(status),
				"createdAt":         llx.TimeData(c.CreatedAt),
				"updatedAt":         llx.TimeData(c.UpdatedAt),
				"clusterSubnet":     llx.StringData(c.ClusterSubnet),
				"serviceSubnet":     llx.StringData(c.ServiceSubnet),
				"vpcUuid":           llx.StringData(c.VPCUUID),
				"autoUpgrade":       llx.BoolData(c.AutoUpgrade),
				"surgeUpgrade":      llx.BoolData(c.SurgeUpgrade),
				"ha":                llx.BoolData(c.HA),
				"ssoEnabled":        llx.BoolData(ssoEnabled),
				"ssoRequired":       llx.BoolData(ssoRequired),
				"ssoIssuerUrl":      llx.StringData(ssoIssuerURL),
				"ssoClientId":       llx.StringData(ssoClientID),
				"tags":              llx.ArrayData(tags, "\x02"),
				"maintenancePolicy": llx.DictData(mp),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			break
		}
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanKubernetesCluster) id() (string, error) {
	return "digitalocean.kubernetes.cluster/" + r.Id.Data, nil
}

func (r *mqlDigitalocean) projects() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		projects, resp, err := client.Projects.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, p := range projects {
			res, err := CreateResource(r.MqlRuntime, "digitalocean.project", map[string]*llx.RawData{
				"id":          llx.StringData(p.ID),
				"name":        llx.StringData(p.Name),
				"description": llx.StringData(p.Description),
				"purpose":     llx.StringData(p.Purpose),
				"environment": llx.StringData(p.Environment),
				"createdAt":   llx.TimeDataPtr(parseDoTime(p.CreatedAt)),
				"updatedAt":   llx.TimeDataPtr(parseDoTime(p.UpdatedAt)),
				"isDefault":   llx.BoolData(p.IsDefault),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			break
		}
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanProject) id() (string, error) {
	return "digitalocean.project/" + r.Id.Data, nil
}

func (r *mqlDigitalocean) sshKeys() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		keys, resp, err := client.Keys.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			res, err := CreateResource(r.MqlRuntime, "digitalocean.sshKey", map[string]*llx.RawData{
				"id":          llx.IntData(int64(k.ID)),
				"name":        llx.StringData(k.Name),
				"fingerprint": llx.StringData(k.Fingerprint),
				"publicKey":   llx.StringData(k.PublicKey),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			break
		}
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanSshKey) id() (string, error) {
	return "digitalocean.sshKey/" + strconv.FormatInt(r.Id.Data, 10), nil
}

func (r *mqlDigitalocean) certificates() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		certs, resp, err := client.Certificates.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, c := range certs {
			dnsNames := make([]interface{}, len(c.DNSNames))
			for i, n := range c.DNSNames {
				dnsNames[i] = n
			}

			res, err := CreateResource(r.MqlRuntime, "digitalocean.certificate", map[string]*llx.RawData{
				"id":              llx.StringData(c.ID),
				"name":            llx.StringData(c.Name),
				"sha1Fingerprint": llx.StringData(c.SHA1Fingerprint),
				"state":           llx.StringData(c.State),
				"type":            llx.StringData(c.Type),
				"dnsNames":        llx.ArrayData(dnsNames, "\x02"),
				"notAfter":        llx.TimeDataPtr(parseDoTime(c.NotAfter)),
				"createdAt":       llx.TimeDataPtr(parseDoTime(c.Created)),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			break
		}
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanCertificate) id() (string, error) {
	return "digitalocean.certificate/" + r.Id.Data, nil
}
