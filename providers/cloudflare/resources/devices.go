// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
)

func (c *mqlCloudflareOneDevice) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareOne) devices() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	records, err := conn.Cf.ListTeamsDevices(context.TODO(), c.AccountID)
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range records {
		rec := records[i]

		res, err := NewResource(c.MqlRuntime, "cloudflare.one.device", map[string]*llx.RawData{
			"id":               llx.StringData(rec.ID),
			"name":             llx.StringData(rec.Name),
			"deviceType":       llx.StringData(rec.DeviceType),
			"model":            llx.StringData(rec.Model),
			"manufacturer":     llx.StringData(rec.Manufacturer),
			"serialNumber":     llx.StringData(rec.SerialNumber),
			"macAddress":       llx.StringData(rec.MacAddress),
			"ip":               llx.StringData(rec.IP),
			"osVersion":        llx.StringData(rec.OSVersion),
			"osDistroName":     llx.StringData(rec.OSDistroName),
			"osDistroRevision": llx.StringData(rec.OsDistroRevision),
			"version":          llx.StringData(rec.Version),
			"deleted":          llx.BoolData(rec.Deleted),
			"created":          llx.TimeData(parseRFC3339(rec.Created)),
			"updated":          llx.TimeData(parseRFC3339(rec.Updated)),
			"lastSeen":         llx.TimeData(parseRFC3339(rec.LastSeen)),
			"revokedAt":        llx.TimeData(parseRFC3339(rec.RevokedAt)),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}

	return result, nil
}

func (c *mqlCloudflareOneDevicePostureRule) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareOne) devicePostureRules() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	records, _, err := conn.Cf.DevicePostureRules(context.TODO(), c.AccountID)
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range records {
		rec := records[i]

		res, err := NewResource(c.MqlRuntime, "cloudflare.one.devicePostureRule", map[string]*llx.RawData{
			"id":          llx.StringData(rec.ID),
			"name":        llx.StringData(rec.Name),
			"type":        llx.StringData(rec.Type),
			"description": llx.StringData(rec.Description),
			"schedule":    llx.StringData(rec.Schedule),
			"expiration":  llx.StringData(rec.Expiration),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}

	return result, nil
}

func (c *mqlCloudflareOneDevicePostureIntegration) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareOne) devicePostureIntegrations() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	records, _, err := conn.Cf.DevicePostureIntegrations(context.TODO(), c.AccountID)
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range records {
		rec := records[i]

		res, err := NewResource(c.MqlRuntime, "cloudflare.one.devicePostureIntegration", map[string]*llx.RawData{
			"id":       llx.StringData(rec.IntegrationID),
			"name":     llx.StringData(rec.Name),
			"type":     llx.StringData(rec.Type),
			"interval": llx.StringData(rec.Interval),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}

	return result, nil
}

// parseRFC3339 parses an RFC3339 timestamp string, returning zero time for empty strings.
func parseRFC3339(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
