// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package updates

import (
	"bytes"
	"io"
	"time"

	"go.mondoo.com/mql/providers/os/connection/shared"
	"howett.net/plist"
)

const (
	macosInstallHistoryPath = "/Library/Receipts/InstallHistory.plist"

	// macosConfigDataContentType marks the XProtect, MRT and Gatekeeper data
	// blobs that softwareupdated installs on its own schedule, often weekly.
	// They are not operating system updates, and counting them would report a
	// Mac months behind on macOS as having been patched days ago.
	macosConfigDataContentType = "config-data"
)

// macosSoftwareUpdateProcesses are the processes that install an update through
// Software Update. Everything else in the install history is a third-party
// package handed to installer(8), which says nothing about OS patch state.
var macosSoftwareUpdateProcesses = map[string]struct{}{
	"softwareupdated": {},
	"softwareupdate":  {},
}

func lastInstalledMacos(conn shared.Connection) (*LastInstalledUpdate, error) {
	f, err := conn.FileSystem().Open(macosInstallHistoryPath)
	if err != nil {
		// A Mac that has never installed anything through Software Update has
		// no history file. That is an absent record, not a failure.
		return nil, nil
	}
	defer f.Close()

	t, ok, err := ParseMacosInstallHistory(f)
	if err != nil || !ok {
		return nil, err
	}
	return &LastInstalledUpdate{Time: t, Source: LastUpdateSourceMacosHistory}, nil
}

type macosInstallHistoryEntry struct {
	Date           time.Time `plist:"date"`
	DisplayName    string    `plist:"displayName"`
	DisplayVersion string    `plist:"displayVersion"`
	ProcessName    string    `plist:"processName"`
	ContentType    string    `plist:"contentType"`
}

// ParseMacosInstallHistory returns the date of the newest Software Update
// install in /Library/Receipts/InstallHistory.plist, ignoring third-party
// installer runs and the config-data blobs. Dates in the plist are UTC.
func ParseMacosInstallHistory(input io.Reader) (time.Time, bool, error) {
	r, err := plistReadSeeker(input)
	if err != nil {
		return time.Time{}, false, err
	}

	var entries []macosInstallHistoryEntry
	if err := plist.NewDecoder(r).Decode(&entries); err != nil {
		return time.Time{}, false, err
	}

	var newest time.Time
	for i := range entries {
		e := entries[i]
		if _, ok := macosSoftwareUpdateProcesses[e.ProcessName]; !ok {
			continue
		}
		if e.ContentType == macosConfigDataContentType {
			continue
		}
		if e.Date.After(newest) {
			newest = e.Date
		}
	}

	if newest.IsZero() {
		return time.Time{}, false, nil
	}
	return newest.UTC(), true, nil
}

// plistReadSeeker gives the plist decoder the seekable reader it needs. A file
// opened from the connection's filesystem already seeks; a command's stdout
// does not, so it is buffered in memory first.
func plistReadSeeker(input io.Reader) (io.ReadSeeker, error) {
	if r, ok := input.(io.ReadSeeker); ok {
		return r, nil
	}
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}
