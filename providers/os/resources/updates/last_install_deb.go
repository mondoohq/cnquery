// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package updates

import (
	"bufio"
	"compress/gzip"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/providers/os/connection/shared"
)

const (
	aptHistoryPath = "/var/log/apt/history.log"
	dpkgLogPath    = "/var/log/dpkg.log"
	dpkgStatusPath = "/var/lib/dpkg/status"

	// dpkgLogTimeLayout is the timestamp dpkg and apt write: local time with no
	// zone. apt separates the date from the time with two spaces and dpkg with
	// one, so the parser rejoins the fields before matching this layout.
	dpkgLogTimeLayout = "2006-01-02 15:04:05"

	// dpkgLogMaxLine caps a single line. An apt Upgrade: line names every
	// package in the run on one line, so on a dist-upgrade of a full desktop it
	// runs well past the scanner's default 64 KB. Past the limit the scan stops
	// and the rest of the log is lost, which would silently report an older
	// update as the newest one.
	dpkgLogMaxLine = 4 * 1024 * 1024

	// dpkgLogMaxRotations bounds how far back through logrotate's numbered
	// copies we look. Only the newest entry matters, so the walk stops at the
	// first file that yields one; it exists for the window right after a
	// rotation, when the live log is empty.
	dpkgLogMaxRotations = 12
)

// lastInstalledDebian reads the newest install or upgrade apt and dpkg
// recorded. Every source is a file read, so this answers the same way on a
// running host, a container image, and a mounted filesystem.
func lastInstalledDebian(conn shared.Connection) (*LastInstalledUpdate, error) {
	return lastInstalledDebianFS(conn.FileSystem(), assetTimeZone())
}

// lastInstalledDebianFS is lastInstalledDebian's filesystem-only body, split out
// so the fallback chain can be exercised against an in-memory filesystem.
func lastInstalledDebianFS(fs afero.Fs, loc *time.Location) (*LastInstalledUpdate, error) {
	// apt's own history is the most precise record: it groups a run into a
	// block and says whether that run installed, upgraded or only removed.
	if t, ok := newestInRotation(fs, aptHistoryPath, func(r io.Reader) (time.Time, bool) {
		return ParseAptHistory(r, loc)
	}); ok {
		return &LastInstalledUpdate{Time: t.UTC(), Source: LastUpdateSourceAptHistory}, nil
	}

	// dpkg logs every package operation, including the ones apt never saw
	// (a bare `dpkg -i`, or a debootstrap that predates any apt run).
	if t, ok := newestInRotation(fs, dpkgLogPath, func(r io.Reader) (time.Time, bool) {
		return ParseDpkgLog(r, loc)
	}); ok {
		return &LastInstalledUpdate{Time: t.UTC(), Source: LastUpdateSourceDpkgLog}, nil
	}

	// Last resort on a host whose logs were stripped, which many slim
	// container images do: dpkg rewrites its status file on every change, so
	// the mtime is the last time the package set moved without saying what
	// moved or whether it was an install at all.
	if fi, err := fs.Stat(dpkgStatusPath); err == nil {
		if mtime := fi.ModTime(); !mtime.IsZero() {
			return &LastInstalledUpdate{Time: mtime.UTC(), Source: LastUpdateSourceDpkgStatus}, nil
		}
	}

	return nil, nil
}

