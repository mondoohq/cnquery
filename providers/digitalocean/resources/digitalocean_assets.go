// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/digitalocean/godo"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/digitalocean/connection"
)

// This file holds the shared resource arg-builders and the singular
// init functions for the resources the provider splits an account into
// (databases, Kubernetes clusters, load balancers, firewalls). Each
// builder maps a godo object to the MQL field set, and is used by both
// the account-level list accessor and the singular init — so a resource
// fetched on its own (digitalocean.database(id: "...") or a connected
// digitalocean-database asset) is populated identically to one listed
// from the account.

// databaseArgs maps a managed database cluster to its MQL fields.
func databaseArgs(db *godo.Database) map[string]*llx.RawData {
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

	return map[string]*llx.RawData{
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
	}
}

func initDigitaloceanDatabase(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.DigitaloceanConnection)
	id := stringArg(args, "id")
	if id == "" {
		id = conn.Conf.Options[connection.OptionDatabase]
	}
	if id == "" {
		return nil, nil, errors.New("digitalocean.database requires an id or a connected digitalocean-database asset")
	}
	db, _, err := conn.Client().Databases.Get(context.Background(), id)
	if err != nil {
		return nil, nil, err
	}
	return databaseArgs(db), nil, nil
}

// firewallArgs maps a cloud firewall to its MQL fields.
func firewallArgs(fw *godo.Firewall) map[string]*llx.RawData {
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

	return map[string]*llx.RawData{
		"id":            llx.StringData(fw.ID),
		"name":          llx.StringData(fw.Name),
		"status":        llx.StringData(fw.Status),
		"createdAt":     llx.TimeDataPtr(parseDoTime(fw.Created)),
		"inboundRules":  llx.ArrayData(inbound, "\x13"),
		"outboundRules": llx.ArrayData(outbound, "\x13"),
		"dropletIds":    llx.ArrayData(dropletIds, "\x05"),
		"tags":          llx.ArrayData(tags, "\x02"),
	}
}

func initDigitaloceanFirewall(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.DigitaloceanConnection)
	id := stringArg(args, "id")
	if id == "" {
		id = conn.Conf.Options[connection.OptionFirewall]
	}
	if id == "" {
		return nil, nil, errors.New("digitalocean.firewall requires an id or a connected digitalocean-firewall asset")
	}
	fw, _, err := conn.Client().Firewalls.Get(context.Background(), id)
	if err != nil {
		return nil, nil, err
	}
	return firewallArgs(fw), nil, nil
}

// loadBalancerArgs maps a load balancer to its MQL fields.
func loadBalancerArgs(lb *godo.LoadBalancer) map[string]*llx.RawData {
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

	return map[string]*llx.RawData{
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
	}
}

func initDigitaloceanLoadBalancer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.DigitaloceanConnection)
	id := stringArg(args, "id")
	if id == "" {
		id = conn.Conf.Options[connection.OptionLoadBalancer]
	}
	if id == "" {
		return nil, nil, errors.New("digitalocean.loadBalancer requires an id or a connected digitalocean-loadbalancer asset")
	}
	lb, _, err := conn.Client().LoadBalancers.Get(context.Background(), id)
	if err != nil {
		return nil, nil, err
	}
	return loadBalancerArgs(lb), nil, nil
}

// kubernetesClusterArgs maps a DOKS cluster to its MQL fields.
func kubernetesClusterArgs(c *godo.KubernetesCluster) map[string]*llx.RawData {
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

	return map[string]*llx.RawData{
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
	}
}

func initDigitaloceanKubernetesCluster(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.DigitaloceanConnection)
	id := stringArg(args, "id")
	if id == "" {
		id = conn.Conf.Options[connection.OptionKubernetes]
	}
	if id == "" {
		return nil, nil, errors.New("digitalocean.kubernetes.cluster requires an id or a connected digitalocean-kubernetes-cluster asset")
	}
	c, _, err := conn.Client().Kubernetes.Get(context.Background(), id)
	if err != nil {
		return nil, nil, err
	}
	return kubernetesClusterArgs(c), nil, nil
}

// stringArg returns the string value of args[key], or "" when absent.
func stringArg(args map[string]*llx.RawData, key string) string {
	if a, ok := args[key]; ok {
		if s, ok := a.Value.(string); ok {
			return s
		}
	}
	return ""
}
