package platform

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/transports"
)

type PlatformInfo struct {
	Name    string   `json:"name"`
	Title   string   `json:"title"`
	Family  []string `json:"family"`
	Release string   `json:"release"`
	Arch    string   `json:"arch"`
}

func (di *PlatformInfo) IsFamily(family string) bool {
	for i := range di.Family {
		if di.Family[i] == family {
			return true
		}
	}
	return false
}

type Detector struct {
	Transport transports.Transport
}

func (d *Detector) Resolve() (bool, *PlatformInfo) {
	log.Debug().Msg("detector> start resolving the platfrom")
	return operatingSystems.Resolve(d.Transport)
}
