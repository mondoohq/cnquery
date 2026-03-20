// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bufio"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

// Known paths for /etc/default/grub
var grubDefaultsPaths = []string{
	"/etc/default/grub",
}

// Known paths for grub.cfg across different distros and boot modes
var grubCfgPaths = []string{
	"/boot/grub2/grub.cfg",               // RHEL/CentOS/Fedora (BIOS)
	"/boot/grub/grub.cfg",                // Debian/Ubuntu (BIOS)
	"/boot/efi/EFI/centos/grub.cfg",      // CentOS (EFI)
	"/boot/efi/EFI/redhat/grub.cfg",      // RHEL (EFI)
	"/boot/efi/EFI/fedora/grub.cfg",      // Fedora (EFI)
	"/boot/efi/EFI/debian/grub.cfg",      // Debian (EFI)
	"/boot/efi/EFI/ubuntu/grub.cfg",      // Ubuntu (EFI)
	"/boot/efi/EFI/sles/grub.cfg",        // SUSE (EFI)
	"/boot/efi/EFI/opensuse/grub.cfg",    // openSUSE (EFI)
	"/boot/efi/EFI/amazon/grub.cfg",      // Amazon Linux (EFI)
	"/boot/efi/EFI/rocky/grub.cfg",       // Rocky Linux (EFI)
	"/boot/efi/EFI/almalinux/grub.cfg",   // AlmaLinux (EFI)
	"/boot/efi/EFI/arch/grub/grub.cfg",   // Arch Linux (EFI)
	"/boot/efi/EFI/BOOT/grub.cfg",        // Generic EFI fallback
	"/boot/efi/EFI/oracle/grub.cfg",      // Oracle Linux (EFI)
	"/boot/efi/EFI/scientific/grub.cfg",  // Scientific Linux (EFI)
	"/boot/efi/EFI/virtuozzo/grub.cfg",   // Virtuozzo (EFI)
	"/boot/efi/EFI/photon/grub.cfg",      // VMware Photon OS (EFI)
	"/boot/efi/EFI/mariner/grub.cfg",     // Azure Linux / Mariner (EFI)
	"/boot/efi/EFI/CBL-Mariner/grub.cfg", // CBL-Mariner (EFI)
	"/boot/efi/EFI/azurelinux/grub.cfg",  // Azure Linux (EFI)
	"/boot/efi/EFI/Microsoft/grub.cfg",   // WSL/Hyper-V (EFI)
}

type mqlGrubConfigInternal struct{}

func initGrubConfig(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn, ok := runtime.Connection.(shared.Connection)
	if !ok {
		return nil, nil, errors.New("wrong connection type")
	}
	fs := conn.FileSystem()
	if fs == nil {
		return nil, nil, errors.New("filesystem not available")
	}

	// Resolve defaultsPath
	if x, ok := args["defaultsPath"]; ok {
		path, ok := x.Value.(string)
		if !ok || path == "" {
			args["defaultsPath"] = llx.StringData(findExistingPath(fs, grubDefaultsPaths))
		}
	} else {
		args["defaultsPath"] = llx.StringData(findExistingPath(fs, grubDefaultsPaths))
	}

	// Resolve grubPath
	if x, ok := args["grubPath"]; ok {
		path, ok := x.Value.(string)
		if !ok || path == "" {
			args["grubPath"] = llx.StringData(findExistingPath(fs, grubCfgPaths))
		}
	} else {
		args["grubPath"] = llx.StringData(findExistingPath(fs, grubCfgPaths))
	}

	return args, nil, nil
}

// findExistingPath returns the first path from candidates that exists on the filesystem,
// or an empty string if none exist.
func findExistingPath(fs afero.Fs, candidates []string) string {
	for _, path := range candidates {
		f, err := fs.Open(path)
		if err == nil {
			f.Close()
			return path
		}
	}
	return ""
}

func (g *mqlGrubConfig) id() (string, error) {
	return "grub.config:" + g.DefaultsPath.Data + "+" + g.GrubPath.Data, nil
}

func (g *mqlGrubConfig) params() (map[string]any, error) {
	path := g.GetDefaultsPath().Data
	if path == "" {
		return map[string]any{}, nil
	}

	conn, ok := g.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		return nil, errors.New("wrong connection type")
	}
	fs := conn.FileSystem()
	if fs == nil {
		return nil, errors.New("filesystem not available")
	}

	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	params, err := ParseGrubDefaults(f)
	if err != nil {
		return nil, err
	}

	result := make(map[string]any, len(params))
	for k, v := range params {
		result[k] = v
	}
	return result, nil
}

