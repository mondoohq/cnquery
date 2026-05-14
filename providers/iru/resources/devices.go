// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/iru/connection"
	"go.mondoo.com/mql/v13/providers/iru/connection/client"
	"go.mondoo.com/mql/v13/types"
)

// mqlIruDeviceInternal caches values needed by lazy-loaded computed
// fields on iru.device — primarily the blueprint and user IDs picked up
// from the summary row, plus a tag set assembled at creation time so
// that the Iru detail endpoint doesn't need to be re-hit for fields that
// the listing endpoint already populated.
type mqlIruDeviceInternal struct {
	cacheBlueprintId string
	cacheUserId      string
}

// devices walks the /devices listing and materializes each row as an
// iru.device resource. Posture, hardware, and volume fields that come
// from /devices/{id}/details are populated lazily by the computed
// methods further down this file — see GetFilevaultEnabled and friends.
func (r *mqlIru) devices() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.IruConnection)
	devices, err := conn.Client.ListDevices()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(devices))
	for i := range devices {
		dev, err := newMqlIruDevice(r.MqlRuntime, &devices[i])
		if err != nil {
			return nil, err
		}
		res = append(res, dev)
	}
	return res, nil
}

// newMqlIruDevice builds an iru.device from a /devices listing row and
// populates the Internal struct's blueprint/user IDs so the typed
// cross-references resolve. Shared between devices() and initIruDevice
// so a direct `iru.device(id: ...)` lookup hydrates the same fields the
// listing path does.
func newMqlIruDevice(runtime *plugin.Runtime, d *client.Device) (*mqlIruDevice, error) {
	tags := make([]any, 0, len(d.Tags))
	for _, t := range d.Tags {
		tags = append(tags, t)
	}
	raw := map[string]*llx.RawData{
		"id":             llx.StringData(d.DeviceID),
		"name":           llx.StringData(d.DeviceName),
		"serialNumber":   llx.StringData(d.SerialNumber),
		"udid":           llx.StringData(d.UDID),
		"assetTag":       llx.StringData(d.AssetTag),
		"model":          llx.StringData(d.Model),
		"platform":       llx.StringData(d.Platform),
		"osVersion":      llx.StringData(d.OSVersion),
		"agentVersion":   llx.StringData(d.AgentVersion),
		"agentInstalled": llx.BoolData(d.AgentInstallDate != ""),
		"lastCheckIn":    llx.TimeDataPtr(client.ParseTime(d.LastCheckIn)),
		"mdmEnabled":     llx.BoolData(strings.EqualFold(d.MDMEnabled, "true") || strings.EqualFold(d.MDMEnabled, "enabled")),
		"tags":           llx.ArrayData(tags, types.String),
	}
	item, err := CreateResource(runtime, "iru.device", raw)
	if err != nil {
		return nil, err
	}
	dev := item.(*mqlIruDevice)
	dev.cacheBlueprintId = d.BlueprintID
	if d.User != nil {
		dev.cacheUserId = d.User.ID
	}
	return dev, nil
}

func (c *mqlIruDevice) id() (string, error) {
	return "iru.device/" + c.Id.Data, nil
}

// initIruDevice supports `iru.device(id: ...)` lookups by ID. It resolves
// the device from the cached tenant-wide listing so the listing's summary
// fields (platform, osVersion, blueprint/user refs, tags, …) are populated
// — the /devices/{id}/details endpoint doesn't carry these and would leave
// `@defaults` fields blank.
func initIruDevice(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	idArg, ok := args["id"]
	if !ok || idArg == nil || idArg.Value == nil {
		return args, nil, nil
	}
	id, ok := idArg.Value.(string)
	if !ok || id == "" {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.IruConnection)
	devices, err := conn.Client.ListDevices()
	if err != nil {
		return args, nil, err
	}
	for i := range devices {
		if devices[i].DeviceID != id {
			continue
		}
		dev, err := newMqlIruDevice(runtime, &devices[i])
		if err != nil {
			return nil, nil, err
		}
		return nil, dev, nil
	}
	return nil, nil, fmt.Errorf("iru.device with id %q not found", id)
}

// getDetails is the single fetch every lazy posture/hardware/network field
// on iru.device shares. The connection caches the result per device ID so
// repeated calls (filevault, gatekeeper, volumes, …) all hit one network
// round trip.
func (c *mqlIruDevice) getDetails() (*client.DeviceDetails, error) {
	conn := c.MqlRuntime.Connection.(*connection.IruConnection)
	return conn.GetDeviceDetails(c.Id.Data)
}

// --- Hardware / OS fields populated from the detail endpoint ---------------

func (c *mqlIruDevice) modelIdentifier() (string, error) {
	d, err := c.getDetails()
	if err != nil {
		return "", err
	}
	return d.HardwareOverview.ModelIdentifier, nil
}

func (c *mqlIruDevice) processor() (string, error) {
	d, err := c.getDetails()
	if err != nil {
		return "", err
	}
	return d.HardwareOverview.ProcessorName, nil
}

