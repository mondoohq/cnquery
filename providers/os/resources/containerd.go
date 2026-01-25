// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/types"
)

const (
	// Default containerd socket path
	defaultContainerdSocket = "/run/containerd/containerd.sock"
)

// containerInfo represents the parsed JSON output from ctr containers info
type containerInfo struct {
	ID      string `json:"ID"`
	Image   string `json:"Image"`
	Runtime struct {
		Name string `json:"Name"`
	} `json:"Runtime"`
	Snapshotter string            `json:"Snapshotter"`
	Labels      map[string]string `json:"Labels"`
}

// taskData holds information about a containerd task
type taskData struct {
	pid    int64
	status string
}

// parseNamespaceList parses the output of "ctr namespaces list -q"
func parseNamespaceList(output string) []string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var nsList []string
	for _, line := range lines {
		if line != "" {
			nsList = append(nsList, line)
		}
	}
	return nsList
}

// parseContainerIDList parses the output of "ctr -n <ns> containers list -q"
func parseContainerIDList(output string) []string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var containerIDs []string
	for _, line := range lines {
		if line != "" {
			containerIDs = append(containerIDs, line)
		}
	}
	return containerIDs
}

// parseTaskList parses the output of "ctr -n <ns> tasks list"
// Format: TASK    PID    STATUS
func parseTaskList(output string) map[string]taskData {
	taskInfo := make(map[string]taskData)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i, line := range lines {
		if i == 0 { // Skip header
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			taskID := fields[0]
			pid, _ := strconv.ParseInt(fields[1], 10, 64)
			status := strings.ToLower(fields[2])
			taskInfo[taskID] = taskData{pid, status}
		}
	}
	return taskInfo
}

// parseContainerInfo parses the JSON output from "ctr -n <ns> containers info <id>"
func parseContainerInfo(jsonData []byte) (*containerInfo, error) {
	var info containerInfo
	if err := json.Unmarshal(jsonData, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

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
	conn := p.MqlRuntime.Connection.(shared.Connection)

	// Use native SDK for local connections, CLI for remote (SSH, etc.)
	if conn.Type() == shared.Type_Local {
		return p.containersViaSDK()
	}
	return p.containersViaCLI()
}

// containersViaSDK uses the containerd Go SDK directly (for local connections)
func (p *mqlContainerd) containersViaSDK() ([]any, error) {
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

			containerRes, err := p.createContainerResource(ns, containerID, info.Image, status, labels, pid, info.Runtime.Name, info.Snapshotter)
			if err != nil {
				return nil, err
			}
			containers = append(containers, containerRes)
		}
	}

	return containers, nil
}

// containersViaCLI uses the ctr CLI via command resource (for remote connections like SSH)
func (p *mqlContainerd) containersViaCLI() ([]any, error) {
	// Get all namespaces using ctr CLI
	o, err := CreateResource(p.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("ctr namespaces list -q"),
	})
	if err != nil {
		return nil, err
	}
	cmd := o.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return nil, errors.New("failed to list namespaces: " + cmd.Stderr.Data)
	}

	nsList := parseNamespaceList(cmd.Stdout.Data)
	var containers []any

	for _, ns := range nsList {
		if ns == "" {
			continue
		}

		// List containers in namespace
		o, err := CreateResource(p.MqlRuntime, "command", map[string]*llx.RawData{
			"command": llx.StringData(fmt.Sprintf("ctr -n %q containers list -q", ns)),
		})
		if err != nil {
			log.Debug().Err(err).Str("namespace", ns).Msg("failed to list containers in namespace, skipping")
			continue
		}
		cmd := o.(*mqlCommand)
		if exit := cmd.GetExitcode(); exit.Data != 0 {
			log.Debug().Str("namespace", ns).Str("stderr", cmd.Stderr.Data).Msg("failed to list containers in namespace, skipping")
			continue
		}

		containerIDs := parseContainerIDList(cmd.Stdout.Data)

		// Get tasks info for this namespace to map PIDs and status
		var taskInfo map[string]taskData

		o, err = CreateResource(p.MqlRuntime, "command", map[string]*llx.RawData{
			"command": llx.StringData(fmt.Sprintf("ctr -n %q tasks list", ns)),
		})
		if err == nil {
			cmd := o.(*mqlCommand)
			if exit := cmd.GetExitcode(); exit.Data == 0 {
				taskInfo = parseTaskList(cmd.Stdout.Data)
			}
		}

		for _, containerID := range containerIDs {
			if containerID == "" {
				continue
			}

			// Get container info as JSON
			o, err := CreateResource(p.MqlRuntime, "command", map[string]*llx.RawData{
				"command": llx.StringData(fmt.Sprintf("ctr -n %q containers info %q", ns, containerID)),
			})
			if err != nil {
				log.Debug().Err(err).Str("container", containerID).Msg("failed to get container info, skipping")
				continue
			}
			cmd := o.(*mqlCommand)
			if exit := cmd.GetExitcode(); exit.Data != 0 {
				log.Debug().Str("container", containerID).Str("stderr", cmd.Stderr.Data).Msg("failed to get container info, skipping")
				continue
			}

			// Parse JSON output from ctr
			info, err := parseContainerInfo([]byte(cmd.Stdout.Data))
			if err != nil {
				log.Debug().Err(err).Str("container", containerID).Msg("failed to parse container info, skipping")
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
			if task, ok := taskInfo[containerID]; ok {
				status = task.status
				pid = task.pid
			}

			containerRes, err := p.createContainerResource(ns, containerID, info.Image, status, labels, pid, info.Runtime.Name, info.Snapshotter)
			if err != nil {
				return nil, err
			}
			containers = append(containers, containerRes)
		}
	}

	return containers, nil
}

// createContainerResource creates a containerd.container MQL resource
func (p *mqlContainerd) createContainerResource(namespace, containerID, image, status string, labels map[string]any, pid int64, runtime, snapshotter string) (*mqlContainerdContainer, error) {
	// Create resource with unique ID combining namespace and container ID
	resourceID := fmt.Sprintf("%s/%s", namespace, containerID)

	containerRes, err := CreateResource(p.MqlRuntime, "containerd.container", map[string]*llx.RawData{
		"__id":        llx.StringData(resourceID),
		"id":          llx.StringData(containerID),
		"image":       llx.StringData(image),
		"status":      llx.StringData(status),
		"labels":      llx.MapData(labels, types.String),
		"pid":         llx.IntData(pid),
		"namespace":   llx.StringData(namespace),
		"runtime":     llx.StringData(runtime),
		"snapshotter": llx.StringData(snapshotter),
	})
	if err != nil {
		return nil, err
	}

	return containerRes.(*mqlContainerdContainer), nil
}

func (p *mqlContainerdContainer) id() (string, error) {
	return p.Id.Data, nil
}
