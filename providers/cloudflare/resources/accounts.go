// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"errors"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
)

func (c *mqlCloudflareAccount) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func initCloudflareAccount(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		resource, err := CreateResource(runtime, "cloudflare.account", args)
		if err != nil {
			return nil, nil, err
		}
		return args, resource, nil
	}

	conn := runtime.Connection.(*connection.CloudflareConnection)

	if conn.Asset() == nil {
		return nil, nil, errors.New("no asset found")
	}

	for _, platformId := range conn.Asset().PlatformIds {
		if accId := strings.TrimPrefix(platformId, connection.PlatformIdCloudflareAccount); accId != platformId {
			acc, ok := runtime.Resources.Get("cloudflare.account\x00" + accId)
			if !ok {
				return nil, nil, errors.New("account not found")
			}

			return args, acc, nil
		}
	}
	return nil, nil, errors.New("account not found or asset not set")
}

func (c *mqlCloudflareAccountRole) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareAccount) roles() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	records, err := conn.Cf.ListAccountRoles(context.TODO(), &cloudflare.ResourceContainer{
		Identifier: c.Id.Data,
		Level:      cloudflare.AccountRouteLevel,
	}, cloudflare.ListAccountRolesParams{})
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range records {
		rec := records[i]

		res, err := NewResource(c.MqlRuntime, "cloudflare.account.role", map[string]*llx.RawData{
			"id":          llx.StringData(rec.ID),
			"name":        llx.StringData(rec.Name),
			"description": llx.StringData(rec.Description),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}

	return result, nil
}
