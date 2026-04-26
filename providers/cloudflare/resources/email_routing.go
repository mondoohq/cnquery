// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
)

type mqlCloudflareZoneEmailRoutingInternal struct {
	zoneID   string
	zoneName string

	dnsLock    sync.Mutex
	dnsFetched bool
	dnsCache   []cloudflare.DNSRecord
	dnsErr     error
}

func (c *mqlCloudflareZone) emailRouting() (*mqlCloudflareZoneEmailRouting, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	settings, err := conn.Cf.GetEmailRoutingSettings(context.TODO(), &cloudflare.ResourceContainer{
		Identifier: c.Id.Data,
	})
	if err != nil {
		var notFound *cloudflare.NotFoundError
		var authN *cloudflare.AuthenticationError
		var authZ *cloudflare.AuthorizationError
		if errors.As(err, &notFound) || errors.As(err, &authN) || errors.As(err, &authZ) {
			c.EmailRouting.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}

	res, err := CreateResource(c.MqlRuntime, "cloudflare.zone.emailRouting", map[string]*llx.RawData{
		"__id":       llx.StringData("cloudflare.zone.emailRouting@" + c.Id.Data),
		"enabled":    llx.BoolData(settings.Enabled),
		"status":     llx.StringData(settings.Status),
		"name":       llx.StringData(settings.Name),
		"createdAt":  llx.TimeDataPtr(settings.Created),
		"modifiedAt": llx.TimeDataPtr(settings.Modified),
	})
	if err != nil {
		return nil, err
	}

	er := res.(*mqlCloudflareZoneEmailRouting)
	er.zoneID = c.Id.Data
	er.zoneName = c.GetName().Data
	return er, nil
}

// dnsRecords returns the suggested DNS records reported by the email-routing
// settings endpoint. These are the MX/SPF/DKIM records Cloudflare expects in
// place for routing to work. We expose them as raw dicts so callers can
// inspect record type, name, content, and TTL alongside `mxConfigured`,
// `spfConfigured`, and `dmarcConfigured` derived booleans.
func (c *mqlCloudflareZoneEmailRouting) dnsRecords() ([]any, error) {
	records, err := c.fetchSuggestedDNSRecords()
	if err != nil {
		return nil, err
	}

	result := make([]any, 0, len(records))
	for i := range records {
		r := records[i]
		entry := map[string]any{
			"type":     r.Type,
			"name":     r.Name,
			"content":  r.Content,
			"ttl":      int64(r.TTL),
			"priority": int64(0),
		}
		if r.Priority != nil {
			entry["priority"] = int64(*r.Priority)
		}
		result = append(result, entry)
	}
	return result, nil
}

func (c *mqlCloudflareZoneEmailRouting) mxConfigured() (bool, error) {
	records, err := c.fetchSuggestedDNSRecords()
	if err != nil {
		return false, err
	}
	for _, r := range records {
		if strings.EqualFold(r.Type, "MX") {
			return true, nil
		}
	}
	return false, nil
}

func (c *mqlCloudflareZoneEmailRouting) spfConfigured() (bool, error) {
	records, err := c.fetchSuggestedDNSRecords()
	if err != nil {
		return false, err
	}
	for _, r := range records {
		if strings.EqualFold(r.Type, "TXT") && strings.Contains(strings.ToLower(r.Content), "v=spf1") {
			return true, nil
		}
	}
	return false, nil
}

// dmarcConfigured queries the zone's existing DNS records (not the suggested
// ones — Cloudflare doesn't auto-generate DMARC for email routing) and looks
// for a `_dmarc.<zone>` TXT record starting with `v=DMARC1`.
func (c *mqlCloudflareZoneEmailRouting) dmarcConfigured() (bool, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	if c.zoneID == "" {
		return false, nil
	}

	records, _, err := conn.Cf.ListDNSRecords(context.TODO(),
		&cloudflare.ResourceContainer{Identifier: c.zoneID},
		cloudflare.ListDNSRecordsParams{Type: "TXT"})
	if err != nil {
		return false, err
	}

	dmarcName := "_dmarc"
	if c.zoneName != "" {
		dmarcName = "_dmarc." + c.zoneName
	}

	for _, r := range records {
		if !strings.EqualFold(r.Name, dmarcName) {
			continue
		}
		if strings.Contains(strings.ToLower(r.Content), "v=dmarc1") {
			return true, nil
		}
	}
	return false, nil
}

func (c *mqlCloudflareZoneEmailRouting) fetchSuggestedDNSRecords() ([]cloudflare.DNSRecord, error) {
	if c.dnsFetched {
		return c.dnsCache, c.dnsErr
	}
	c.dnsLock.Lock()
	defer c.dnsLock.Unlock()
	if c.dnsFetched {
		return c.dnsCache, c.dnsErr
	}

	if c.zoneID == "" {
		c.dnsFetched = true
		return nil, nil
	}

	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)
	records, err := conn.Cf.GetEmailRoutingDNSSettings(context.TODO(), &cloudflare.ResourceContainer{
		Identifier: c.zoneID,
	})
	c.dnsCache = records
	c.dnsErr = err
	c.dnsFetched = true
	return c.dnsCache, c.dnsErr
}
