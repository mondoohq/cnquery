// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package updates

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/providers/os/connection/shared"
	"go.mondoo.com/mql/providers/os/resources/powershell"
)

const (
	WindowsUpdateFormat = "wsus"

	// WindowsUpdateCriteriaSoftware selects installable software updates
	// (drivers excluded). This is what os.update reports.
	WindowsUpdateCriteriaSoftware = "IsInstalled=0 and Type='Software'"
	// WindowsUpdateCriteriaAvailable selects every installable, non-hidden
	// update (drivers included). This is what windows.update.available reports.
	WindowsUpdateCriteriaAvailable = "IsInstalled=0 and IsHidden=0"
)

// windowsUpdateSearchQuery builds a PowerShell snippet that searches the
// Windows Update Agent with the given criteria and emits one rich JSON record
// per update. It is the single source of the WUA "search" used by both
// os.update (via WindowsUpdateManager) and windows.update.available.
//
// IMPORTANT: criteria is concatenated into the script verbatim, so it must be
// a trusted constant (e.g. WindowsUpdateCriteria*), never user input.
func windowsUpdateSearchQuery(criteria string) string {
	// $ErrorActionPreference is Stop so a failed search is a terminating error
	// and powershell.exe exits non-zero. Without it the COM failure is a
	// non-terminating error: the exit status stays 0, $searcher stays null,
	// and $searcher.Updates piped to ForEach-Object still runs the block once
	// with a null input, so the caller is handed one empty update and reads it
	// as a host with nothing outstanding. A host whose update agent could not
	// be reached must fail the check, not report itself fully patched.
	return `
$ErrorActionPreference='Stop';
$ProgressPreference='SilentlyContinue';
$updateSession = new-object -com "Microsoft.Update.Session"
$searcher = $updateSession.CreateupdateSearcher().Search("` + criteria + `")
$updates = $searcher.Updates | ForEach-Object {
	$update = $_
	New-Object psobject -Property @{
		"UpdateID" = $update.Identity.UpdateID;
		"Title" = $update.Title
		"MsrcSeverity" = $update.MsrcSeverity
		"SupportUrl" = $update.SupportUrl
		"RebootRequired" = [bool]$update.RebootRequired
		"KBArticleIDs" = @($update.KBArticleIDs)
		"CveIDs" = @($update.CveIDs)
		"Categories" = @($update.Categories | ForEach-Object { $_.Name })
	}
}
@($updates) | ConvertTo-Json -Depth 3`
}

// WindowsUpdate is the rich representation of an update returned by a Windows
// Update Agent search. It carries everything both consumers need; each maps it
// to its own output type.
type WindowsUpdate struct {
	UpdateID       string   `json:"UpdateID"`
	Title          string   `json:"Title"`
	MsrcSeverity   string   `json:"MsrcSeverity"`
	SupportUrl     string   `json:"SupportUrl"`
	RebootRequired bool     `json:"RebootRequired"`
	KBArticleIDs   []string `json:"KBArticleIDs"`
	CveIDs         []string `json:"CveIDs"`
	Categories     []string `json:"Categories"`
}

type WindowsUpdateManager struct {
	conn shared.Connection
}

func (um *WindowsUpdateManager) Name() string {
	return "Windows Server Update Services Manager"
}

func (um *WindowsUpdateManager) Format() string {
	return WindowsUpdateFormat
}

func (um *WindowsUpdateManager) List() ([]OperatingSystemUpdate, error) {
	updates, err := SearchWindowsUpdates(um.conn, WindowsUpdateCriteriaSoftware)
	if err != nil {
		return nil, err
	}

	res := make([]OperatingSystemUpdate, 0, len(updates))
	for i := range updates {
		osUpdate, ok := updates[i].toOperatingSystemUpdate()
		if !ok {
			log.Warn().Str("update", updates[i].UpdateID).Msg("ms update has no kb assigned")
			continue
		}
		res = append(res, osUpdate)
	}
	return res, nil
}

// toOperatingSystemUpdate maps a WindowsUpdate to the cross-platform
// OperatingSystemUpdate shape used by os.update. Updates without a KB article
// (ok == false) are skipped, since the KB is the os.update identity.
func (u WindowsUpdate) toOperatingSystemUpdate() (OperatingSystemUpdate, bool) {
	if len(u.KBArticleIDs) == 0 {
		return OperatingSystemUpdate{}, false
	}
	return OperatingSystemUpdate{
		ID:          u.UpdateID,
		Name:        u.KBArticleIDs[0],
		Description: u.Title,
		Severity:    u.MsrcSeverity,
		Format:      "windows/updates",
		Restart:     u.RebootRequired,
	}, true
}

// SearchWindowsUpdates runs a Windows Update Agent search with the given
// criteria and returns the parsed updates.
func SearchWindowsUpdates(conn shared.Connection, criteria string) ([]WindowsUpdate, error) {
	cmd := powershell.Encode(windowsUpdateSearchQuery(criteria))
	c, err := conn.RunCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("could not search for windows updates: %w", err)
	}
	if c.ExitStatus != 0 {
		stderr, err := io.ReadAll(c.Stderr)
		if err != nil {
			return nil, err
		}
		return nil, errors.New("failed to retrieve updates: " + string(stderr))
	}
	return ParseWindowsUpdates(c.Stdout)
}

func ParseWindowsUpdates(input io.Reader) ([]WindowsUpdate, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	// handle case where no updates are available
	if len(strings.TrimSpace(string(data))) == 0 {
		return []WindowsUpdate{}, nil
	}

	// ConvertTo-Json emits a bare object (not a single-element array) when the
	// search returns exactly one update.
	var updates []WindowsUpdate
	arrErr := json.Unmarshal(data, &updates)
	if arrErr == nil {
		return dropEmptyWindowsUpdates(updates), nil
	}

	var single WindowsUpdate
	if err := json.Unmarshal(data, &single); err != nil {
		return nil, arrErr
	}
	return dropEmptyWindowsUpdates([]WindowsUpdate{single}), nil
}

// dropEmptyWindowsUpdates removes records that carry no identity at all.
//
// A record with neither an update ID nor a title is not an update; it is what
// a null decodes to. The script's ErrorActionPreference keeps a failed search
// from producing one, and this keeps any other source of a blank record from
// being counted as an outstanding update.
func dropEmptyWindowsUpdates(updates []WindowsUpdate) []WindowsUpdate {
	res := make([]WindowsUpdate, 0, len(updates))
	for i := range updates {
		if updates[i].UpdateID == "" && updates[i].Title == "" {
			continue
		}
		res = append(res, updates[i])
	}
	return res
}
