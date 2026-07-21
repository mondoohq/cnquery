// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"

	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/os/connection/shared"
	"go.mondoo.com/mql/providers/os/resources/powershell"
)

// signature returns the file's code signature. macOS uses codesign, Windows uses
// Authenticode; on every other platform (no universal per-file ELF signing model)
// it resolves to null.
func (s *mqlFile) signature(path string) (*mqlFileSignature, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)
	pf := conn.Asset().Platform
	switch {
	case pf != nil && pf.IsFamily("darwin"):
		return s.codesignSignature(conn, path)
	case pf != nil && pf.IsFamily("windows"):
		return s.authenticodeSignature(conn, path)
	default:
		s.Signature = plugin.TValue[*mqlFileSignature]{State: plugin.StateIsSet | plugin.StateIsNull}
		return nil, nil
	}
}

func (s *mqlFile) newSignature(path string, signed, verified bool, authority, teamID, issuer string, ts *time.Time, format string) (*mqlFileSignature, error) {
	res, err := CreateResource(s.MqlRuntime, "file.signature", map[string]*llx.RawData{
		// scope the id to the file path: many files share the same authority
		// (e.g. every Apple binary is signed by "Software Signing"), so an
		// id built from the signature fields alone would collide across files.
		"__id":      llx.StringData("file.signature/" + path),
		"signed":    llx.BoolData(signed),
		"verified":  llx.BoolData(verified),
		"authority": llx.StringData(authority),
		"teamId":    llx.StringData(teamID),
		"issuer":    llx.StringData(issuer),
		"timestamp": llx.TimeDataPtr(ts),
		"format":    llx.StringData(format),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlFileSignature), nil
}

func (s *mqlFile) nullSignature() (*mqlFileSignature, error) {
	s.Signature = plugin.TValue[*mqlFileSignature]{State: plugin.StateIsSet | plugin.StateIsNull}
	return nil, nil
}

// --- macOS (codesign) ---

func (s *mqlFile) codesignSignature(conn shared.Connection, path string) (*mqlFileSignature, error) {
	// exit 0 => the cryptographic seal verifies
	verifyCmd, err := conn.RunCommand("codesign --verify --deep --strict " + unixSingleQuote(path))
	if err != nil {
		return nil, err
	}
	verified := verifyCmd.ExitStatus == 0

	// -dvvv writes the signing metadata to stderr
	dispCmd, err := conn.RunCommand("codesign -dvvv " + unixSingleQuote(path))
	if err != nil {
		return nil, err
	}
	meta := readAll(dispCmd.Stderr)

	// nothing to report for a path codesign cannot read at all (missing file, etc.)
	if strings.Contains(meta, "No such file") {
		return s.nullSignature()
	}

	signed := !strings.Contains(meta, "code object is not signed at all")
	var authority, teamID string
	var ts *time.Time
	for _, raw := range strings.Split(meta, "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case authority == "" && strings.HasPrefix(line, "Authority="):
			authority = strings.TrimPrefix(line, "Authority=")
		case strings.HasPrefix(line, "TeamIdentifier="):
			if v := strings.TrimPrefix(line, "TeamIdentifier="); v != "not set" {
				teamID = v
			}
		case ts == nil && strings.HasPrefix(line, "Timestamp="):
			// codesign's -dvvv timestamp is locale-formatted, so this parse is
			// best-effort: on a non-English macOS it may not match and ts stays
			// nil (the signature's signed/verified/authority fields are
			// unaffected).
			v := strings.TrimPrefix(line, "Timestamp=")
			if t, e := time.Parse("Jan 2, 2006 at 3:04:05 PM", v); e == nil {
				ts = &t
			}
		}
	}
	return s.newSignature(path, signed, verified, authority, teamID, "", ts, "codesign")
}

// --- Windows (Authenticode) ---

const authenticodeScript = `$s = Get-AuthenticodeSignature -LiteralPath '%s'
[pscustomobject]@{
  Status  = $s.Status.ToString()
  Subject = $s.SignerCertificate.Subject
  Issuer  = $s.SignerCertificate.Issuer
} | ConvertTo-Json -Compress`

func (s *mqlFile) authenticodeSignature(conn shared.Connection, path string) (*mqlFileSignature, error) {
	script := strings.Replace(authenticodeScript, "%s", psSingleQuote(path), 1)
	cmd, err := conn.RunCommand(powershell.Encode(script))
	if err != nil {
		return nil, err
	}
	out := bytes.TrimSpace([]byte(readAll(cmd.Stdout)))
	if len(out) == 0 {
		return s.nullSignature()
	}

	var r struct {
		Status  string `json:"Status"`
		Subject string `json:"Subject"`
		Issuer  string `json:"Issuer"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		return s.nullSignature()
	}

	signed := r.Status != "" && r.Status != "NotSigned"
	verified := r.Status == "Valid"
	return s.newSignature(path, signed, verified, r.Subject, "", r.Issuer, nil, "authenticode")
}

// --- helpers ---

func readAll(r io.Reader) string {
	if r == nil {
		return ""
	}
	b, _ := io.ReadAll(r)
	return string(b)
}

// unixSingleQuote wraps a path in single quotes for a POSIX shell, escaping any
// embedded single quotes.
func unixSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// psSingleQuote escapes a path for a PowerShell single-quoted literal.
func psSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
