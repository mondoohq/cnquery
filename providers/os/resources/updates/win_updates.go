// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package updates

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/powershell"
)

const (
	WindowsUpdateFormat = "wsus"
)

var WINDOWS_QUERY_WSUS_AVAILABLE = `
$ProgressPreference='SilentlyContinue';
$updateSession = new-object -com "Microsoft.Update.Session"
$searcher=$updateSession.CreateupdateSearcher().Search(("IsInstalled=0 and Type='Software'"))
$updates = $searcher.Updates | ForEach-Object {
	$update = $_
	$value = New-Object psobject -Property @{
		"UpdateID" =  $update.Identity.UpdateID;
		"Title" = $update.Title
		"MsrcSeverity" = $update.MsrcSeverity
		"RevisionNumber" =  $update.Identity.RevisionNumber;
		"CategoryIDs" = @($update.Categories | % { $_.CategoryID })
		"SecurityBulletinIDs" = $update.SecurityBulletinIDs
		"RebootRequired" = $update.RebootRequired
		"KBArticleIDs" = $update.KBArticleIDs
		"CveIDs" = @($update.CveIDs)
	}
	$value
}
@($updates) | ConvertTo-Json`

type powershellWinUpdate struct {
	UpdateID       string   `json:"UpdateID"`
	Title          string   `json:"Title"`
	MsrcSeverity   string   `json:"MsrcSeverity"`
	Revision       string   `json:"Revision"`
	RebootRequired bool     `json:"RebootRequired"`
	CategoryIDs    []string `json:"CategoryIDs"`
	KBArticleIDs   []string `json:"KBArticleIDs"`
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
	cmd := powershell.Encode(WINDOWS_QUERY_WSUS_AVAILABLE)
	c, err := um.conn.RunCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}
	return ParseWindowsUpdates(c.Stdout)
}

func ParseWindowsUpdates(input io.Reader) ([]OperatingSystemUpdate, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	// handle case where no packages are installed
	if len(data) == 0 {
		return []OperatingSystemUpdate{}, nil
	}

	var powerShellUpdates []powershellWinUpdate
	err = json.Unmarshal(data, &powerShellUpdates)
	if err != nil {
		return nil, err
	}

	updates := make([]OperatingSystemUpdate, len(powerShellUpdates))
	for i := range powerShellUpdates {
		if len(powerShellUpdates[i].KBArticleIDs) == 0 {
			log.Warn().Str("update", powerShellUpdates[i].UpdateID).Msg("ms update has no kb assigned")
			continue
		}

		// todo: we may want to make that decision server-side, since it does not require us to update the agent
		// therefore we need additional information to be transmitted via the packages eg. labels
		// important := false
		// for ci := range powerShellUpdates[i].CategoryIDs {
		// 	id := powerShellUpdates[i].CategoryIDs[ci]
		// 	classification := wsusClassificationGUID[strings.ToLower(id)]
		// 	if classification == CriticalUpdates || classification == SecurityUpdates || classification == UpdateRollups {
		// 		important = true
		// 	}
		// }

		updates[i] = OperatingSystemUpdate{
			ID:          powerShellUpdates[i].UpdateID,
			Name:        powerShellUpdates[i].KBArticleIDs[0],
			Description: powerShellUpdates[i].Title,
			Version:     powerShellUpdates[i].Revision,
			Severity:    powerShellUpdates[i].MsrcSeverity,
			Format:      "windows/updates",
			Restart:     powerShellUpdates[i].RebootRequired,
		}
	}
	return updates, nil
}