func (c *mqlIruDevice) coreCount() (int64, error) {
	d, err := c.getDetails()
	if err != nil {
		return 0, err
	}
	return int64(d.HardwareOverview.NumberOfCores), nil
}

func (c *mqlIruDevice) memory() (int64, error) {
	d, err := c.getDetails()
	if err != nil {
		return 0, err
	}
	return d.HardwareOverview.MemoryBytes, nil
}

func (c *mqlIruDevice) appleSilicon() (bool, error) {
	d, err := c.getDetails()
	if err != nil {
		return false, err
	}
	return d.HardwareOverview.IsAppleSilicon, nil
}

func (c *mqlIruDevice) batteryCycleCount() (int64, error) {
	d, err := c.getDetails()
	if err != nil {
		return 0, err
	}
	return int64(d.HardwareOverview.BatteryCycleCount), nil
}

func (c *mqlIruDevice) wifiMac() (string, error) {
	d, err := c.getDetails()
	if err != nil {
		return "", err
	}
	return d.HardwareOverview.WifiMAC, nil
}

func (c *mqlIruDevice) ethernetMac() (string, error) {
	d, err := c.getDetails()
	if err != nil {
		return "", err
	}
	return d.HardwareOverview.EthernetMAC, nil
}

func (c *mqlIruDevice) bluetoothMac() (string, error) {
	d, err := c.getDetails()
	if err != nil {
		return "", err
	}
	return d.HardwareOverview.BluetoothMAC, nil
}

func (c *mqlIruDevice) lastEnrollment() (*time.Time, error) {
	d, err := c.getDetails()
	if err != nil {
		return nil, err
	}
	return client.ParseTime(d.General.LastEnrollment), nil
}

func (c *mqlIruDevice) supervised() (bool, error) {
	d, err := c.getDetails()
	if err != nil {
		return false, err
	}
	return d.General.IsSupervised, nil
}

// --- MDM / activation lock ------------------------------------------------

func (c *mqlIruDevice) userApprovedMdm() (bool, error) {
	d, err := c.getDetails()
	if err != nil {
		return false, err
	}
	return d.MDM.UserApprovedMDM, nil
}

func (c *mqlIruDevice) dep() (bool, error) {
	d, err := c.getDetails()
	if err != nil {
		return false, err
	}
	return d.MDM.DEPEnrolled || d.MDM.InstalledFromDEP, nil
}

func (c *mqlIruDevice) bootstrapTokenEscrowed() (bool, error) {
	d, err := c.getDetails()
	if err != nil {
		return false, err
	}
	return d.MDM.BootstrapTokenEscrowed, nil
}

func (c *mqlIruDevice) activationLockEnabled() (bool, error) {
	d, err := c.getDetails()
	if err != nil {
		return false, err
	}
	return d.ActivationLock.ActivationLockEnabled, nil
}

func (c *mqlIruDevice) activationLockBypassCodePresent() (bool, error) {
	d, err := c.getDetails()
	if err != nil {
		return false, err
	}
	return d.ActivationLock.BypassCodePresent, nil
}

// --- FileVault. Note we expose only `escrowed`/`type`, not the raw key.
// The full personal recovery key is sensitive enough that an audit
// schema shouldn't surface it by default; the escrow boolean answers
// the policy question without making the key queryable.
// --------------------------------------------------------------------------

func (c *mqlIruDevice) filevaultEnabled() (bool, error) {
	d, err := c.getDetails()
	if err != nil {
		return false, err
	}
	return d.FileVault.Enabled, nil
}

func (c *mqlIruDevice) filevaultRecoveryKeyEscrowed() (bool, error) {
	d, err := c.getDetails()
	if err != nil {
		return false, err
	}
	return d.FileVault.RecoveryKeyEscrowed, nil
}

func (c *mqlIruDevice) filevaultRecoveryKeyType() (string, error) {
	d, err := c.getDetails()
	if err != nil {
		return "", err
	}
	return d.FileVault.RecoveryKeyType, nil
}

// --- Security posture -----------------------------------------------------

func (c *mqlIruDevice) gatekeeperEnabled() (bool, error) {
	d, err := c.getDetails()
	if err != nil {
		return false, err
	}
	return d.SecurityInfo.GatekeeperEnabled, nil
}

func (c *mqlIruDevice) sipEnabled() (bool, error) {
	d, err := c.getDetails()
	if err != nil {
		return false, err
	}
	return d.SecurityInfo.SIPEnabled, nil
}

func (c *mqlIruDevice) firewallEnabled() (bool, error) {
	d, err := c.getDetails()
	if err != nil {
		return false, err
	}
	return d.SecurityInfo.FirewallEnabled, nil
}

func (c *mqlIruDevice) firewallStealthMode() (bool, error) {
	d, err := c.getDetails()
	if err != nil {
		return false, err
	}
	return d.SecurityInfo.FirewallStealthMode, nil
}

func (c *mqlIruDevice) autoUpdateEnabled() (bool, error) {
	d, err := c.getDetails()
	if err != nil {
		return false, err
	}
	return d.SecurityInfo.AutoUpdateEnabled, nil
}

