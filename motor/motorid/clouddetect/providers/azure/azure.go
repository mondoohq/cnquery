package azure

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/motor/motorid/azcompute"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/resources/packs/os/smbios"
)

const (
	azureIdentifierFileLinux = "/sys/class/dmi/id/sys_vendor"
)

func Detect(provider os.OperatingSystemProvider, pf *platform.Platform) (string, []string) {
	sysVendor := ""
	if pf.IsFamily("linux") {
		// Fetching the product version from the smbios manager is slow
		// because it iterates through files we don't need to check. This
		// is an optimization for our sshfs. Also, be aware that on linux,
		// you may not have access to all the smbios things under /sys, so
		// you want to make sure to only check the product_version file
		content, err := afero.ReadFile(provider.FS(), azureIdentifierFileLinux)
		if err != nil {
			log.Debug().Err(err).Msgf("unable to read %s", azureIdentifierFileLinux)
			return "", nil
		}
		sysVendor = string(content)
	} else {
		mgr, err := smbios.ResolveManager(provider, pf)
		if err != nil {
			return "", nil
		}
		info, err := mgr.Info()
		if err != nil {
			log.Debug().Err(err).Msg("failed to query smbios")
			return "", nil
		}
		sysVendor = info.SysInfo.Vendor
	}

	if strings.Contains(sysVendor, "Microsoft Corporation") {
		mdsvc, err := azcompute.Resolve(provider, pf)
		if err != nil {
			log.Debug().Err(err).Msg("failed to get metadata resolver")
			return "", nil
		}
		id, err := mdsvc.Identify()
		if err != nil {
			log.Debug().Err(err).
				Str("transport", provider.Kind().String()).
				Strs("platform", pf.GetFamily()).
				Msg("failed to get azure platform id")
			return "", nil
		}
		return id.InstanceID, []string{id.AccountID}
	}

	return "", nil
}
