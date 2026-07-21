// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"fmt"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/zero_trust"
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
	iter := conn.Cf.ZeroTrust.Gateway.Rules.ListAutoPaging(context.TODO(), zero_trust.GatewayRuleListParams{
		AccountID: cloudflare.F(c.AccountID),
	})
	for iter.Next() {
		rec := iter.Current()

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
			"precedence":    llx.IntData(rec.Precedence),
			"traffic":       llx.StringData(rec.Traffic),
			"identity":      llx.StringData(rec.Identity),
			"devicePosture": llx.StringData(rec.DevicePosture),
			"filters":       llx.ArrayData(filters, types.String),
			"version":       llx.IntData(rec.Version),
			"createdAt":     timeOrNil(rec.CreatedAt),
			"updatedAt":     timeOrNil(rec.UpdatedAt),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}
	if err := iter.Err(); err != nil {
		return degradedList(err)
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
	iter := conn.Cf.ZeroTrust.Gateway.Lists.ListAutoPaging(context.TODO(), zero_trust.GatewayListListParams{
		AccountID: cloudflare.F(c.AccountID),
	})
	for iter.Next() {
		rec := iter.Current()

		res, err := NewResource(c.MqlRuntime, "cloudflare.one.list", map[string]*llx.RawData{
			"id":          llx.StringData(rec.ID),
			"name":        llx.StringData(rec.Name),
			"type":        llx.StringData(string(rec.Type)),
			"description": llx.StringData(rec.Description),
			"count":       llx.IntData(int64(rec.Count)),
			"createdAt":   timeOrNil(rec.CreatedAt),
			"updatedAt":   timeOrNil(rec.UpdatedAt),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}
	if err := iter.Err(); err != nil {
		return degradedList(err)
	}

	return result, nil
}

func (c *mqlCloudflareOneLocation) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

// gatewayLocation is the response shape of the gateway-locations endpoint. The
// cloudflare-go v6 typed Location no longer exposes anonymized_logs_enabled, so
// we read the endpoint via the client's generic Get to preserve that field.
type gatewayLocation struct {
	ID                    string    `json:"id"`
	Name                  string    `json:"name"`
	DOHSubdomain          string    `json:"doh_subdomain"`
	IP                    string    `json:"ip"`
	AnonymizedLogsEnabled bool      `json:"anonymized_logs_enabled"`
	ClientDefault         bool      `json:"client_default"`
	ECSSupport            bool      `json:"ecs_support"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (c *mqlCloudflareOne) locations() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	records, err := cfGetPaged[gatewayLocation](conn, fmt.Sprintf("accounts/%s/gateway/locations", c.AccountID))
	if err != nil {
		return degradedList(err)
	}

	var result []any
	for i := range records {
		rec := records[i]

		res, err := NewResource(c.MqlRuntime, "cloudflare.one.location", map[string]*llx.RawData{
			"id":                    llx.StringData(rec.ID),
			"name":                  llx.StringData(rec.Name),
			"dohSubdomain":          llx.StringData(rec.DOHSubdomain),
			"ip":                    llx.StringData(rec.IP),
			"anonymizedLogsEnabled": llx.BoolData(rec.AnonymizedLogsEnabled),
			"clientDefault":         llx.BoolData(rec.ClientDefault),
			"ecsSupport":            llx.BoolData(rec.ECSSupport),
			"createdAt":             timeOrNil(rec.CreatedAt),
			"updatedAt":             timeOrNil(rec.UpdatedAt),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
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
	iter := conn.Cf.ZeroTrust.DLP.Profiles.ListAutoPaging(context.TODO(), zero_trust.DLPProfileListParams{
		AccountID: cloudflare.F(c.AccountID),
	})
	for iter.Next() {
		rec := iter.Current()

		res, err := NewResource(c.MqlRuntime, "cloudflare.one.dlpProfile", map[string]*llx.RawData{
			"id":                llx.StringData(rec.ID),
			"name":              llx.StringData(rec.Name),
			"type":              llx.StringData(string(rec.Type)),
			"description":       llx.StringData(rec.Description),
			"allowedMatchCount": llx.IntData(rec.AllowedMatchCount),
			"ocrEnabled":        llx.BoolData(rec.OCREnabled),
			"createdAt":         timeOrNil(rec.CreatedAt),
			"updatedAt":         timeOrNil(rec.UpdatedAt),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}
	if err := iter.Err(); err != nil {
		return degradedList(err)
	}

	return result, nil
}
