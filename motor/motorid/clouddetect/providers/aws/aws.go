package aws

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/motorid/awsec2"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/providers/os"
	"go.mondoo.io/mondoo/resources/packs/os/smbios"
)

func readValue(provider os.OperatingSystemProvider, fPath string) string {
	content, err := afero.ReadFile(provider.FS(), fPath)
	if err != nil {
		log.Debug().Err(err).Msgf("unable to read %s", fPath)
		return ""
	}
	return string(content)
}

func Detect(provider os.OperatingSystemProvider, p *platform.Platform) string {
	var values []string
	if p.IsFamily("linux") {
		// Fetching the data from the smbios manager is slow for some transports
		// because it iterates through files we don't need to check. This
		// is an optimzation for our sshfs. Also, be aware that on linux,
		// you may not have access to all the smbios things under /sys, so
		// you want to make sure to only check the files we actually look at

		values = []string{
			readValue(provider, "/sys/class/dmi/id/product_version"),
			readValue(provider, "/sys/class/dmi/id/bios_vendor"),
		}
	} else {
		mgr, err := smbios.ResolveManager(provider, p)
		if err != nil {
			return ""
		}
		info, err := mgr.Info()
		if err != nil {
			log.Debug().Err(err).Msg("failed to query smbios")
			return ""
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
				return ""
			}
			id, err := mdsvc.InstanceID()
			if err != nil {
				log.Debug().Err(err).
					Str("transport", provider.Kind().String()).
					Strs("platform", p.GetFamily()).
					Msg("failed to get aws platform id")
				return ""
			}
			return id
		}
	}

	return ""
}
