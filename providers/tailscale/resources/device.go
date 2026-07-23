// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"
	"sync/atomic"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/tailscale/connection"
	"go.mondoo.com/mql/v13/types"
)

// mqlTailscaleDeviceInternal caches subnet route lookups so the advertised
// and enabled route accessors share a single SubnetRoutes API call.
// routesFetched is atomic because those two accessors are distinct MQL fields,
// which the runtime may resolve concurrently.
type mqlTailscaleDeviceInternal struct {
	routesLock    sync.Mutex
	routesFetched atomic.Bool
	routes        *tsclient.DeviceRoutes
}

func (r *mqlTailscaleDevice) id() (string, error) {
	return "tailscale/device/" + r.Id.Data, nil
}

func initTailscaleDevice(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.TailscaleConnection)

	// On a discovered device asset the device is implied by the asset itself, so
	// a bare `tailscale.device` resolves without an explicit id argument.
	args = withDefaultArg(args, "id", connection.DeviceIdFromAsset(conn.Asset()))

	id, err := requiredStringArg(args, "id")
	if err != nil {
		return nil, nil, err
	}

	device, err := conn.Client().Devices().Get(context.Background(), id)
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

func (d *mqlTailscaleDevice) fetchRoutes() (*tsclient.DeviceRoutes, error) {
	if d.routesFetched.Load() {
		return d.routes, nil
	}
	d.routesLock.Lock()
	defer d.routesLock.Unlock()
	if d.routesFetched.Load() {
		return d.routes, nil
	}
	conn := d.MqlRuntime.Connection.(*connection.TailscaleConnection)
	routes, err := conn.Client().Devices().SubnetRoutes(context.Background(), d.Id.Data)
	if err != nil {
		return nil, err
	}
	d.routes = routes
	d.routesFetched.Store(true)
	return d.routes, nil
}

func (d *mqlTailscaleDevice) advertisedRoutes() ([]any, error) {
	r, err := d.fetchRoutes()
	if err != nil {
		return nil, err
	}
	if r == nil {
		return []any{}, nil
	}
	return convert.SliceAnyToInterface(r.Advertised), nil
}

func (d *mqlTailscaleDevice) enabledRoutes() ([]any, error) {
	r, err := d.fetchRoutes()
	if err != nil {
		return nil, err
	}
	if r == nil {
		return []any{}, nil
	}
	return convert.SliceAnyToInterface(r.Enabled), nil
}
