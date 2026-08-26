// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/os/connection/shared"
	"go.mondoo.com/mql/providers/os/resources/packages"
	"go.mondoo.com/mql/providers/os/resources/updates"
)

// windowsDefinitionUpdates is the classification the Windows Update Agent gives
// Defender signature updates. They install daily, so counting them would report
// a host years behind on Windows as patched this morning. The string is the one
// windows.ClassifyUpdate emits.
const windowsDefinitionUpdates = "Definition Updates"

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

// resolveLastInstalledUpdate finds the newest update install recorded on the
// asset. rpm-based platforms and Windows are answered from resources that
// already hold the data, so neither pays for a second rpm database read or a
// second PowerShell round trip; everything else is read from its own files by
// the updates package.
func resolveLastInstalledUpdate(runtime *plugin.Runtime) (*updates.LastInstalledUpdate, error) {
	conn, ok := runtime.Connection.(shared.Connection)
	if !ok {
		return nil, nil
	}
	asset := conn.Asset()
	if asset == nil || asset.Platform == nil {
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

// isRpmPlatform reports whether the asset's packages come from rpm, mirroring
// the platforms packages.ResolveSystemPkgManagers hands to RpmPkgManager.
func isRpmPlatform(pf *inventory.Platform) bool {
	switch pf.Name {
	case "amazonlinux", "photon", "wrlinux", "bottlerocket", "azurelinux", "mageia":
		return true
	}
	return pf.IsFamily("redhat") || pf.IsFamily("euler") || pf.IsFamily("suse")
}

// lastInstalledRpm takes the newest %{INSTALLTIME} across the installed rpms.
// rpm records an install time per package, which makes this the most precise
// answer available on these platforms, and it is already parsed on both the
// command path and the static rpmdb path. dnf's own history database is not
// used: container images routinely ship with /var/lib/dnf emptied while the rpm
// database stays intact.
func lastInstalledRpm(runtime *plugin.Runtime) (*updates.LastInstalledUpdate, error) {
	obj, err := CreateResource(runtime, "packages", nil)
	if err != nil {
		return nil, err
	}

	list := obj.(*mqlPackages).GetList()
	if list.Error != nil {
		return nil, list.Error
	}

	var newest time.Time
	for i := range list.Data {
		pkg, ok := list.Data[i].(*mqlPackage)
		if !ok {
			continue
		}
		// A rpm host can also carry snap, nix or flatpak packages, whose
		// install times say nothing about the OS.
		if pkg.Format.Data != packages.RpmPkgFormat {
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

	if newest.IsZero() {
		return nil, nil
	}
	return &updates.LastInstalledUpdate{Time: newest.UTC(), Source: updates.LastUpdateSourceRpmDB}, nil
}

// lastInstalledWindows prefers the Windows Update Agent install history, which
// carries a classification per update and so lets Defender signature updates be
// dropped. When that history cannot be read it falls back to the registry's
// last-successful-install time, which is coarser: it counts the definition
// updates the history pass excludes, which the reported source makes visible.
func lastInstalledWindows(runtime *plugin.Runtime) (*updates.LastInstalledUpdate, error) {
	obj, err := CreateResource(runtime, "windows.update", nil)
	if err != nil {
		return nil, err
	}
	wu := obj.(*mqlWindowsUpdate)

	installed := wu.GetInstalled()
	if installed.Error != nil {
		// The fallback below still produces an answer, so this is not fatal. Log
		// it anyway: without it, a permission problem or a blocked PowerShell
		// looks identical to a host that genuinely has no agent history, and the
		// coarser registry source gets used with nothing to explain why.
		log.Debug().Err(installed.Error).
			Msg("mql[os.lastUpdate]> windows update agent history unavailable, falling back to the registry")
	} else {
		var newest time.Time
		for i := range installed.Data {
			entry, ok := installed.Data[i].(*mqlWindowsUpdateEntry)
			if !ok || entry.Classification.Data == windowsDefinitionUpdates {
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
		if !newest.IsZero() {
			return &updates.LastInstalledUpdate{Time: newest.UTC(), Source: updates.LastUpdateSourceWindowsUpdate}, nil
		}
	}

	config := wu.GetConfig()
	if config.Error != nil || config.Data == nil {
		return nil, nil
	}
	last := config.Data.GetLastInstallSuccess()
	if last.Error != nil || last.Data == nil || last.Data.IsZero() {
		return nil, nil
	}
	return &updates.LastInstalledUpdate{Time: last.Data.UTC(), Source: updates.LastUpdateSourceWindowsRegistry}, nil
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
