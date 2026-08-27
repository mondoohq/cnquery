// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/os/connection/shared"
	"go.mondoo.com/mql/providers/os/detector"
	"go.mondoo.com/mql/providers/os/resources/packages"
	"go.mondoo.com/mql/providers/os/resources/updates"
	"go.mondoo.com/mql/providers/os/resources/windows"
)

// lastUpdateCache resolves the asset's newest update install once and shares it
// between lastUpdate, lastUpdateAge and lastUpdateSource, which would otherwise
// each pay for the same log read or package listing. It is embedded in both
// mqlOsInternal and mqlOsBaseInternal, since `os` and `os.base` carry the same
// three fields.
type lastUpdateCache struct {
	once   sync.Once
	update *updates.LastInstalledUpdate
	err    error
}

// get resolves the newest update install once and hands the same outcome to
// every field that asks.
//
// sync.Once rather than a mutex guarding a "fetched" flag: the executor resolves
// a resource's fields in separate goroutines, and reading such a flag outside
// the lock races with the write inside it. There is no happens-before edge
// between the two, so a reader can observe the flag set before the value it
// guards is visible.
//
// A failure is cached alongside a success. Each of the three fields resolves
// once through the runtime's own field cache, so retrying would buy at most two
// further attempts at a failure that is deterministic within a scan (an
// unreadable package database, a missing permission) while paying for the full
// package listing again.
func (c *lastUpdateCache) get(runtime *plugin.Runtime) (*updates.LastInstalledUpdate, error) {
	c.once.Do(func() {
		c.update, c.err = resolveLastInstalledUpdate(runtime)
	})
	return c.update, c.err
}

type mqlOsInternal struct {
	lastUpdateCache
}

type mqlOsBaseInternal struct {
	lastUpdateCache
}

// resolveLastInstalledUpdate finds the newest operating system update install
// recorded on the asset. rpm-based platforms and Windows are answered from
// resources that already hold the data, so neither pays for a second rpm
// database read or a second PowerShell round trip; everything else is read from
// its own files by the updates package.
func resolveLastInstalledUpdate(runtime *plugin.Runtime) (*updates.LastInstalledUpdate, error) {
	conn, ok := runtime.Connection.(shared.Connection)
	if !ok {
		return nil, nil
	}
	asset := conn.Asset()
	if asset == nil || asset.Platform == nil {
		return nil, nil
	}

	// Containers are out of scope, and the check comes before the dispatch so
	// no rpm listing or PowerShell round trip is paid for an asset that is
	// going to read null anyway.
	if isContainerAsset(conn, asset.Platform) {
		return nil, nil
	}

	switch {
	case isRpmPlatform(asset.Platform):
		return lastInstalledRpm(runtime)
	case asset.Platform.Name == "windows":
		return lastInstalledWindows(runtime)
	}
	return updates.ResolveLastInstalledUpdate(conn)
}

// isContainerAsset reports whether the asset is a container or a container
// image.
//
// A container is not patched, it is rebuilt, so the age of the newest install
// recorded inside it is the age of the image build rather than a statement
// about whether anyone is maintaining the workload. Reporting that as patch age
// would let a base image that has not moved in a year read as freshly patched
// on the day it was pulled, so these fields read null instead and the question
// is left to image provenance.
//
// Three signals because each covers what the others miss: Kind is what the
// platform resolver sets for a docker connection, the device type additionally
// catches an os-release VARIANT_ID of container (an image reached over a
// filesystem or tar connection, where Kind may be unset), and the connection
// types cover registry scans.
//
// Type_Tar and Type_FileSystem are deliberately absent. A tar can be a virtual
// machine export and a filesystem connection is routinely a mounted host root,
// so excluding them would null the field on assets that genuinely are patched.
func isContainerAsset(conn shared.Connection, pf *inventory.Platform) bool {
	return isContainerPlatform(pf) || isContainerConnection(conn.Type())
}

// isContainerPlatform reports whether the detected platform is a container or a
// container image.
func isContainerPlatform(pf *inventory.Platform) bool {
	if pf == nil {
		return false
	}
	if pf.Kind == "container" || pf.Kind == "container-image" {
		return true
	}
	return pf.Metadata[detector.MetadataDeviceType] == detector.DeviceTypeContainer
}

