// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package aws

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/id/awsec2"
	"go.mondoo.com/cnquery/v11/providers/os/id/awsecs"
	"go.mondoo.com/cnquery/v11/providers/os/resources/smbios"
)

func readValue(conn shared.Connection, fPath string) string {
	content, err := afero.ReadFile(conn.FileSystem(), fPath)
	if err != nil {
		log.Debug().Err(err).Msgf("unable to read %s", fPath)
		return ""
	}
	return string(content)
}

func Detect(conn shared.Connection, p *inventory.Platform, smbiosMgr smbios.SmBiosManager) (string, string, []string) {
	var values []string
	if p.IsFamily("linux") {
		// Fetching the data from the smbios manager is slow for some transports
		// because it iterates through files we don't need to check. This
		// is an optimization for our sshfs. Also, be aware that on linux,
		// you may not have access to all the smbios things under /sys, so
		// you want to make sure to only check the files we actually look at

		values = []string{
			readValue(conn, "/sys/class/dmi/id/product_version"),
			readValue(conn, "/sys/class/dmi/id/bios_vendor"),
		}
	} else {
		info, err := smbiosMgr.Info()
		if err != nil {
			log.Debug().Err(err).Msg("failed to query smbios")
			return "", "", nil
		}
		values = []string{
			info.SysInfo.Version,
			info.BIOS.Vendor,
		}
	}

	isAws := false
	for _, v := range values {
		if strings.Contains(strings.ToLower(v), "amazon") {
			isAws = true
			break
		}
	}

	if isAws {
		mdsvc, err := awsec2.Resolve(conn, p)
		if err != nil {
			log.Debug().Err(err).Msg("failed to get metadata resolver")
			return "", "", nil
		}
		id, err := mdsvc.Identify()
		if err == nil {
			return id.InstanceID, id.InstanceName, []string{id.AccountID}
		}
		log.Debug().Err(err).
			Strs("platform", p.GetFamily()).
			Msg("failed to get AWS platform id")
		// try ecs
		mdsvcEcs, err := awsecs.Resolve(conn, p)
		if err != nil {
			log.Debug().Err(err).Msg("failed to get metadata resolver")
			return "", "", nil
		}
		idEcs, err := mdsvcEcs.Identify()
		if err == nil {
			return idEcs.PlatformIds[0], idEcs.Name, []string{idEcs.AccountPlatformID}
		}
		log.Debug().Err(err).
			Strs("platform", p.GetFamily()).
			Msg("failed to get AWS platform id")
	}

	return "", "", nil
}
