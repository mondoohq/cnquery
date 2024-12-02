// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"errors"
	"strings"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/cloudflare/connection"
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