// isContainerConnection reports whether a connection type can only ever reach a
// container or a container image.
//
// Type_Tar and Type_FileSystem are deliberately absent: a tar can be a virtual
// machine export and a filesystem connection is routinely a mounted host root.
func isContainerConnection(t shared.ConnectionType) bool {
	switch t {
	case shared.Type_DockerContainer, shared.Type_DockerImage, shared.Type_DockerSnapshot,
		shared.Type_DockerRegistry, shared.Type_ContainerRegistry, shared.Type_RegistryImage,
		shared.Type_DockerFile:
		return true
	}
	return false
}

// isRpmPlatform reports whether the asset's packages come from rpm, mirroring
// the platforms packages.ResolveSystemPkgManagers hands to RpmPkgManager.
func isRpmPlatform(pf *inventory.Platform) bool {
	switch pf.Name {
	case "amazonlinux", "photon", "wrlinux", "bottlerocket", "azurelinux", "mageia":
		return true
	}
	return pf.IsFamily("redhat") || pf.IsFamily("euler") || pf.IsFamily("suse")
}

// rpmVendorAnchors are packages that only ever come from the operating system
// vendor. Their %{VENDOR} is what the vendor calls itself on this asset, which
// is how the OS vendor is identified without shipping a distribution-to-vendor
// table that goes stale and silently nulls the field on a distribution nobody
// added to it.
//
// Several anchors, unioned, because a distribution can ship more than one
// vendor string across its own packages (Amazon Linux uses both "Amazon Linux"
// and "Amazon.com"). Trusting a single anchor would drop the other spelling.
var rpmVendorAnchors = []string{"glibc", "bash", "coreutils", "systemd", "filesystem"}

// lastInstalledRpm takes the newest %{INSTALLTIME} across the rpms shipped by
// the operating system vendor. rpm records an install time per package, which
// makes this the most precise answer available on these platforms, and it is
// already parsed on both the command path and the static rpmdb path.
//
// dnf's own history database is not used: it is emptied on assets whose rpm
// database stays intact, and reading it would need sqlite over the connection's
// filesystem where the vendor is already in hand.
//
// The vendor match is what makes the timestamp mean patch state. Without it a
// third-party rpm - Docker CE, Grafana, a vendor's agent - counts as an
// operating system update, and those move far more often than a distribution
// does.
func lastInstalledRpm(runtime *plugin.Runtime) (*updates.LastInstalledUpdate, error) {
	obj, err := CreateResource(runtime, "packages", nil)
	if err != nil {
		return nil, err
	}

	list := obj.(*mqlPackages).GetList()
	if list.Error != nil {
		return nil, list.Error
	}

	newest, ok := newestVendorRpmInstall(list.Data)
	if !ok {
		return nil, nil
	}
	return &updates.LastInstalledUpdate{Time: newest.UTC(), Source: updates.LastUpdateSourceRpmDB}, nil
}

// newestVendorRpmInstall returns the newest install time across the rpms shipped
// by this asset's operating system vendor, and whether one was found.
func newestVendorRpmInstall(list []any) (time.Time, bool) {
	vendors := rpmOSVendors(list)
	if len(vendors) == 0 {
		// No anchor resolved a vendor, so no package can be attributed. An
		// unattributable answer is not a coarser answer, it is none.
		return time.Time{}, false
	}

	var newest time.Time
	for i := range list {
		pkg, ok := list[i].(*mqlPackage)
		if !ok {
			continue
		}
		// A rpm host can also carry snap, nix or flatpak packages, whose
		// install times say nothing about the OS.
		if pkg.Format.Data != packages.RpmPkgFormat {
			continue
		}
		if _, ok := vendors[normalizeVendor(pkg.Vendor.Data)]; !ok {
			continue
		}
		installed := pkg.InstallDate.Data
		if installed == nil || installed.IsZero() {
			continue
		}
		if installed.After(newest) {
			newest = *installed
		}
	}

	return newest, !newest.IsZero()
}

// rpmOSVendors returns the set of vendor strings the anchor packages carry on
// this asset.
func rpmOSVendors(list []any) map[string]struct{} {
	vendors := map[string]struct{}{}
	for i := range list {
		pkg, ok := list[i].(*mqlPackage)
		if !ok || pkg.Format.Data != packages.RpmPkgFormat {
			continue
		}
		name := pkg.Name.Data
		anchor := false
		for _, a := range rpmVendorAnchors {
			if name == a {
				anchor = true
				break
			}
		}
		if !anchor {
			continue
		}
		if v := normalizeVendor(pkg.Vendor.Data); v != "" {
			vendors[v] = struct{}{}
		}
	}
	return vendors
}

