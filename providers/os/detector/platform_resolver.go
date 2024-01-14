// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
)

type detect func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error)

type PlatformResolver struct {
	Name     string
	IsFamily bool
	Children []*PlatformResolver
	Detect   detect
}

func (r *PlatformResolver) Resolve(conn shared.Connection) (*inventory.Platform, bool) {
	// prepare detect info object
	di := &inventory.Platform{}
	di.Family = make([]string, 0)

	// start recursive platform resolution
	pi, resolved := r.resolvePlatform(di, conn)

	// if we have a container image use the architecture specified in the transport as it is resolved
	// using the container image properties
	tarConn, ok := conn.(*connection.TarConnection)
	if resolved && ok {
		pi.Arch = tarConn.PlatformArchitecture
		di.Runtime = "docker-image"
		di.Kind = "container-image"

		// if the platform name is not set, we should fallback to the scratch operating system
		if len(pi.Name) == 0 {
			di.Name = "scratch"
			di.Arch = tarConn.PlatformArchitecture
			return di, true
		}
	}

	containerConn, ok := conn.(*connection.DockerContainerConnection)
	if resolved && ok {
		pi.Arch = containerConn.PlatformArchitecture
		di.Runtime = string(containerConn.Type())
		di.Kind = "container"

		// if the platform name is not set, we should fallback to the scratch operating system
		if len(pi.Name) == 0 {
			di.Name = "scratch"
			di.Arch = pi.Arch
			return di, true
		}
	}

	log.Debug().Str("platform", pi.Name).Strs("family", pi.Family).Msg("platform> detected os")
	return pi, resolved
}

// Resolve tries to find recursively all
// platforms until a leaf (operating systems) detect
// mechanism is returning true
func (r *PlatformResolver) resolvePlatform(pf *inventory.Platform, conn shared.Connection) (*inventory.Platform, bool) {
	detected, err := r.Detect(r, pf, conn)
	if err != nil {
		return pf, false
	}

	// if detection is true but we have a family
	if detected == true && r.IsFamily == true {
		// we are a family and we may have children to try
		for _, c := range r.Children {
			detected, resolved := c.resolvePlatform(pf, conn)
			if resolved {
				// add family hierarchy
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