func (g *mqlGrubConfig) entries() ([]any, error) {
	path := g.GetGrubPath().Data
	if path == "" {
		return []any{}, nil
	}

	conn, ok := g.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		return nil, errors.New("wrong connection type")
	}
	fs := conn.FileSystem()
	if fs == nil {
		return nil, errors.New("filesystem not available")
	}

	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	entries, err := ParseGrubCfgEntries(f)
	if err != nil {
		return nil, err
	}

	resources := make([]any, 0, len(entries))
	for i, entry := range entries {
		resource, err := CreateResource(g.MqlRuntime, "grub.config.entry", map[string]*llx.RawData{
			"__id":      llx.StringData("grub.config.entry:" + strconv.Itoa(i) + ":" + entry.Title),
			"title":     llx.StringData(entry.Title),
			"cmdline":   llx.StringData(entry.Cmdline),
			"initrd":    llx.StringData(entry.Initrd),
			"isSubmenu": llx.BoolData(entry.IsSubmenu),
		})
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *mqlGrubConfig) passwordProtected() (bool, error) {
	path := g.GetGrubPath().Data
	if path == "" {
		return false, nil
	}

	conn, ok := g.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		return false, errors.New("wrong connection type")
	}
	fs := conn.FileSystem()
	if fs == nil {
		return false, errors.New("filesystem not available")
	}

	f, err := fs.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	return ParseGrubPasswordProtection(f)
}

func (e *mqlGrubConfigEntry) id() (string, error) {
	return "grub.config.entry:" + e.Title.Data + ":" + e.Cmdline.Data, nil
}

// ParseGrubDefaults parses /etc/default/grub which is a shell-style key=value file.
func ParseGrubDefaults(r io.Reader) (map[string]string, error) {
	params := map[string]string{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments and empty lines
		if line == "" || line[0] == '#' {
			continue
		}
		// Parse KEY=VALUE (shell-style, with optional quoting)
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		// Strip surrounding quotes
		value = stripQuotes(value)
		params[key] = value
	}
	return params, scanner.Err()
}

// stripQuotes removes surrounding single or double quotes from a string.
func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// GrubEntry represents a parsed GRUB menu entry.
type GrubEntry struct {
	Title     string
	Cmdline   string
	Initrd    string
	IsSubmenu bool
}

var (
	reMenuEntry = regexp.MustCompile(`^\s*menuentry\s+['"]([^'"]+)['"]`)
	reSubmenu   = regexp.MustCompile(`^\s*submenu\s+['"]([^'"]+)['"]`)
	reLinux     = regexp.MustCompile(`^\s*(?:linux|linux16|linuxefi)\s+(.+)`)
	reInitrd    = regexp.MustCompile(`^\s*(?:initrd|initrd16|initrdefi)\s+(.+)`)
)

// ParseGrubCfgEntries parses grub.cfg for menuentry and submenu blocks.
func ParseGrubCfgEntries(r io.Reader) ([]GrubEntry, error) {
	var entries []GrubEntry
	var current *GrubEntry
	depth := 0

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		if m := reMenuEntry.FindStringSubmatch(line); m != nil {
			// Flush any in-progress entry that was never closed
			if current != nil {
				entries = append(entries, *current)
			}
			current = &GrubEntry{Title: m[1]}
			depth = 0
			if strings.Contains(line, "{") {
				depth = 1
			}
			continue
		}

		if m := reSubmenu.FindStringSubmatch(line); m != nil {
			// Flush any in-progress entry that was never closed
			if current != nil {
				entries = append(entries, *current)
				current = nil
			}
			entries = append(entries, GrubEntry{Title: m[1], IsSubmenu: true})
			depth = 0
			if strings.Contains(line, "{") {
				depth = 1
			}
			continue
		}

		// Skip comment lines for brace counting to avoid false matches
		// from braces inside comments (e.g., "# echo {something}")
		trimmed := strings.TrimSpace(line)
		isComment := len(trimmed) > 0 && trimmed[0] == '#'

		if current != nil {
			if !isComment {
				if strings.Contains(line, "{") {
					depth++
				}
				if strings.Contains(line, "}") {
					depth--
					if depth <= 0 {
						entries = append(entries, *current)
						current = nil
						depth = 0
						continue
					}
				}
			}

			if m := reLinux.FindStringSubmatch(line); m != nil {
				current.Cmdline = strings.TrimSpace(m[1])
			}
			if m := reInitrd.FindStringSubmatch(line); m != nil {
				current.Initrd = strings.TrimSpace(m[1])
			}
		} else if !isComment {
			// Track braces outside of menu entries (e.g., submenu closing braces)
			if strings.Contains(line, "{") {
				depth++
			}
			if strings.Contains(line, "}") {
				depth--
				if depth < 0 {
					depth = 0
				}
			}
		}
	}

	// If we ended mid-entry (no closing brace), still include it
	if current != nil {
		entries = append(entries, *current)
	}

	return entries, scanner.Err()
}

var (
	reSuperusers = regexp.MustCompile(`(?m)^\s*set\s+superusers\s*=`)
	rePassword   = regexp.MustCompile(`(?m)^\s*password_pbkdf2\s+`)
)

// ParseGrubPasswordProtection reads grub.cfg and checks for password protection.
// GRUB password protection requires both 'set superusers=' and 'password_pbkdf2' directives.
func ParseGrubPasswordProtection(r io.Reader) (bool, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return false, err
	}

	hasSuperusers := reSuperusers.Match(content)
	hasPassword := rePassword.Match(content)

	return hasSuperusers && hasPassword, nil
}

// os.linux accessor
func (s *mqlOsLinux) grub() (*mqlGrubConfig, error) {
	res, err := CreateResource(s.MqlRuntime, "grub.config", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGrubConfig), nil
}
