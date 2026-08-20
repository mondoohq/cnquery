// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"io"
	"strings"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/providers/os/connection/shared"
)

// parseFipsEnabled interprets the content of /proc/sys/crypto/fips_enabled.
// The file contains "1" when FIPS mode is active and "0" otherwise.
func parseFipsEnabled(content string) bool {
	return strings.TrimSpace(content) == "1"
}

// normalizeCryptoPolicy trims the output of `update-crypto-policies --show`
// and returns the first line (e.g. "DEFAULT", "FUTURE", "FIPS", "FIPS:OSPP").
func normalizeCryptoPolicy(stdout string) string {
	trimmed := strings.TrimSpace(stdout)
	if trimmed == "" {
		return ""
	}
	if idx := strings.IndexByte(trimmed, '\n'); idx >= 0 {
		return strings.TrimSpace(trimmed[:idx])
	}
	return trimmed
}

// fipsEnabled reports whether the OS is running in FIPS mode.
//
// On Linux this reads /proc/sys/crypto/fips_enabled through the connection
// filesystem. When the file is absent or unreadable (non-Linux, containers
// without procfs), it degrades gracefully to false rather than failing.
func (p *mqlOs) fipsEnabled() (bool, error) {
	conn, ok := p.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		return false, nil
	}

	afs := &afero.Afero{Fs: conn.FileSystem()}
	content, err := afs.ReadFile("/proc/sys/crypto/fips_enabled")
	if err != nil {
		// file not present / not readable (non-Linux, container without procfs)
		return false, nil
	}

	return parseFipsEnabled(string(content)), nil
}

// cryptoPolicy returns the active system-wide crypto policy as reported by
// `update-crypto-policies --show` (RHEL/Fedora). On platforms without that
// tool, or when the command is missing/errors/exits non-zero, it returns an
// empty string gracefully.
func (p *mqlOs) cryptoPolicy() (string, error) {
	conn, ok := p.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		return "", nil
	}

	// systems without run-command capability (e.g. static images) can't run it
	if !conn.Capabilities().Has(shared.Capability_RunCommand) {
		return "", nil
	}

	cmd, err := conn.RunCommand("update-crypto-policies --show")
	if err != nil {
		return "", nil
	}
	if cmd.ExitStatus != 0 {
		return "", nil
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return "", nil
	}

	return normalizeCryptoPolicy(string(data)), nil
}
