// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/id/azcompute"
	"go.mondoo.com/cnquery/v11/providers/os/resources/smbios"
)

const (
	azureIdentifierFileLinux = "/sys/class/dmi/id/sys_vendor"
)

func Detect(conn shared.Connection, pf *inventory.Platform, smbiosMgr smbios.SmBiosManager) (string, string, []string) {
	sysVendor := ""
	if pf.IsFamily("linux") {
		// Fetching the product version from the smbios manager is slow
		// because it iterates through files we don't need to check. This
		// is an optimization for our sshfs. Also, be aware that on linux,
		// you may not have access to all the smbios things under /sys, so
		// you want to make sure to only check the product_version file
		content, err := afero.ReadFile(conn.FileSystem(), azureIdentifierFileLinux)
		if err != nil {
			log.Debug().Err(err).Msgf("unable to read %s", azureIdentifierFileLinux)
			return "", "", nil
		}
		sysVendor = string(content)
	} else {
		info, err := smbiosMgr.Info()
		if err != nil {
			log.Debug().Err(err).Msg("failed to query smbios")
			return "", "", nil
		}
		sysVendor = info.SysInfo.Vendor
	}

	if strings.Contains(sysVendor, "Microsoft Corporation") {
		mdsvc, err := azcompute.Resolve(conn, pf)
		if err != nil {
			log.Debug().Err(err).Msg("failed to get metadata resolver")
			return "", "", nil
		}
		id, err := mdsvc.Identify()
		if err != nil {
			log.Debug().Err(err).
				Strs("platform", pf.GetFamily()).
				Msg("failed to get Azure platform id")
			return "", "", nil
		}
		return id.InstanceID, "", []string{id.AccountID}
	}

	return "", "", nil
}
