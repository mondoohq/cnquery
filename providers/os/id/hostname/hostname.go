// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hostname

import (
	"fmt"
	"io"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/registry"
)

// Hostname returns the hostname of the system.

// On Linux systems we prefer `hostname -f` over `/etc/hostname` since systemd is not updating the value all the time.
// On Windows the `hostname` command (without the -f flag) works more reliable than `powershell -c "$env:computername"`
// since it will return a non-zero exit code.
func Hostname(conn shared.Connection, pf *inventory.Platform) (string, bool) {
	if !pf.IsFamily(inventory.FAMILY_UNIX) && !pf.IsFamily(inventory.FAMILY_WINDOWS) {
		log.Warn().Msg("your platform is not supported for hostname detection")
		return "", false
	}

	// On unix systems we try to get the hostname via `hostname -f` first since it returns the fqdn.
	if pf.IsFamily(inventory.FAMILY_UNIX) {
		fqdn, err := runCommand(conn, "hostname -f")
		if err == nil && fqdn != "localhost" && fqdn != "" {
			return fqdn, true
		}
		log.Debug().Err(err).Msg("could not detect hostname via `hostname -f` command")

		// If the output of `hostname -f` is localhost, we try to fetch it via `getent hosts`,
		// start with the most common protocol IPv4.
		hostname, err := parseGetentHosts(conn, "127.0.0.1")
		if err == nil && hostname != "" {
			return hostname, true
		}
		log.Debug().Err(err).Str("ipversion", "IPv4").Msg("could not detect hostname")

		// When IPv4 is not configured, try IPv6.
		hostname, err = parseGetentHosts(conn, "::1")
		if err == nil && hostname != "" {
			return hostname, true
		}
		log.Debug().Err(err).Str("ipversion", "IPv6").Msg("could not detect hostname")
	}

	// This is the preferred way to get the hostname on windows, it is important to not use the -f flag here
	hostname, err := runCommand(conn, "hostname")
	if err == nil && hostname != "" {
		return hostname, true
	}
	log.Debug().Err(err).Msg("could not run `hostname` command")

	// Fallback to for unix systems to /etc/hostname, since hostname command is not available on all systems
	// This mechanism is also working for static analysis
	if pf.IsFamily(inventory.FAMILY_LINUX) {
		afs := &afero.Afero{Fs: conn.FileSystem()}
		ok, err := afs.Exists("/etc/hostname")
		if err == nil && ok {
			content, err := afs.ReadFile("/etc/hostname")
			if err == nil {
				return strings.TrimSpace(string(content)), true
			}
		} else {
			log.Debug().Err(err).Msg("could not read /etc/hostname file")
		}
	}

	// Fallback for windows systems to using registry for static analysis
	if pf.IsFamily(inventory.FAMILY_WINDOWS) && conn.Capabilities().Has(shared.Capability_FileSearch) {
		fi, err := conn.FileInfo(registry.SystemRegPath)
		if err != nil {
			log.Debug().Err(err).Msg("could not find SYSTEM registry file, cannot perform hostname lookup")
			return "", false
		}

		rh := registry.NewRegistryHandler()
		defer func() {
			err := rh.UnloadSubkeys()
			if err != nil {
				log.Debug().Err(err).Msg("could not unload registry subkeys")
			}
		}()
		err = rh.LoadSubkey(registry.System, fi.Path)
		if err != nil {
			log.Debug().Err(err).Msg("could not load SYSTEM registry key file")
			return "", false
		}
		key, err := rh.GetRegistryItemValue(registry.System, "ControlSet001\\Control\\ComputerName\\ComputerName", "ComputerName")
		if err == nil {
			return key.Value.String, true
		}

		// we also can try ControlSet002 as a fallback
		log.Debug().Err(err).Msg("unable to read windows registry, trying ControlSet002 fallback")
		key, err = rh.GetRegistryItemValue(registry.System, "ControlSet002\\Control\\ComputerName\\ComputerName", "ComputerName")
		if err == nil {
			return key.Value.String, true
		}
	}

	return "", false
}

// runCommand is a wrapper around shared.Connection.RunCommand that helps execute commands
// and read the standard output all in one function.
func runCommand(conn shared.Connection, commandString string) (string, error) {
	cmd, err := conn.RunCommand(commandString)
	if err != nil {
		return "", err
	}

	if cmd.ExitStatus != 0 {
		outErr, err := io.ReadAll(cmd.Stderr)
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("failed to run command: %s", outErr)
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

// parseGetentHosts runs `getent hosts <address>` and returns the first valid hostname
// that is not a variant of "localhost".
func parseGetentHosts(conn shared.Connection, ip string) (string, error) {
	output, err := runCommand(conn, fmt.Sprintf("getent hosts %s", ip))
	if err != nil {
		return "", err
	}

	fields := strings.Fields(output)

	if len(fields) < 2 {
		return "", fmt.Errorf("no hostnames found for IP %s", ip)
	}

	for _, host := range fields[1:] {
		if !isLocalhostVariant(host) {
			return host, nil
		}
	}

	return "", fmt.Errorf("no non-localhost hostname found for IP %s", ip)
}

// isLocalhostVariant returns true if the given hostname is a variant of "localhost"
func isLocalhostVariant(host string) bool {
	lh := strings.ToLower(host)
	return lh == "localhost" ||
		lh == "localhost.localdomain" ||
		lh == "ip6-localhost" ||
		lh == "ip6-loopback"
}
