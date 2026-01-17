// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/resources/containers"
)

func initContainer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// If we already have all parameters, no need to resolve
	if len(args) > 2 {
		return args, nil, nil
	}

	idValue, ok := args["id"]
	if ok {
		_, ok := idValue.Value.(string)
		if !ok {
			return nil, nil, errors.New("id has invalid type")
		}
	}
	return args, nil, nil
}

func (c *mqlContainer) id() (string, error) {
	return c.Id.Data, nil
}

func (c *mqlContainers) list() ([]any, error) {
	conn := c.MqlRuntime.Connection.(shared.Connection)
	cm, err := containers.ResolveManager(conn)
	if err != nil {
		log.Debug().Err(err).Msg("mql[containers]> could not retrieve container manager")
		return nil, errors.New("cannot find container runtime")
	}

	// Retrieve all containers
	containerList, err := cm.List()
	if err != nil {
		log.Warn().Err(err).Msg("mql[containers]> could not retrieve container list")
		return nil, err
	}
	log.Debug().Int("containers", len(containerList)).Msg("mql[containers]> found containers")

	result := make([]any, len(containerList))
	for i, container := range containerList {
		labels := make(map[string]any)
		for k, v := range container.Labels {
			labels[k] = v
		}

		o, err := CreateResource(c.MqlRuntime, "container", map[string]*llx.RawData{
			"id":        llx.StringData(container.ID),
			"name":      llx.StringData(container.Name),
			"image":     llx.StringData(container.Image),
			"status":    llx.StringData(container.Status),
			"state":     llx.StringData(container.State),
			"createdAt": llx.TimeData(container.Created),
			"runtime":   llx.StringData(container.Runtime),
			"labels":    llx.MapData(labels, "string"),
		})
		if err != nil {
			return nil, err
		}

		result[i] = o
	}

	return result, nil
}

func (c *mqlContainers) running() ([]any, error) {
	// Get all containers first
	allContainers := c.GetList()
	if allContainers.Error != nil {
		return nil, allContainers.Error
	}

	// Filter for running containers
	result := []any{}
	for _, containerInterface := range allContainers.Data {
		container := containerInterface.(*mqlContainer)
		state := container.GetState()
		if state.Error != nil {
			continue
		}
		// Check if state is running (different runtimes may use different values)
		if state.Data == "running" || state.Data == "Running" {
			result = append(result, container)
		}
	}

	return result, nil
}
