// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"io"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

// safeBinaryName matches a plain executable name — letters, digits, and the
// punctuation real tool binaries use. A binary name never contains a slash,
// whitespace, or shell metacharacters, so rejecting those keeps the `command -v`
// call safe even if a caller ever passes untrusted input. This is defense in
// depth: shellQuote already single-quotes the value, and POSIX shells perform
// no expansion inside single quotes.
var safeBinaryName = regexp.MustCompile(`^[A-Za-z0-9._+-]+$`)

// PkgFileOwnershipResolver is an optional capability implemented by package
// managers that can reverse-resolve a file path to the installed package that
// owns it (e.g. `pacman -Qo`, `dpkg -S`, `rpm -qf`, `apk info --who-owns`).
// It is the inverse of Files(): Files() maps a package to its paths, this maps
// a path back to its package. Managers that cannot answer ownership queries
// simply do not implement it, and FindPackageOwningFile skips them.
type PkgFileOwnershipResolver interface {
	// FindFileOwner returns the name of the installed package that owns the
	// file at the given absolute path, spelled exactly as List() reports it so
	// callers can resolve it against the enumerated packages. It returns
	// ("", nil) when no package owns the path — that is an ordinary outcome,
	// not an error. An error is reserved for unexpected failures.
	FindFileOwner(path string) (string, error)
}

// FindPackageOwningFile asks each resolved package manager that supports
// ownership lookup which installed package owns absPath, returning the first
// owner found. It returns "" when no manager can attribute the path (unowned,
// or no manager implements the capability on this platform).
func FindPackageOwningFile(conn shared.Connection, absPath string) (string, error) {
	if absPath == "" {
		return "", nil
	}
	// Every implemented resolver shells out to the manager CLI; without command
	// execution there is nothing to ask.
	if !conn.Capabilities().Has(shared.Capability_RunCommand) {
		return "", nil
	}

	pms, err := ResolveSystemPkgManagers(conn)
	if err != nil {
		return "", err
	}
	return findFileOwner(pms, absPath), nil
}

// findFileOwner asks each manager that supports ownership lookup which package
// owns absPath, returning the first owner found. Per-manager errors are logged
// and skipped (an unowned path is an ordinary outcome, not a failure).
func findFileOwner(pms []OperatingSystemPkgManager, absPath string) string {
	for _, pm := range pms {
		resolver, ok := pm.(PkgFileOwnershipResolver)
		if !ok {
			continue
		}
		name, err := resolver.FindFileOwner(absPath)
		if err != nil {
			log.Debug().Err(err).Str("pkgManager", pm.Name()).Str("path", absPath).
				Msg("mql[packages]> file ownership lookup failed")
			continue
		}
		if name != "" {
			return name
		}
	}
	return ""
}

// FindPackageOwningBinary resolves binaryName on the target's PATH (via
// `command -v`) and returns the installed package that owns the resolved
// binary. It queries both the PATH entry and its symlink target, since native
// installers often symlink a launcher into ~/.local/bin while the package
// database records the canonical file. Returns "" when the binary is not on
// PATH or no package owns it.
func FindPackageOwningBinary(conn shared.Connection, binaryName string) (string, error) {
	if binaryName == "" || !safeBinaryName.MatchString(binaryName) {
		return "", nil
	}
	if !conn.Capabilities().Has(shared.Capability_RunCommand) {
		return "", nil
	}

	paths := binaryCandidatePaths(conn, binaryName)
	if len(paths) == 0 {
		return "", nil
	}

	// Resolve the platform's package managers once and reuse them across every
	// candidate path — platform detection + manager instantiation is not free.
	pms, err := ResolveSystemPkgManagers(conn)
	if err != nil {
		return "", err
	}
	for _, path := range paths {
		if name := findFileOwner(pms, path); name != "" {
			return name, nil
		}
	}
	return "", nil
}

// binaryCandidatePaths returns the PATH location of binaryName and its resolved
// symlink target (deduplicated, in that order). Empty when the binary is not on
// PATH.
func binaryCandidatePaths(conn shared.Connection, binaryName string) []string {
	cmd, err := conn.RunCommand("command -v " + shellQuote(binaryName))
	if err != nil || cmd.ExitStatus != 0 {
		return nil
	}
	path := strings.TrimSpace(readCommandOutput(cmd.Stdout))
	if path == "" || !strings.HasPrefix(path, "/") {
		// command -v prints shell builtins/aliases without a leading slash;
		// only real filesystem paths can be owned by a package.
		return nil
	}

	paths := []string{path}
	if resolved, err := conn.RunCommand("readlink -f " + shellQuote(path)); err == nil && resolved.ExitStatus == 0 {
		if r := strings.TrimSpace(readCommandOutput(resolved.Stdout)); r != "" && r != path {
			paths = append(paths, r)
		}
	}
	return paths
}

// shellQuote single-quotes s for safe interpolation into a shell command,
// escaping any embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// readCommandOutput drains a command's stdout reader into a string.
func readCommandOutput(r io.Reader) string {
	if r == nil {
		return ""
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return ""
	}
	return string(b)
}

// firstLine returns the first line of s (without the trailing newline).
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
