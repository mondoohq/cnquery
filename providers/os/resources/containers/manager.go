// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package containers

import (
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
)

type OSContainer struct {
	ID      string
	Name    string
	Image   string
	Status  string
	State   string
	Created time.Time
	Runtime string
	Labels  map[string]string
}

type OSContainerManager interface {
	Name() string
	List() ([]*OSContainer, error)
}

func ResolveManager(conn shared.Connection) (OSContainerManager, error) {
	var cm OSContainerManager

	asset := conn.Asset()
	if asset == nil || asset.Platform == nil {
		return nil, errors.New("cannot find OS information for container detection")
	}

	// Try Docker first
	dockerMgr := &DockerManager{conn: conn}
	if dockerMgr.IsAvailable() {
		log.Debug().Str("manager", "docker").Msg("detected docker container runtime")
		return dockerMgr, nil
	}

	// Try containerd
	containerdMgr := &ContainerdManager{conn: conn}
	if containerdMgr.IsAvailable() {
		log.Debug().Str("manager", "containerd").Msg("detected containerd container runtime")
		return containerdMgr, nil
	}

	return cm, errors.New("no container runtime detected")
}
