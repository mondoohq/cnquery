// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package updates

import (
	"time"

	"go.mondoo.com/mql/providers/os/connection/shared"
)

// Sources a last-installed-update timestamp can come from. They are reported
// verbatim through os.lastUpdateSource so a policy can tell an exact install
// event apart from a filesystem timestamp.
const (
	LastUpdateSourceRpmDB           = "rpmdb"
	LastUpdateSourceAptHistory      = "apt-history"
	LastUpdateSourceDpkgLog         = "dpkg-log"
	LastUpdateSourceDpkgStatus      = "dpkg-status-mtime"
	LastUpdateSourceMacosHistory    = "macos-install-history"
	LastUpdateSourceWindowsUpdate   = "windows-update-agent"
	LastUpdateSourceWindowsRegistry = "windows-update-registry"
)

// LastInstalledUpdate is the most recent update install a platform records.
type LastInstalledUpdate struct {
	// Time the update was installed, always in UTC.
	Time time.Time
	// Source is one of the LastUpdateSource* constants.
	Source string
}

// ResolveLastInstalledUpdate reads the newest update install recorded by the
// asset's own update mechanism. It covers the platforms whose record lives in a
// file: the apt and dpkg logs on Debian, and the install history plist on
// macOS. rpm-based platforms and Windows are resolved by the os resource
// instead, because both already have the answer in a resource the runtime has
// likely cached (`packages` carries %{INSTALLTIME} for every rpm, and
// `windows.update` carries the update agent history), and going through it
// avoids a second read of the rpm database or a second PowerShell round trip.
//
// A platform that keeps no such record is not an error: the result is nil and
// os.lastUpdate reads null. A fleet query has to stay usable on the platforms
// it does not cover.
func ResolveLastInstalledUpdate(conn shared.Connection) (*LastInstalledUpdate, error) {
	asset := conn.Asset()
	if asset == nil || asset.Platform == nil {
		return nil, nil
	}

	switch {
	case asset.Platform.IsFamily("debian"):
		return lastInstalledDebian(conn)
	case asset.Platform.Name == "macos":
		return lastInstalledMacos(conn)
	}
	return nil, nil
}

// assetTimeZone returns the zone the Debian update logs are read in. dpkg and
// apt write local time with no offset, so the zone has to come from somewhere
// other than the log, and we take the scanning host's. Scanning a machine in a
// different zone over SSH or through a mounted filesystem is then off by the
// difference between the two, which is noise at the day granularity these
// timestamps are used at, and only matters to an hours-based check.
func assetTimeZone() *time.Location {
	return time.Local
}
