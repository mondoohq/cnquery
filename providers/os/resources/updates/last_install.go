// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package updates

import (
	"time"

	"go.mondoo.com/mql/providers/os/connection/shared"
	"go.mondoo.com/mql/providers/os/resources/date"
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
// other than the log itself.
//
// It comes from the asset: /etc/localtime, /etc/timezone and the rest, all read
// through the connection's filesystem, so a container image or a mounted
// snapshot answers as well as a running host. That matters more than it first
// appears, because an image is almost always UTC while the machine scanning it
// rarely is, and reading the log in the scanner's zone would shift every
// timestamp by that offset.
//
// An asset carrying no zone information at all falls back to UTC, not to the
// scanner's zone. A Unix system with neither /etc/localtime nor /etc/timezone
// runs in UTC, which is what the stripped container images that reach this
// fallback actually do, so UTC is a fact about the asset where the scanner's
// zone is only ever a guess about it.
func assetTimeZone(conn shared.Connection) *time.Location {
	if loc := date.LocationFromFS(conn.FileSystem()); loc != nil {
		return loc
	}
	return time.UTC
}
