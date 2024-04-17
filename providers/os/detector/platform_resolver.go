// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/docker"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/connection/tar"
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
	platform := &inventory.Platform{}
	platform.Family = make([]string, 0)

	// start recursive platform resolution
	pi, resolved := r.resolvePlatform(platform, conn)

	// if we have a container image use the architecture specified in the transport as it is resolved
	// using the container image properties
	tarConn, ok := conn.(*tar.Connection)
	if resolved && ok {
		pi.Arch = tarConn.PlatformArchitecture
		platform.Runtime = "docker-image"
		platform.Kind = "container-image"

		// if the platform name is not set, we should fallback to the scratch operating system
		if len(pi.Name) == 0 {
			platform.Name = "scratch"
			platform.Arch = tarConn.PlatformArchitecture
			return platform, true
		}
	}

	containerConn, ok := conn.(*docker.ContainerConnection)
	if resolved && ok {
		pi.Arch = containerConn.PlatformArchitecture
		platform.Runtime = string(containerConn.Type())
		platform.Kind = "container"

		// if the platform name is not set, we should fallback to the scratch operating system
		if len(pi.Name) == 0 {
			platform.Name = "scratch"
			platform.Arch = pi.Arch
			return platform, true
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