// ParseAptHistory returns the newest Start-Date of an apt run that installed,
// upgraded or reinstalled a package. A block that only removed packages is not
// an update install, and a block carrying an Error: line did not finish, so
// neither counts.
//
// A block looks like:
//
//	Start-Date: 2026-05-09  14:17:19
//	Commandline: apt-get dist-upgrade
//	Upgrade: libpam-modules:arm64 (1.5.3-5ubuntu5, 1.5.3-5ubuntu5.5)
//	End-Date: 2026-05-09  14:17:20
func ParseAptHistory(r io.Reader, loc *time.Location) (time.Time, bool) {
	var newest, start time.Time
	haveStart, changed, failed := false, false, false

	flush := func() {
		if haveStart && changed && !failed && start.After(newest) {
			newest = start
		}
		start = time.Time{}
		haveStart, changed, failed = false, false, false
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), dpkgLogMaxLine)
	for scanner.Scan() {
		key, value, ok := strings.Cut(strings.TrimSpace(scanner.Text()), ":")
		if !ok {
			continue
		}

		switch key {
		case "Start-Date":
			// A block without an End-Date (the run was killed) still opens the
			// next one, so flush on Start-Date as well as on End-Date.
			flush()
			if t, err := parseDpkgTime(value, loc); err == nil {
				start, haveStart = t, true
			}
		case "Install", "Upgrade", "Reinstall":
			if strings.TrimSpace(value) != "" {
				changed = true
			}
		case "Error":
			failed = true
		case "End-Date":
			flush()
		}
	}
	flush()

	return newest, !newest.IsZero()
}

// ParseDpkgLog returns the newest timestamp on a dpkg.log line recording a
// package being installed or upgraded. `status installed` is the line dpkg
// writes once a package is fully configured; the `install` and `upgrade` action
// lines count too, so a run interrupted before configuration still registers.
// Removals, purges, triggers and startup lines are not update installs.
//
//	2026-05-09 14:23:31 upgrade libpam-modules:arm64 1.5.3-5ubuntu5 1.5.3-5ubuntu5.5
//	2026-05-09 14:23:32 status installed unminimize:arm64 0.2.1
func ParseDpkgLog(r io.Reader, loc *time.Location) (time.Time, bool) {
	var newest time.Time

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), dpkgLogMaxLine)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}

		switch fields[2] {
		case "install", "upgrade":
		case "status":
			if fields[3] != "installed" {
				continue
			}
		default:
			continue
		}

		t, err := parseDpkgTime(fields[0]+" "+fields[1], loc)
		if err != nil {
			continue
		}
		if t.After(newest) {
			newest = t
		}
	}

	return newest, !newest.IsZero()
}

// parseDpkgTime reads the timestamp dpkg and apt write, tolerating the
// single-space and double-space separators both spellings use.
func parseDpkgTime(value string, loc *time.Location) (time.Time, error) {
	fields := strings.Fields(value)
	if len(fields) < 2 {
		return time.Time{}, errors.New("not a dpkg timestamp: " + value)
	}
	if loc == nil {
		loc = time.UTC
	}
	return time.ParseInLocation(dpkgLogTimeLayout, fields[0]+" "+fields[1], loc)
}

// newestInRotation walks a log and its logrotate copies newest first, returning
// the first timestamp any of them yields.
func newestInRotation(fs afero.Fs, base string, parse func(io.Reader) (time.Time, bool)) (time.Time, bool) {
	for _, path := range rotationPaths(base) {
		if t, ok := parseRotatedFile(fs, path, parse); ok {
			return t, true
		}
	}
	return time.Time{}, false
}

// rotationPaths lists a log and its logrotate copies, newest first. dpkg keeps
// the first rotation uncompressed while apt gzips every one, so both spellings
// are tried rather than guessing per file.
func rotationPaths(base string) []string {
	paths := make([]string, 0, 1+2*dpkgLogMaxRotations)
	paths = append(paths, base)
	for i := 1; i <= dpkgLogMaxRotations; i++ {
		n := strconv.Itoa(i)
		paths = append(paths, base+"."+n, base+"."+n+".gz")
	}
	return paths
}

func parseRotatedFile(fs afero.Fs, path string, parse func(io.Reader) (time.Time, bool)) (time.Time, bool) {
	f, err := fs.Open(path)
	if err != nil {
		return time.Time{}, false
	}
	defer f.Close()

	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return time.Time{}, false
		}
		defer gz.Close()
		r = gz
	}

	return parse(r)
}
