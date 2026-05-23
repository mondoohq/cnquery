// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
	"go.mondoo.com/mql/v13/types"
)

func (c *mqlCloudflareOneGatewayRule) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareOne) gatewayRules() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	var result []any
	err := paginateRaw(context.TODO(), conn.Cf, fmt.Sprintf("/accounts/%s/gateway/rules", c.AccountID), func(raw json.RawMessage) error {
		var records []cloudflare.TeamsRule
		if err := json.Unmarshal(raw, &records); err != nil {
			return err
		}
		for i := range records {
			rec := records[i]

			filters := make([]any, len(rec.Filters))
			for j, f := range rec.Filters {
				filters[j] = string(f)
			}

			res, err := NewResource(c.MqlRuntime, "cloudflare.one.gatewayRule", map[string]*llx.RawData{
				"id":            llx.StringData(rec.ID),
				"name":          llx.StringData(rec.Name),
				"description":   llx.StringData(rec.Description),
				"action":        llx.StringData(string(rec.Action)),
				"enabled":       llx.BoolData(rec.Enabled),
				"precedence":    llx.IntData(int64(rec.Precedence)),
				"traffic":       llx.StringData(rec.Traffic),
				"identity":      llx.StringData(rec.Identity),
				"devicePosture": llx.StringData(rec.DevicePosture),
				"filters":       llx.ArrayData(filters, types.String),
				"version":       llx.IntData(int64(rec.Version)),
				"createdAt":     llx.TimeDataPtr(rec.CreatedAt),
				"updatedAt":     llx.TimeDataPtr(rec.UpdatedAt),
			})
			if err != nil {
				return err
			}
			result = append(result, res)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *mqlCloudflareOneList) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareOne) lists() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	var result []any
	err := paginateRaw(context.TODO(), conn.Cf, fmt.Sprintf("/accounts/%s/gateway/lists", c.AccountID), func(raw json.RawMessage) error {
		var records []cloudflare.TeamsList
		if err := json.Unmarshal(raw, &records); err != nil {
			return err
		}
		for i := range records {
			rec := records[i]
			res, err := NewResource(c.MqlRuntime, "cloudflare.one.list", map[string]*llx.RawData{
				"id":          llx.StringData(rec.ID),
				"name":        llx.StringData(rec.Name),
				"type":        llx.StringData(rec.Type),
				"description": llx.StringData(rec.Description),
				"count":       llx.IntData(int64(rec.Count)),
				"createdAt":   llx.TimeDataPtr(rec.CreatedAt),
				"updatedAt":   llx.TimeDataPtr(rec.UpdatedAt),
			})
			if err != nil {
				return err
			}
			result = append(result, res)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *mqlCloudflareOneLocation) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareOne) locations() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	var result []any
	err := paginateRaw(context.TODO(), conn.Cf, fmt.Sprintf("/accounts/%s/gateway/locations", c.AccountID), func(raw json.RawMessage) error {
		var records []cloudflare.TeamsLocation
		if err := json.Unmarshal(raw, &records); err != nil {
			return err
		}
		for i := range records {
			rec := records[i]

			ecsSupport := false
			if rec.ECSSupport != nil {
				ecsSupport = *rec.ECSSupport
			}

			res, err := NewResource(c.MqlRuntime, "cloudflare.one.location", map[string]*llx.RawData{
				"id":                    llx.StringData(rec.ID),
				"name":                  llx.StringData(rec.Name),
				"dohSubdomain":          llx.StringData(rec.Subdomain),
				"ip":                    llx.StringData(rec.Ip),
				"anonymizedLogsEnabled": llx.BoolData(rec.AnonymizedLogsEnabled),
				"clientDefault":         llx.BoolData(rec.ClientDefault),
				"ecsSupport":            llx.BoolData(ecsSupport),
				"createdAt":             llx.TimeDataPtr(rec.CreatedAt),
				"updatedAt":             llx.TimeDataPtr(rec.UpdatedAt),
			})
			if err != nil {
				return err
			}
			result = append(result, res)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *mqlCloudflareOneDlpProfile) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareOne) dlpProfiles() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	var result []any
	err := paginateRaw(context.TODO(), conn.Cf, fmt.Sprintf("/accounts/%s/dlp/profiles", c.AccountID), func(raw json.RawMessage) error {
		var records []cloudflare.DLPProfile
		if err := json.Unmarshal(raw, &records); err != nil {
			return err
		}
		for i := range records {
			rec := records[i]

			ocrEnabled := false
			if rec.OCREnabled != nil {
				ocrEnabled = *rec.OCREnabled
			}

			res, err := NewResource(c.MqlRuntime, "cloudflare.one.dlpProfile", map[string]*llx.RawData{
				"id":                llx.StringData(rec.ID),
				"name":              llx.StringData(rec.Name),
				"type":              llx.StringData(rec.Type),
				"description":       llx.StringData(rec.Description),
				"allowedMatchCount": llx.IntData(int64(rec.AllowedMatchCount)),
				"ocrEnabled":        llx.BoolData(ocrEnabled),
				"createdAt":         llx.TimeDataPtr(rec.CreatedAt),
				"updatedAt":         llx.TimeDataPtr(rec.UpdatedAt),
			})
			if err != nil {
				return err
			}
			result = append(result, res)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
