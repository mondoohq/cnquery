package platform

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/transports"
)

func NewDetector(t transports.Transport) *Detector {
	return &Detector{
		transport: t,
	}
}

type Detector struct {
	transport transports.Transport
	cache     *Platform
}

func (d *Detector) resolveOS() (*Platform, bool) {
	log.Debug().Msg("detector> start resolving the platfrom")
	return operatingSystems.Resolve(d.transport)
}

func (d *Detector) Platform() (*Platform, error) {
	if d.transport == nil {
		return nil, errors.New("cannot detect platform without a transport")
	}

	// check if platform is in cache
	if d.cache != nil {
		return d.cache, nil
	}

	var pi *Platform
	switch d.transport.(type) {
	default:
		var resolved bool
		pi, resolved = d.resolveOS()
		if !resolved {
			return nil, errors.New("could not determine operating system")
		}
	}

	// cache value
	d.cache = pi
	return pi, nil
}
