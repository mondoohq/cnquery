// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"io"
	"strings"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/os/connection/shared"
)

// Both fields on this file report a security posture, so the failure mode that
// matters is not an error, it is a confident answer nobody read. A host whose
// procfs was never populated is not a host with FIPS switched off, and a host
// with no update-crypto-policies is not a host whose crypto policy is the empty
// string. Reported as false and "", those two are indistinguishable from a real
// reading, and an audit over them passes or fails on data that does not exist.
//
// So every path that did not actually read a value reports null instead.

// fipsSysctl is the Linux sysctl that reports whether the kernel is running in
// FIPS mode. It holds "1" when FIPS mode is active and "0" when it is not.
const fipsSysctl = "/proc/sys/crypto/fips_enabled"

// cryptoPoliciesCmd shows the active system-wide crypto policy on RHEL, Fedora,
// and their derivatives.
const cryptoPoliciesCmd = "update-crypto-policies --show"

// parseFipsEnabled interprets the content of fipsSysctl.
//
// ok is false for anything that is not one of the two documented values,
// including an empty file. A file that says something else has not told us
// whether FIPS is on, and guessing "off" from it would be the same invention
// as guessing it from a file that was not there at all.
func parseFipsEnabled(content string) (enabled bool, ok bool) {
	switch strings.TrimSpace(content) {
	case "1":
		return true, true
	case "0":
		return false, true
	default:
		return false, false
	}
}

// normalizeCryptoPolicy trims the output of cryptoPoliciesCmd and returns the
// first line, e.g. "DEFAULT", "FUTURE", "FIPS", "FIPS:OSPP". It returns an empty
// string when there was nothing to read, which callers report as null rather
// than as a policy named "".
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

// fipsEnabled reports whether the OS is running in FIPS mode, by reading
// fipsSysctl through the connection filesystem.
//
// Null whenever that read did not produce one of the two documented values. The
// file is absent on a kernel built without the FIPS sysctl, and equally absent
// on Windows, on macOS, and in an image whose /proc was never populated. Those
// are not the same fact as each other, and none of them is "FIPS is off".
func (p *mqlOs) fipsEnabled() (bool, error) {
	unknown := func() (bool, error) {
		p.FipsEnabled.State = plugin.StateIsSet | plugin.StateIsNull
		return false, nil
	}

	conn, ok := p.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		return unknown()
	}

	afs := &afero.Afero{Fs: conn.FileSystem()}
	content, err := afs.ReadFile(fipsSysctl)
	if err != nil {
		return unknown()
	}

	enabled, ok := parseFipsEnabled(string(content))
	if !ok {
		return unknown()
	}
	return enabled, nil
}

// cryptoPolicy returns the active system-wide crypto policy as reported by
// cryptoPoliciesCmd.
//
// Null when the policy could not be read: no run-command capability (a static
// image), no such tool (Debian, Ubuntu, macOS, Windows), a non-zero exit, or no
// output. An empty string would name a policy, and a check written against a
// policy name cannot tell that apart from a system that was never asked.
func (p *mqlOs) cryptoPolicy() (string, error) {
	unknown := func() (string, error) {
		p.CryptoPolicy.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}

	conn, ok := p.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		return unknown()
	}

	if !conn.Capabilities().Has(shared.Capability_RunCommand) {
		return unknown()
	}

	cmd, err := conn.RunCommand(cryptoPoliciesCmd)
	if err != nil || cmd == nil {
		return unknown()
	}
	if cmd.ExitStatus != 0 || cmd.Stdout == nil {
		return unknown()
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return unknown()
	}

	policy := normalizeCryptoPolicy(string(data))
	if policy == "" {
		return unknown()
	}
	return policy, nil
}
