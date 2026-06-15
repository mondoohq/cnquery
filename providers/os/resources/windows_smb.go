// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"io"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/powershell"
	"go.mondoo.com/mql/v13/providers/os/resources/windows"
)

// smbStdout runs a PowerShell command and returns its stdout reader, failing
// on a non-zero exit status.
func smbStdout(conn shared.Connection, command string) (io.Reader, error) {
	executedCmd, err := conn.RunCommand(powershell.Encode(command))
	if err != nil {
		return nil, err
	}
	if executedCmd.ExitStatus != 0 {
		stderr, err := io.ReadAll(executedCmd.Stderr)
		if err != nil {
			return nil, err
		}
		return nil, errors.New("failed to query SMB information: " + string(stderr))
	}
	return executedCmd.Stdout, nil
}

func (w *mqlWindowsSmb) shares() ([]any, error) {
	conn := w.MqlRuntime.Connection.(shared.Connection)
	stdout, err := smbStdout(conn, windows.SMB_SHARES)
	if err != nil {
		return nil, err
	}
	shares, err := windows.ParseWindowsSmbShares(stdout)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(shares))
	for _, s := range shares {
		r, err := CreateResource(w.MqlRuntime, "windows.smb.share", map[string]*llx.RawData{
			"__id":        llx.StringData("windows.smb.share/" + s.ScopeName + "/" + s.Name),
			"name":        llx.StringData(s.Name),
			"path":        llx.StringData(s.Path),
			"description": llx.StringData(s.Description),
			"scopeName":   llx.StringData(s.ScopeName),
			"shareType":   llx.StringData(s.ShareType),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (w *mqlWindowsSmb) sessions() ([]any, error) {
	conn := w.MqlRuntime.Connection.(shared.Connection)
	stdout, err := smbStdout(conn, windows.SMB_SESSIONS)
	if err != nil {
		return nil, err
	}
	sessions, err := windows.ParseWindowsSmbSessions(stdout)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(sessions))
	for _, s := range sessions {
		r, err := CreateResource(w.MqlRuntime, "windows.smb.session", map[string]*llx.RawData{
			// SessionId keeps concurrent sessions from the same client+user distinct.
			"__id":               llx.StringData(fmt.Sprintf("windows.smb.session/%s/%s/%d", s.ClientComputerName, s.ClientUserName, s.SessionId)),
			"clientComputerName": llx.StringData(s.ClientComputerName),
			"clientUserName":     llx.StringData(s.ClientUserName),
			"dialect":            llx.StringData(s.Dialect),
			"numOpens":           llx.IntData(s.NumOpens),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (w *mqlWindowsSmb) connections() ([]any, error) {
	conn := w.MqlRuntime.Connection.(shared.Connection)
	stdout, err := smbStdout(conn, windows.SMB_CONNECTIONS)
	if err != nil {
		return nil, err
	}
	connections, err := windows.ParseWindowsSmbConnections(stdout)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(connections))
	for _, c := range connections {
		r, err := CreateResource(w.MqlRuntime, "windows.smb.connection", map[string]*llx.RawData{
			// Get-SmbConnection exposes no unique per-connection id, so key on the
			// stable connection attributes (server + share + user + dialect)
			// rather than the list index — identity then survives output-order
			// changes between refreshes. These four fields together identify a
			// distinct client->server SMB connection in practice.
			"__id":       llx.StringData(fmt.Sprintf("windows.smb.connection/%s/%s/%s/%s", c.ServerName, c.ShareName, c.UserName, c.Dialect)),
			"serverName": llx.StringData(c.ServerName),
			"shareName":  llx.StringData(c.ShareName),
			"userName":   llx.StringData(c.UserName),
			"dialect":    llx.StringData(c.Dialect),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}