// normalizeVendor folds the case and trims the padding rpm vendor strings carry
// so that two spellings of the same vendor compare equal.
func normalizeVendor(vendor string) string {
	return strings.ToLower(strings.TrimSpace(vendor))
}

// lastInstalledWindows takes the newest Windows Update Agent history entry that
// patched the operating system itself.
//
// The registry's last-successful-install time is not used as a fallback. It is
// a bare timestamp with no classification attached, so it counts the Defender
// signature updates that land daily and the .NET and Office servicing this pass
// excludes; a host years behind on Windows would read as patched this morning.
// A connection that cannot reach the agent history reads null instead.
func lastInstalledWindows(runtime *plugin.Runtime) (*updates.LastInstalledUpdate, error) {
	obj, err := CreateResource(runtime, "windows.update", nil)
	if err != nil {
		return nil, err
	}
	wu := obj.(*mqlWindowsUpdate)

	installed := wu.GetInstalled()
	if installed.Error != nil {
		// A connection that cannot reach the agent history has no answer, and
		// no answer is null rather than an error: one Windows host that cannot
		// run PowerShell must not fail a fleet query. Log it, because without
		// that a permission problem or a blocked PowerShell looks identical to
		// a host that genuinely has no history.
		log.Debug().Err(installed.Error).
			Msg("mql[os.lastUpdate]> windows update agent history unavailable")
		return nil, nil
	}

	var newest time.Time
	for i := range installed.Data {
		entry, ok := installed.Data[i].(*mqlWindowsUpdateEntry)
		if !ok {
			continue
		}
		if !windows.IsOperatingSystemUpdate(entryCategories(entry), entry.Title.Data) {
			continue
		}
		date := entry.Date.Data
		if date == nil || date.IsZero() {
			continue
		}
		if date.After(newest) {
			newest = *date
		}
	}

	if newest.IsZero() {
		return nil, nil
	}
	return &updates.LastInstalledUpdate{Time: newest.UTC(), Source: updates.LastUpdateSourceWindowsUpdate}, nil
}

// entryCategories converts a history entry's categories to the string slice the
// classifier takes. A non-string element is skipped rather than stringified,
// since a category that is not a string is not a product name.
func entryCategories(entry *mqlWindowsUpdateEntry) []string {
	out := make([]string, 0, len(entry.Categories.Data))
	for _, c := range entry.Categories.Data {
		if s, ok := c.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

// lastUpdateAge turns an install time into the duration-typed time MQL uses for
// durations, matching how uptime is reported. A timestamp in the future (a
// skewed clock, or a log written in a zone ahead of the scanner's) reads as zero
// rather than as a negative age.
func lastUpdateAge(installed time.Time) *time.Time {
	age := time.Now().Unix() - installed.Unix()
	if age < 0 {
		age = 0
	}
	return MqlTime(llx.DurationToTime(age))
}

func (p *mqlOs) lastUpdate() (*time.Time, error) {
	update, err := p.get(p.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if update == nil {
		// Mark the field resolved-and-null so the runtime does not treat it as
		// unresolved and re-invoke this accessor on every read.
		p.LastUpdate.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return MqlTime(update.Time), nil
}

func (p *mqlOs) lastUpdateAge() (*time.Time, error) {
	update, err := p.get(p.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if update == nil {
		p.LastUpdateAge.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return lastUpdateAge(update.Time), nil
}

func (p *mqlOs) lastUpdateSource() (string, error) {
	update, err := p.get(p.MqlRuntime)
	if err != nil {
		return "", err
	}
	if update == nil {
		p.LastUpdateSource.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	return update.Source, nil
}

func (p *mqlOsBase) lastUpdate() (*time.Time, error) {
	update, err := p.get(p.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if update == nil {
		p.LastUpdate.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return MqlTime(update.Time), nil
}

func (p *mqlOsBase) lastUpdateAge() (*time.Time, error) {
	update, err := p.get(p.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if update == nil {
		p.LastUpdateAge.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return lastUpdateAge(update.Time), nil
}

func (p *mqlOsBase) lastUpdateSource() (string, error) {
	update, err := p.get(p.MqlRuntime)
	if err != nil {
		return "", err
	}
	if update == nil {
		p.LastUpdateSource.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	return update.Source, nil
}
