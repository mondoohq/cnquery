// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package updates

import (
	"time"

	"go.mondoo.com/mql/providers/os/connection/shared"
	"go.mondoo.com/mql/providers/os/resources/date"
)

// Sources a last-installed-update timestamp can come from, reported verbatim
// through os.lastUpdateSource.
//
// Every one of them names an exact install event that could be attributed to
// the operating system vendor. A record that cannot carry that attribution is
// not a weaker source, it is not a source: dpkg.log names no repository and no
// command, and the mtime of the dpkg status file does not even say whether the
// change was an install, so both read null rather than answering a different
// question than the one asked.
//
// They differ in how tightly the attribution holds, which is why they stay
// distinct rather than collapsing into one string. A policy that only accepts a
// vendor security channel can require LastUpdateSourceAptSecurity, while a
// looser one takes any non-null answer.
const (
	// LastUpdateSourceRpmDB is the newest %{INSTALLTIME} across the rpms
	// carrying an operating system vendor's %{VENDOR}.
	LastUpdateSourceRpmDB = "rpmdb"
	// LastUpdateSourceAptSecurity is an unattended-upgrades run. Its default
	// Allowed-Origins is the distribution's -security pocket, which makes it
	// the closest an asset comes to recording "this was a vendor security
	// advisory" without an external advisory database.
	LastUpdateSourceAptSecurity = "apt-security"
	// LastUpdateSourceAptHistory is an operator-run full upgrade (apt upgrade,
	// dist-upgrade, full-upgrade). Vendor packages are what such a run
	// overwhelmingly moves, but it can also carry a third-party repository, so
	// it ranks below the security channel.
	LastUpdateSourceAptHistory = "apt-history"
	// LastUpdateSourceMacosHistory is the newest Software Update install in
	// /Library/Receipts/InstallHistory.plist.
	LastUpdateSourceMacosHistory = "macos-install-history"
	// LastUpdateSourceWindowsUpdate is the newest Windows Update Agent history
	// entry for an operating system product.
	LastUpdateSourceWindowsUpdate = "windows-update-agent"
)

// LastInstalledUpdate is the most recent operating system update install a
// platform records.
type LastInstalledUpdate struct {
	// Time the update was installed, always in UTC.
	Time time.Time
	// Source is one of the LastUpdateSource* constants.
	Source string
}

// ResolveLastInstalledUpdate reads the newest operating system update install
// recorded by the asset's own update mechanism. It covers the platforms whose
// record lives in a file: the apt history log on Debian, and the install
// history plist on macOS. rpm-based platforms and Windows are resolved by the
// os resource instead, because both already have the answer in a resource the
// runtime has likely cached (`packages` carries %{INSTALLTIME} and %{VENDOR}
// for every rpm, and `windows.update` carries the update agent history), and
// going through it avoids a second read of the rpm database or a second
// PowerShell round trip.
//
// A platform that keeps no attributable record is not an error: the result is
// nil and os.lastUpdate reads null. A fleet query has to stay usable on the
// platforms it does not cover.
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

// assetTimeZone returns the zone the Debian update logs are read in. apt writes
// local time with no offset, so the zone has to come from somewhere other than
// the log itself.
//
// It comes from the asset: /etc/localtime, /etc/timezone and the rest, all read
// through the connection's filesystem, so a mounted snapshot answers as well as
// a running host. That matters more than it first appears, because the machine
// scanning a snapshot is rarely in the zone of the machine it came from, and
// reading the log in the scanner's zone would shift every timestamp by that
// offset.
//
// An asset carrying no zone information at all falls back to UTC, not to the
// scanner's zone. A Unix system with neither /etc/localtime nor /etc/timezone
// runs in UTC, so UTC is a fact about the asset where the scanner's zone is
// only ever a guess about it.
func assetTimeZone(conn shared.Connection) *time.Location {
	if loc := date.LocationFromFS(conn.FileSystem()); loc != nil {
		return loc
	}
	return time.UTC
}
