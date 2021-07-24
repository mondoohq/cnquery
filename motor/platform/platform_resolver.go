package platform

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/tar"
)

type detect func(p *PlatformResolver, di *Platform, t transports.Transport) (bool, error)

type PlatformResolver struct {
	Name      string
	IsFamiliy bool
	Children  []*PlatformResolver
	Detect    detect
}

func (p *PlatformResolver) Resolve(t transports.Transport) (*Platform, bool) {
	// prepare detect info object
	di := &Platform{}
	di.Family = make([]string, 0)

	// start recursive platform resolution
	pi, resolved := p.resolvePlatform(di, t)

	// if we have a docker image, we should fallback to the scratch operating system
	_, ok := t.(*tar.Transport)
	if resolved && len(pi.Name) == 0 && ok {
		di.Name = "scratch"
		return di, true
	}

	log.Debug().Str("platform", pi.Name).Strs("family", pi.Family).Msg("platform> detected os")
	return pi, resolved
}

// Resolve tries to find recursively all
// platforms until a leaf (operating systems) detect
// mechanism is returning true
func (p *PlatformResolver) resolvePlatform(di *Platform, t transports.Transport) (*Platform, bool) {
	detected, err := p.Detect(p, di, t)
	if err != nil {
		return di, false
	}

	// if detection is true but we have a family
	if detected == true && p.IsFamiliy == true {
		// we are a familiy and we may have childs to try
		for _, c := range p.Children {
			detected, resolved := c.resolvePlatform(di, t)
			if resolved {
				// add family hieracy
				detected.Family = append(di.Family, p.Name)
				return detected, resolved
			}
		}

		// we reached this point, we know it is the platfrom but we could not
		// identify the system
		// TODO: add generic platform instance
		// TODO: should we return an error?
	}

	// return if the detect is true and we have a leaf
	if detected && p.IsFamiliy == false {
		return di, true
	}

	// could not find it
	return di, false
}
