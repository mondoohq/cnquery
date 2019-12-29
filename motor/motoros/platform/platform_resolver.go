package platform

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

type detect func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error)

type PlatformResolver struct {
	Name     string
	Familiy  bool
	Children []*PlatformResolver
	Detect   detect
}

func (p *PlatformResolver) Resolve(t types.Transport) (bool, *PlatformInfo) {
	// prepare detect info object
	di := &PlatformInfo{}
	di.Family = make([]string, 0)

	// start recursive platform resolution
	ok, pi := p.resolvePlatform(di, t)
	log.Debug().Str("platform", pi.Name).Strs("family", pi.Family).Msg("platform> detected os")
	return ok, pi
}

// Resolve tries to find recursively all
// platforms until a leaf (operating systems) detect
// mechanism is returning true
func (p *PlatformResolver) resolvePlatform(di *PlatformInfo, t types.Transport) (bool, *PlatformInfo) {
	detected, err := p.Detect(p, di, t)
	if err != nil {
		return false, di
	}

	// if detection is true but we have a family
	if detected == true && p.Familiy == true {
		// we are a familiy and we may have childs to try
		for _, c := range p.Children {
			resolved, detected := c.resolvePlatform(di, t)
			if resolved {
				// add family hieracy
				detected.Family = append(di.Family, p.Name)
				return resolved, detected
			}
		}

		// we reached this point, we know it is the platfrom but we could not
		// identify the system
		// TODO: add generic platform instance
		// TODO: should we return an error?
	}

	// return if the detect is true and we have a leaf
	if detected && p.Familiy == false {
		return true, di
	}

	// could not find it
	return false, di
}
