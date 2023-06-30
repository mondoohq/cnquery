package gcp

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/providers/os/connection"
	"go.mondoo.com/cnquery/providers/os/id/gce"
	"go.mondoo.com/cnquery/resources/packs/os/smbios"
)

const (
	gceIdentifierFileLinux = "/sys/class/dmi/id/product_name"
)

func Detect(conn connection.Connection, p *platform.Platform) (string, string, []string) {
	productName := ""
	if p.IsFamily("linux") {
		// Fetching the product version from the smbios manager is slow
		// because it iterates through files we don't need to check. This
		// is an optimization for our sshfs. Also, be aware that on linux,
		// you may not have access to all the smbios things under /sys, so
		// you want to make sure to only check the one file
		content, err := afero.ReadFile(conn.FileSystem(), gceIdentifierFileLinux)
		if err != nil {
			log.Debug().Err(err).Msgf("unable to read %s", gceIdentifierFileLinux)
			return "", "", nil
		}
		productName = string(content)
	} else {
		mgr, err := smbios.ResolveManager(conn, p)
		if err != nil {
			return "", "", nil
		}
		info, err := mgr.Info()
		if err != nil {
			log.Debug().Err(err).Msg("failed to query smbios")
			return "", "", nil
		}
		productName = info.SysInfo.Model
	}

	if strings.Contains(productName, "Google Compute Engine") {
		mdsvc, err := gce.Resolve(conn, p)
		if err != nil {
			log.Debug().Err(err).Msg("failed to get metadata resolver")
			return "", "", nil
		}
		id, err := mdsvc.Identify()
		if err != nil {
			log.Debug().Err(err).
				Strs("platform", p.GetFamily()).
				Msg("failed to get GCE platform id")
			return "", "", nil
		}
		return id.InstanceID, "", []string{id.ProjectID}
	}
	return "", "", nil
}
