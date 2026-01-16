// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package containers

import (
	"errors"
	"time"

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
		return dockerMgr, nil
	}

	// Try Podman
	podmanMgr := &PodmanManager{conn: conn}
	if podmanMgr.IsAvailable() {
		return podmanMgr, nil
	}

	// Try containerd
	containerdMgr := &ContainerdManager{conn: conn}
	if containerdMgr.IsAvailable() {
		return containerdMgr, nil
	}

	return cm, errors.New("no container runtime detected")
}
