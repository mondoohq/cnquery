package platform

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/transports"
)

type Detector struct {
	Transport transports.Transport
	cache     *Platform
}

func (d *Detector) Resolve() (*Platform, bool) {
	log.Debug().Msg("detector> start resolving the platfrom")
	return operatingSystems.Resolve(d.Transport)
}

func (d *Detector) Platform() (*Platform, error) {
	if d.Transport == nil {
		return nil, errors.New("cannot detect platform without a transport")
	}

	// check if platform is in cache
	if d.cache != nil {
		return d.cache, nil
	}

	di, resolved := d.Resolve()
	if !resolved {
		return nil, errors.New("could not determine operating system")
	}

	// cache value
	d.cache = di
	return di, nil
}
