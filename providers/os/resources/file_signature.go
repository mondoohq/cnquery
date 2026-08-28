// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/os/connection/shared"
	"go.mondoo.com/mql/providers/os/resources/powershell"
)

// file.signature is both a resource name and the field path that reaches it.
// The compiler resolves the longest matching resource name first, so a bare
// `file.signature` builds the sub-resource instead of calling the parent's
// accessor, and without an Init every field reads null: a check asserting
// `signed` would quietly pass on an unsigned binary. There is no file to
// inspect in that form, so say so instead of handing back an empty instance.
func initFileSignature(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if _, ok := args["__id"]; ok {
		return args, nil, nil
	}
	return nil, nil, errors.New("file.signature needs a file, query it as file(\"/path/to/file\").signature")
}

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
			ts = parseCodesignTimestamp(strings.TrimPrefix(line, "Timestamp="))
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

// parseCodesignTimestamp parses the timestamp codesign prints in its -dvvv
// output, e.g. "Apr 1, 2026 at 6:38:28 AM".
//
// codesign renders the date through ICU, which on current macOS separates the
// time from AM/PM with U+202F (narrow no-break space) rather than a plain
// space, so the layout below only matches after the Unicode spaces are folded
// to ASCII. The format is also locale-dependent: on a non-English macOS the
// month name will not match and the timestamp stays null, which leaves the
// signature's signed/verified/authority fields unaffected.
func parseCodesignTimestamp(v string) *time.Time {
	v = strings.TrimSpace(unicodeSpaces.Replace(v))
	if t, err := time.Parse("Jan 2, 2006 at 3:04:05 PM", v); err == nil {
		return &t
	}
	return nil
}

// unicodeSpaces folds the non-ASCII spaces ICU emits to a plain space.
var unicodeSpaces = strings.NewReplacer(
	"\u202f", " ", // narrow no-break space
	"\u00a0", " ", // no-break space
	"\u2009", " ", // thin space
)

// unixSingleQuote wraps a path in single quotes for a POSIX shell, escaping any
// embedded single quotes.
func unixSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// psSingleQuote escapes a path for a PowerShell single-quoted literal.
func psSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
