package aws

import (
	"bytes"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/motorid/awsec2"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
)

const (
	awsIdentifierFileLinux = "/sys/class/dmi/id/product_version"
)

func Detect(t transports.Transport, p *platform.Platform) string {
	if !t.Capabilities().HasCapability(transports.Capability_RunCommand) {
		// we are unable to query for metadata without being able to execute commands
		// we might be able to get the instance id for nitro systems from
		// /sys/devices/virtual/dmi/id/board_asset_tag

		return ""
	}

	if isLinux(p) {
		content, err := afero.ReadFile(t.FS(), awsIdentifierFileLinux)
		if err != nil {
			log.Debug().Err(err).Msgf("unable to read %s", awsIdentifierFileLinux)
			return ""
		}
		if bytes.Contains(content, []byte("amazon")) {
			mdsvc := awsec2.NewUnix(t, p)
			id, err := mdsvc.InstanceID()
			if err != nil {
				log.Debug().Err(err).
					Str("transport", t.Kind().String()).
					Strs("platform", p.GetFamily()).
					Msg("failed to get aws platform id")
				return ""
			}
			return id
		}
	}

	return ""
}

func isLinux(p *platform.Platform) bool {
	for i := range p.Family {
		if p.Family[i] == "linux" {
			return true
		}
	}
	return false
}
