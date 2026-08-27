// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package updates

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/providers/os/connection/shared"
	"go.mondoo.com/mql/providers/os/resources/logrotate"
)

const (
	aptHistoryPath = "/var/log/apt/history.log"

	// aptHistoryTimeLayout is the timestamp apt writes: local time with no
	// zone. apt separates the date from the time with two spaces, so the parser
	// rejoins the fields before matching this layout.
	aptHistoryTimeLayout = "2006-01-02 15:04:05"

	// aptHistoryMaxLine caps a single line. An Upgrade: line names every
	// package in the run on one line, so on a dist-upgrade of a full desktop it
	// runs well past the scanner's default 64 KB. Past the limit the scan stops
	// and the rest of the log is lost, which would silently report an older
	// update as the newest one.
	aptHistoryMaxLine = 4 * 1024 * 1024
)

// aptUpgradeVerbs are the subcommands that upgrade whatever the configured
// repositories offer. A run driven by one of them is an operating system patch
// run: it is not aimed at a package the operator named.
var aptUpgradeVerbs = map[string]struct{}{
	"upgrade":      {},
	"dist-upgrade": {},
	"full-upgrade": {},
	"safe-upgrade": {}, // aptitude's spelling
}

// aptTargetedVerbs are the subcommands aimed at packages the operator named.
// `apt install curl` moves installed software, and dpkg records it exactly the
// way it records a security upgrade, but it says nothing about patch state.
// Their presence disqualifies a run outright, even alongside an upgrade verb.
var aptTargetedVerbs = map[string]struct{}{
	"install":          {},
	"reinstall":        {},
	"remove":           {},
	"purge":            {},
	"autoremove":       {},
	"autopurge":        {},
	"build-dep":        {},
	"download":         {},
	"source":           {},
	"deselect-upgrade": {},
}

// lastInstalledDebian reads the newest operating system patch run apt recorded.
// Every source is a file read, so this answers the same way on a running host
// and on a mounted filesystem.
func lastInstalledDebian(conn shared.Connection) (*LastInstalledUpdate, error) {
	return lastInstalledDebianFS(conn.FileSystem(), assetTimeZone(conn))
}

// lastInstalledDebianFS is lastInstalledDebian's filesystem-only body, split out
// so the rotation walk can be exercised against an in-memory filesystem.
//
// Only apt's own history answers here. dpkg.log records the same package
// operations but carries neither the invoking command nor a repository, so it
// cannot tell an operator's `apt install` from a security upgrade; under a
// field that means "operating system patch" it would answer a different
// question, so a host whose apt history has rotated away reads null.
func lastInstalledDebianFS(fs afero.Fs, loc *time.Location) (*LastInstalledUpdate, error) {
	for _, path := range logrotate.Paths(aptHistoryPath, logrotate.DefaultMaxRotations) {
		f, err := logrotate.Open(fs, path)
		if err != nil {
			continue
		}
		t, source, ok := ParseAptHistory(f, loc)
		f.Close()
		if !ok {
			continue
		}
		return &LastInstalledUpdate{Time: t.UTC(), Source: source}, nil
	}
	return nil, nil
}

// ParseAptHistory returns the Start-Date of the newest apt run that patched the
// operating system, and the source constant naming what kind of run it was.
//
// A run qualifies when all of these hold:
//
//   - it installed, upgraded or reinstalled something (a removal-only run is
//     not a patch),
//   - it carries no Error: line (a run that did not finish did not patch),
//   - and its Commandline: is an upgrade of the configured repositories rather
//     than an operation on packages the operator named.
//
// The newest qualifying run wins regardless of its kind. An unattended-upgrades
// run from last month does not outrank an operator's dist-upgrade from
// yesterday; the source reports which one answered.
//
// A block looks like:
//
//	Start-Date: 2026-05-09  14:17:19
//	Commandline: apt-get dist-upgrade
//	Upgrade: libpam-modules:arm64 (1.5.3-5ubuntu5, 1.5.3-5ubuntu5.5)
//	End-Date: 2026-05-09  14:17:20
func ParseAptHistory(r io.Reader, loc *time.Location) (time.Time, string, bool) {
	var newest, start time.Time
	var newestSource, source string
	haveStart, changed, failed := false, false, false

	flush := func() {
		if haveStart && changed && !failed && source != "" && start.After(newest) {
			newest, newestSource = start, source
		}
		start = time.Time{}
		haveStart, changed, failed = false, false, false
		source = ""
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), aptHistoryMaxLine)
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
			if t, err := parseAptTime(value, loc); err == nil {
				start, haveStart = t, true
			}
		case "Commandline":
			source = classifyAptCommandline(value)
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

	return newest, newestSource, !newest.IsZero()
}

// classifyAptCommandline reports which source constant an apt run's command
// line earns, or "" when the run is not an operating system patch.
//
// It matches whole tokens rather than searching the string, so an operator
// installing a package whose name contains an apt subcommand (`apt-get install
// upgrade-helper`) is still read as the targeted install it is. Scanning every
// token instead of locating "the verb" keeps option values from being mistaken
// for one: `apt-get -o Dpkg::Options::=--force-confdef upgrade` carries a
// non-flag token before the subcommand.
//
// A block with no Commandline: at all cannot be attributed and earns "". Some
// apt front ends write none, which costs their runs; that is the conservative
// direction, because the alternative is counting an install as a patch.
func classifyAptCommandline(cmdline string) string {
	if strings.TrimSpace(cmdline) == "" {
		return ""
	}

	// unattended-upgrades is checked first because it names no subcommand at
	// all: its command line is the path to the binary.
	if strings.Contains(cmdline, "unattended-upgrade") {
		return LastUpdateSourceAptSecurity
	}

	upgrade := false
	for _, token := range strings.Fields(cmdline) {
		if _, ok := aptTargetedVerbs[token]; ok {
			return ""
		}
		if _, ok := aptUpgradeVerbs[token]; ok {
			upgrade = true
		}
	}
	if upgrade {
		return LastUpdateSourceAptHistory
	}
	return ""
}

// parseAptTime reads the timestamp apt writes, tolerating the single-space and
// double-space separators both spellings use.
func parseAptTime(value string, loc *time.Location) (time.Time, error) {
	fields := strings.Fields(value)
	if len(fields) < 2 {
		return time.Time{}, errors.New("not an apt timestamp: " + value)
	}
	if loc == nil {
		loc = time.UTC
	}
	return time.ParseInLocation(aptHistoryTimeLayout, fields[0]+" "+fields[1], loc)
}
