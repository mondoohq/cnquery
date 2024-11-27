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

func (c *mqlCloudflareZone) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func initCloudflareZone(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		resource, err := CreateResource(runtime, "cloudflare.zone", args)
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
		if zoneId := strings.TrimPrefix(platformId, connection.PlatformIdCloudflareZone); zoneId != platformId {
			zone, ok := runtime.Resources.Get("cloudflare.zone\x00" + zoneId)
			if !ok {
				return nil, nil, errors.New("zone not found")
			}

			return args, zone, nil
		}
	}
	return nil, nil, errors.New("zone not found or asset not set")
}

func (c *mqlCloudflareZoneAccount) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}
