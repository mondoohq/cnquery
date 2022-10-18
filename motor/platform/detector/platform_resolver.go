package detector

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/tar"
)

type detect func(r *PlatformResolver, pf *platform.Platform, p os.OperatingSystemProvider) (bool, error)

type PlatformResolver struct {
	Name     string
	IsFamily bool
	Children []*PlatformResolver
	Detect   detect
}

func (r *PlatformResolver) Resolve(p os.OperatingSystemProvider) (*platform.Platform, bool) {
	// prepare detect info object
	di := &platform.Platform{}
	di.Family = make([]string, 0)

	// start recursive platform resolution
	pi, resolved := r.resolvePlatform(di, p)

	// if we have a container image use the architecture specified in the transport as it is resolved
	// using the container image properties
	tarTransport, ok := p.(*tar.Provider)
	if resolved && ok {
		pi.Arch = tarTransport.PlatformArchitecture

		// if the platform name is not set, we should fallback to the scratch operating system
		if len(pi.Name) == 0 {
			di.Name = "scratch"
			di.Arch = tarTransport.PlatformArchitecture
			return di, true
		}
	}

	log.Debug().Str("platform", pi.Name).Strs("family", pi.Family).Msg("platform> detected os")
	return pi, resolved
}

// Resolve tries to find recursively all
// platforms until a leaf (operating systems) detect
// mechanism is returning true
func (r *PlatformResolver) resolvePlatform(pf *platform.Platform, p os.OperatingSystemProvider) (*platform.Platform, bool) {
	detected, err := r.Detect(r, pf, p)
	if err != nil {
		return pf, false
	}

	// if detection is true but we have a family
	if detected == true && r.IsFamily == true {
		// we are a family and we may have children to try
		for _, c := range r.Children {
			detected, resolved := c.resolvePlatform(pf, p)
			if resolved {
				// add family hieracy
				detected.Family = append(pf.Family, r.Name)
				return detected, resolved
			}
		}

		// we reached this point, we know it is the platfrom but we could not
		// identify the system
		// TODO: add generic platform instance
		// TODO: should we return an error?
	}

	// return if the detect is true and we have a leaf
	if detected && r.IsFamily == false {
		return pf, true
	}

	// could not find it
	return pf, false
}
