// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/types"
)

const (
	// Default containerd socket path
	defaultContainerdSocket = "/run/containerd/containerd.sock"
)

// containerdClient creates a new containerd client connected to the default socket
func containerdClient() (*client.Client, error) {
	cl, err := client.New(defaultContainerdSocket)
	if err != nil {
		return nil, err
	}
	log.Debug().Msg("containerd client> connected to containerd socket")
	return cl, nil
}

func (p *mqlContainerd) containers() ([]any, error) {
	ctx := context.Background()

	cl, err := containerdClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to containerd: %w", err)
	}
	defer cl.Close()

	// Get all namespaces
	nsService := cl.NamespaceService()
	nsList, err := nsService.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	var containers []any

	for _, ns := range nsList {
		if ns == "" {
			continue
		}

		// Set the namespace context for this iteration
		nsCtx := namespaces.WithNamespace(ctx, ns)

		// List containers in this namespace
		ctrs, err := cl.Containers(nsCtx)
		if err != nil {
			log.Debug().Err(err).Str("namespace", ns).Msg("failed to list containers in namespace, skipping")
			continue
		}

		for _, ctr := range ctrs {
			containerID := ctr.ID()

			// Get container info
			info, err := ctr.Info(nsCtx)
			if err != nil {
				log.Debug().Err(err).Str("container", containerID).Msg("failed to get container info, skipping")
				continue
			}

			// Convert labels to map[string]any
			labels := make(map[string]any)
			for k, v := range info.Labels {
				labels[k] = v
			}

			// Get task info - default to "created" if no task exists
			status := "created"
			var pid int64

			task, err := ctr.Task(nsCtx, nil)
			if err == nil {
				// Task exists, get its status and PID
				pid = int64(task.Pid())

				taskStatus, err := task.Status(nsCtx)
				if err == nil {
					status = strings.ToLower(string(taskStatus.Status))
				}
			}
			// If task doesn't exist, container is in "created" state

			// Create resource with unique ID combining namespace and container ID
			resourceID := fmt.Sprintf("%s/%s", ns, containerID)

			containerRes, err := CreateResource(p.MqlRuntime, "containerd.container", map[string]*llx.RawData{
				"__id":        llx.StringData(resourceID),
				"id":          llx.StringData(containerID),
				"image":       llx.StringData(info.Image),
				"status":      llx.StringData(status),
				"labels":      llx.MapData(labels, types.String),
				"pid":         llx.IntData(pid),
				"namespace":   llx.StringData(ns),
				"runtime":     llx.StringData(info.Runtime.Name),
				"snapshotter": llx.StringData(info.Snapshotter),
			})
			if err != nil {
				return nil, err
			}

			containers = append(containers, containerRes.(*mqlContainerdContainer))
		}
	}

	return containers, nil
}

func (p *mqlContainerdContainer) id() (string, error) {
	return p.Id.Data, nil
}
