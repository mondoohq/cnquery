// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import "io"

const (
	// ShareType is an enum; "$($_.ShareType)" forces its label (e.g.
	// FileSystemDirectory) rather than its numeric value.
	SMB_SHARES      = `Get-SmbShare | Select-Object Name,Path,Description,ScopeName,@{Name='ShareType';Expression={"$($_.ShareType)"}} | ConvertTo-Json`
	SMB_SESSIONS    = `Get-SmbSession | Select-Object SessionId,ClientComputerName,ClientUserName,Dialect,NumOpens | ConvertTo-Json`
	SMB_CONNECTIONS = `Get-SmbConnection | Select-Object ServerName,ShareName,UserName,Dialect | ConvertTo-Json`
)

type WindowsSmbShare struct {
	Name        string `json:"Name"`
	Path        string `json:"Path"`
	Description string `json:"Description"`
	ScopeName   string `json:"ScopeName"`
	ShareType   string `json:"ShareType"`
}

type WindowsSmbSession struct {
	// SessionId uniquely identifies a session; a single client+user pair can
	// hold multiple concurrent sessions, so it is required to key them apart.
	SessionId          uint64 `json:"SessionId"`
	ClientComputerName string `json:"ClientComputerName"`
	ClientUserName     string `json:"ClientUserName"`
	Dialect            string `json:"Dialect"`
	NumOpens           int64  `json:"NumOpens"`
}

type WindowsSmbConnection struct {
	ServerName string `json:"ServerName"`
	ShareName  string `json:"ShareName"`
	UserName   string `json:"UserName"`
	Dialect    string `json:"Dialect"`
}

func ParseWindowsSmbShares(input io.Reader) ([]WindowsSmbShare, error) {
	return streamDecodeJSONArray[WindowsSmbShare](input)
}

func ParseWindowsSmbSessions(input io.Reader) ([]WindowsSmbSession, error) {
	return streamDecodeJSONArray[WindowsSmbSession](input)
}

func ParseWindowsSmbConnections(input io.Reader) ([]WindowsSmbConnection, error) {
	return streamDecodeJSONArray[WindowsSmbConnection](input)
}
