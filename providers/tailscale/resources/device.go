// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/tailscale/connection"
	"go.mondoo.com/mql/v13/types"
	tsclient "tailscale.com/client/tailscale/v2"
)

// The resource keys devices by the legacy numeric id rather than nodeId. It is
// the value carried in discovered device platform ids, so changing it would
// re-identify every previously scanned device asset. Both address the same
// device through the API, and nodeId is exposed as its own field.
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

	// GetWithAllFields rather than Get: the subnet routes, posture identity,
	// SSH state and connectivity details are omitted from the default field
	// set, and every one of them backs a resource field.
	device, err := conn.Client().Devices().GetWithAllFields(context.Background(), id)
	if err != nil {
		return nil, nil, err
	}

	resource, err := createTailscaleDeviceResource(runtime, device)
	if err != nil {
		return nil, nil, err
	}

	return args, resource.(*mqlTailscaleDevice), nil
}

// lastSeenTime converts a device's last-seen timestamp for MQL. Tailscale
// leaves it unset while the device holds a connection to the coordination
// server, which the resource reports as null rather than substituting a
// timestamp the API never gave us.
func lastSeenTime(t *tsclient.Time) *time.Time {
	if t == nil {
		return nil
	}
	return &t.Time
}

func createTailscaleDeviceResource(runtime *plugin.Runtime, device *tsclient.Device) (plugin.Resource, error) {
	// The fields below are only populated when the device was fetched with the
	// full field set. A device fetched without them arrives with nil pointers,
	// so each block degrades to a zero value rather than dereferencing.
	var distroName, distroVersion, distroCodeName string
	if device.Distro != nil {
		distroName = device.Distro.Name
		distroVersion = device.Distro.Version
		distroCodeName = device.Distro.CodeName
	}

	var serialNumbers, hardwareAddresses []string
	var postureIdentityDisabled bool
	if device.PostureIdentity != nil {
		serialNumbers = device.PostureIdentity.SerialNumbers
		hardwareAddresses = device.PostureIdentity.HardwareAddresses
		postureIdentityDisabled = device.PostureIdentity.Disabled
	}

	var endpoints []string
	var derpRelay string
	var mappingVariesByDestIP bool
	if device.ClientConnectivity != nil {
		endpoints = device.ClientConnectivity.Endpoints
		derpRelay = device.ClientConnectivity.DERP
		mappingVariesByDestIP = device.ClientConnectivity.MappingVariesByDestIP
	}

	return CreateResource(runtime, "tailscale.device", map[string]*llx.RawData{
		"id":                        llx.StringData(device.ID),
		"nodeId":                    llx.StringData(device.NodeID),
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
		"isEphemeral":               llx.BoolData(device.IsEphemeral),
		"connectedToControl":        llx.BoolData(device.ConnectedToControl),
		"sshEnabled":                llx.BoolData(device.SSHEnabled),
		"keyExpiryDisabled":         llx.BoolData(device.KeyExpiryDisabled),
		"updateAvailable":           llx.BoolData(device.UpdateAvailable),
		"createdAt":                 llx.TimeData(device.Created.Time),
		"expiresAt":                 llx.TimeData(device.Expires.Time),
		"lastSeenAt":                llx.TimeDataPtr(lastSeenTime(device.LastSeen)),
		"tags":                      llx.ArrayData(convert.SliceAnyToInterface(device.Tags), types.String),
		"addresses":                 llx.ArrayData(convert.SliceAnyToInterface(device.Addresses), types.String),
		"advertisedRoutes":          llx.ArrayData(convert.SliceAnyToInterface(device.AdvertisedRoutes), types.String),
		"enabledRoutes":             llx.ArrayData(convert.SliceAnyToInterface(device.EnabledRoutes), types.String),
		"distroName":                llx.StringData(distroName),
		"distroVersion":             llx.StringData(distroVersion),
		"distroCodeName":            llx.StringData(distroCodeName),
		"postureSerialNumbers":      llx.ArrayData(convert.SliceAnyToInterface(serialNumbers), types.String),
		"postureHardwareAddresses":  llx.ArrayData(convert.SliceAnyToInterface(hardwareAddresses), types.String),
		"postureIdentityDisabled":   llx.BoolData(postureIdentityDisabled),
		"endpoints":                 llx.ArrayData(convert.SliceAnyToInterface(endpoints), types.String),
		"derpRelay":                 llx.StringData(derpRelay),
		"mappingVariesByDestIp":     llx.BoolData(mappingVariesByDestIP),
	})
}