func (c *mqlIruDevice) remoteDesktopEnabled() (bool, error) {
	d, err := c.getDetails()
	if err != nil {
		return false, err
	}
	return d.SecurityInfo.RemoteDesktopEnabled, nil
}

func (c *mqlIruDevice) screensaverLockEnabled() (bool, error) {
	d, err := c.getDetails()
	if err != nil {
		return false, err
	}
	return d.SecurityInfo.ScreensaverLockEnabled, nil
}

func (c *mqlIruDevice) findMyEnabled() (bool, error) {
	d, err := c.getDetails()
	if err != nil {
		return false, err
	}
	return d.SecurityInfo.FindMyEnabled, nil
}

func (c *mqlIruDevice) secureBootLevel() (string, error) {
	d, err := c.getDetails()
	if err != nil {
		return "", err
	}
	return d.SecurityInfo.SecureBootLevel, nil
}

func (c *mqlIruDevice) xprotectVersion() (string, error) {
	d, err := c.getDetails()
	if err != nil {
		return "", err
	}
	return d.SecurityInfo.XProtectVersion, nil
}

func (c *mqlIruDevice) mrtVersion() (string, error) {
	d, err := c.getDetails()
	if err != nil {
		return "", err
	}
	return d.SecurityInfo.MRTVersion, nil
}

// --- Network --------------------------------------------------------------

func (c *mqlIruDevice) ipAddress() (string, error) {
	d, err := c.getDetails()
	if err != nil {
		return "", err
	}
	return d.NetworkInfo.IPAddress, nil
}

func (c *mqlIruDevice) publicIpAddress() (string, error) {
	d, err := c.getDetails()
	if err != nil {
		return "", err
	}
	return d.NetworkInfo.PublicIP, nil
}

// --- Volumes / apps / profiles --------------------------------------------

func (c *mqlIruDevice) volumes() ([]any, error) {
	d, err := c.getDetails()
	if err != nil {
		return nil, err
	}
	res := make([]any, 0, len(d.Volumes))
	for _, v := range d.Volumes {
		res = append(res, map[string]any{
			"name":        v.Name,
			"identifier":  v.Identifier,
			"size_bytes":  v.SizeBytes,
			"free_bytes":  v.FreeBytes,
			"encrypted":   v.Encrypted,
			"boot_volume": v.BootVolume,
		})
	}
	return res, nil
}

func (c *mqlIruDevice) apps() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.IruConnection)
	apps, err := conn.GetDeviceApps(c.Id.Data)
	if err != nil {
		if client.IsAccessDenied(err) {
			log.Warn().Err(err).Str("device", c.Id.Data).Msg("iru> access denied to device apps; skipping")
			return nil, nil
		}
		return nil, err
	}
	res := make([]any, 0, len(apps))
	for _, a := range apps {
		item, err := CreateResource(c.MqlRuntime, "iru.app", map[string]*llx.RawData{
			"__id":           llx.StringData("iru.app/" + c.Id.Data + "/" + a.BundleID),
			"name":           llx.StringData(a.Name),
			"bundleId":       llx.StringData(a.BundleID),
			"version":        llx.StringData(a.Version),
			"shortVersion":   llx.StringData(a.ShortVersion),
			"path":           llx.StringData(a.Path),
			"teamIdentifier": llx.StringData(a.TeamID),
			"isAppleApp":     llx.BoolData(a.IsAppleApp),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, item)
	}
	return res, nil
}

func (c *mqlIruDevice) profiles() ([]any, error) {
	d, err := c.getDetails()
	if err != nil {
		return nil, err
	}
	profiles := d.InstalledProfiles

	res := make([]any, 0, len(profiles))
	for _, p := range profiles {
		item, err := CreateResource(c.MqlRuntime, "iru.profile", map[string]*llx.RawData{
			"__id":         llx.StringData("iru.profile/" + c.Id.Data + "/" + p.Identifier),
			"identifier":   llx.StringData(p.Identifier),
			"displayName":  llx.StringData(p.DisplayName),
			"organization": llx.StringData(p.Organization),
			"uuid":         llx.StringData(p.UUID),
			"installedBy":  llx.StringData(p.InstalledBy),
			"removable":    llx.BoolData(p.IsRemovable),
			"encrypted":    llx.BoolData(p.IsEncrypted),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, item)
	}
	return res, nil
}

// --- Typed cross-references -----------------------------------------------

func (c *mqlIruDevice) blueprint() (*mqlIruBlueprint, error) {
	if c.cacheBlueprintId == "" {
		c.Blueprint.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	bp, err := NewResource(c.MqlRuntime, "iru.blueprint", map[string]*llx.RawData{
		"id": llx.StringData(c.cacheBlueprintId),
	})
	if err != nil {
		return nil, err
	}
	return bp.(*mqlIruBlueprint), nil
}

func (c *mqlIruDevice) user() (*mqlIruUser, error) {
	if c.cacheUserId == "" {
		c.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	u, err := NewResource(c.MqlRuntime, "iru.user", map[string]*llx.RawData{
		"id": llx.StringData(c.cacheUserId),
	})
	if err != nil {
		return nil, err
	}
	return u.(*mqlIruUser), nil
}
