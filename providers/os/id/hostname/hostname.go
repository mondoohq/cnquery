// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hostname

import (
	"io"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
)

func Hostname(conn shared.Connection, pf *inventory.Platform) (string, bool) {
	var hostname string

	if !pf.IsFamily(inventory.FAMILY_UNIX) && !pf.IsFamily(inventory.FAMILY_WINDOWS) {
		log.Warn().Msg("your platform is not supported for hostname detection")
		return hostname, false
	}

	// linux:
	// we prefer the hostname over /etc/hostname since systemd is not updating the value all the time
	//
	// windows:
	// hostname command works more reliable than t.RunCommand("powershell -c \"$env:computername\"")
	// since it will return a non-zero exit code.
	cmd, err := conn.RunCommand("hostname")
	if err == nil && cmd.ExitStatus == 0 {
		data, err := io.ReadAll(cmd.Stdout)
		if err == nil {
			return strings.TrimSpace(string(data)), true
		}
	} else {
		log.Debug().Err(err).Msg("could not run hostname command")
	}

	// try to use /etc/hostname since it's also working on static analysis
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

	return "", false
}
