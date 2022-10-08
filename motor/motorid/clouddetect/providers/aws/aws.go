package aws

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/motor/motorid/awsec2"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/resources/packs/os/smbios"
)

func readValue(provider os.OperatingSystemProvider, fPath string) string {
	content, err := afero.ReadFile(provider.FS(), fPath)
	if err != nil {
		log.Debug().Err(err).Msgf("unable to read %s", fPath)
		return ""
	}
	return string(content)
}

func Detect(provider os.OperatingSystemProvider, p *platform.Platform) (string, []string) {
	var values []string
	if p.IsFamily("linux") {
		// Fetching the data from the smbios manager is slow for some transports
		// because it iterates through files we don't need to check. This
		// is an optimization for our sshfs. Also, be aware that on linux,
		// you may not have access to all the smbios things under /sys, so
		// you want to make sure to only check the files we actually look at

		values = []string{
			readValue(provider, "/sys/class/dmi/id/product_version"),
			readValue(provider, "/sys/class/dmi/id/bios_vendor"),
		}
	} else {
		mgr, err := smbios.ResolveManager(provider, p)
		if err != nil {
			return "", nil
		}
		info, err := mgr.Info()
		if err != nil {
			log.Debug().Err(err).Msg("failed to query smbios")
			return "", nil
		}
		values = []string{
			info.SysInfo.Version,
			info.BIOS.Vendor,
		}
	}

	for _, v := range values {
		if strings.Contains(strings.ToLower(v), "amazon") {
			mdsvc, err := awsec2.Resolve(provider, p)
			if err != nil {
				log.Debug().Err(err).Msg("failed to get metadata resolver")
				return "", nil
			}
			id, err := mdsvc.Identify()
			if err != nil {
				log.Debug().Err(err).
					Str("transport", provider.Kind().String()).
					Strs("platform", p.GetFamily()).
					Msg("failed to get aws platform id")
				return "", nil
			}
			return id.InstanceID, []string{id.AccountID}
		}
	}

	return "", nil
}
