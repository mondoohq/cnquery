// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/tailscale/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func (r *mqlTailscaleDevice) id() (string, error) {
	return "tailscale/device/" + r.Id.Data, nil
}

func initTailscaleDevice(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := args["id"]
	if !ok {
		// TODO try to get the id from the connection
		return nil, nil, errors.New("missing required argument 'id'")
	}

	conn := runtime.Connection.(*connection.TailscaleConnection)
	device, err := conn.Client().Devices().Get(context.Background(), id.Value.(string))
	if err != nil {
		return nil, nil, err
	}

	resource, err := createTailscaleDeviceResource(runtime, device)
	if err != nil {
		return nil, nil, err
	}

	return args, resource.(*mqlTailscaleDevice), nil
}

func createTailscaleDeviceResource(runtime *plugin.Runtime, device *tsclient.Device) (plugin.Resource, error) {
	return CreateResource(runtime, "tailscale.device", map[string]*llx.RawData{
		"id":                        llx.StringData(device.ID),
		"hostname":                  llx.StringData(device.Hostname),
		"os":                        llx.StringData(device.OS),
		"name":                      llx.StringData(device.Name),
		"user":                      llx.StringData(device.User),
		"clientVersion":             llx.StringData(device.ClientVersion),
		"machineKey":                llx.StringData(device.MachineKey),
		"nodeKey":                   llx.StringData(device.NodeKey),
		"tailnetLockError":          llx.StringData(device.TailnetLockError),
		"tailnetLockKey":            llx.StringData(device.TailnetLockKey),
		"blocksIncomingConnections": llx.BoolData(device.BlocksIncomingConnections),
		"authorized":                llx.BoolData(device.Authorized),
		"isExternal":                llx.BoolData(device.IsExternal),
		"keyExpiryDisabled":         llx.BoolData(device.KeyExpiryDisabled),
		"updateAvailable":           llx.BoolData(device.UpdateAvailable),
		"createdAt":                 llx.TimeData(device.Created.Time),
		"expiresAt":                 llx.TimeData(device.Expires.Time),
		"lastSeenAt":                llx.TimeData(device.LastSeen.Time),
		"tags":                      llx.ArrayData(convert.SliceAnyToInterface(device.Tags), types.String),
		"addresses":                 llx.ArrayData(convert.SliceAnyToInterface(device.Addresses), types.String),
	})
}
