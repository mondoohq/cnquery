package gce

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/motorid/gce"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/providers/os"
	"go.mondoo.io/mondoo/resources/packs/os/smbios"
)

const (
	gceIdentifierFileLinux = "/sys/class/dmi/id/product_name"
)

func Detect(provider os.OperatingSystemProvider, pf *platform.Platform) string {
	productName := ""
	if pf.IsFamily("linux") {
		// Fetching the product version from the smbios manager is slow
		// because it iterates through files we don't need to check. This
		// is an optimzation for our sshfs. Also, be aware that on linux,
		// you may not have access to all the smbios things under /sys, so
		// you want to make sure to only check the one file
		content, err := afero.ReadFile(provider.FS(), gceIdentifierFileLinux)
		if err != nil {
			log.Debug().Err(err).Msgf("unable to read %s", gceIdentifierFileLinux)
			return ""
		}
		productName = string(content)
	} else {
		mgr, err := smbios.ResolveManager(provider, pf)
		if err != nil {
			return ""
		}
		info, err := mgr.Info()
		if err != nil {
			log.Debug().Err(err).Msg("failed to query smbios")
			return ""
		}
		productName = info.SysInfo.Model
	}

	if strings.Contains(productName, "Google Compute Engine") {
		mdsvc, err := gce.Resolve(provider, pf)
		if err != nil {
			log.Debug().Err(err).Msg("failed to get metadata resolver")
			return ""
		}
		id, err := mdsvc.InstanceID()
		if err != nil {
			log.Debug().Err(err).
				Str("transport", provider.Kind().String()).
				Strs("platform", pf.GetFamily()).
				Msg("failed to get gce platform id")
			return ""
		}
		return id
	}

	return ""
}
