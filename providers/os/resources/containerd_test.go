// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerdSocketPath(t *testing.T) {
	// Verify the default socket path is set correctly
	assert.Equal(t, "/run/containerd/containerd.sock", defaultContainerdSocket)
}

func TestParseTaskList(t *testing.T) {
	taskOutput := `TASK                                                                PID        STATUS
a5e26880c8937d5d0ee37ffc9c3e3605448ff11a6eab018162beac0be6e66919    3440258    RUNNING
98deb1bb7adca2cce91ae97270238814436c4de3b8b15da64922bea1948f8671    3440318    RUNNING`

	taskInfo := parseTaskList(taskOutput)

	require.Len(t, taskInfo, 2)

	// Check first task - status should be normalized to lowercase
	task1, exists := taskInfo["a5e26880c8937d5d0ee37ffc9c3e3605448ff11a6eab018162beac0be6e66919"]
	require.True(t, exists)
	assert.Equal(t, int64(3440258), task1.pid)
	assert.Equal(t, "running", task1.status)

	// Check second task
	task2, exists := taskInfo["98deb1bb7adca2cce91ae97270238814436c4de3b8b15da64922bea1948f8671"]
	require.True(t, exists)
	assert.Equal(t, int64(3440318), task2.pid)
	assert.Equal(t, "running", task2.status)
}

func TestParseTaskListEmpty(t *testing.T) {
	// Only header, no tasks
	taskOutput := `TASK    PID    STATUS`
	taskInfo := parseTaskList(taskOutput)
	assert.Empty(t, taskInfo)
}

func TestParseTaskListMalformed(t *testing.T) {
	// Lines with insufficient fields should be skipped
	taskOutput := `TASK                                                                PID        STATUS
container1    12345    RUNNING
container2    incomplete`

	taskInfo := parseTaskList(taskOutput)

	require.Len(t, taskInfo, 1)

	// Only first task should be parsed
	task1, exists := taskInfo["container1"]
	require.True(t, exists)
	assert.Equal(t, int64(12345), task1.pid)
	assert.Equal(t, "running", task1.status)

	// Second task should not exist
	_, exists = taskInfo["container2"]
	assert.False(t, exists)
}

func TestParseTaskListWithPausedStatus(t *testing.T) {
	taskOutput := `TASK          PID     STATUS
container1    12345   RUNNING
container2    67890   PAUSED
container3    11111   STOPPED`

	taskInfo := parseTaskList(taskOutput)

	require.Len(t, taskInfo, 3)

	// Status should be normalized to lowercase
	assert.Equal(t, "running", taskInfo["container1"].status)
	assert.Equal(t, "paused", taskInfo["container2"].status)
	assert.Equal(t, "stopped", taskInfo["container3"].status)
}

func TestParseContainerInfo(t *testing.T) {
	// Actual output from ctr -n moby containers info
	jsonOutput := `{
    "ID": "a5e26880c8937d5d0ee37ffc9c3e3605448ff11a6eab018162beac0be6e66919",
    "Labels": {
        "com.docker/engine.bundle.path": "/var/run/docker/containerd/a5e26880c8937d5d0ee37ffc9c3e3605448ff11a6eab018162beac0be6e66919"
    },
    "Image": "",
    "Runtime": {
        "Name": "io.containerd.runc.v2",
        "Options": {
            "type_url": "containerd.runc.v1.Options",
            "value": "MgRydW5jOhwvdmFyL3J1bi9kb2NrZXIvcnVudGltZS1ydW5jSAE="
        }
    },
    "SnapshotKey": "",
    "Snapshotter": "",
    "CreatedAt": "2026-01-16T18:04:53.753134751Z",
    "UpdatedAt": "2026-01-16T18:04:53.753134751Z"
}`

	info, err := parseContainerInfo([]byte(jsonOutput))
	require.NoError(t, err)

	assert.Equal(t, "a5e26880c8937d5d0ee37ffc9c3e3605448ff11a6eab018162beac0be6e66919", info.ID)
	assert.Equal(t, "", info.Image)
	assert.Equal(t, "io.containerd.runc.v2", info.Runtime.Name)
	assert.Equal(t, "", info.Snapshotter)

	require.NotNil(t, info.Labels)
	assert.Equal(t, "/var/run/docker/containerd/a5e26880c8937d5d0ee37ffc9c3e3605448ff11a6eab018162beac0be6e66919",
		info.Labels["com.docker/engine.bundle.path"])
}

func TestParseContainerInfoWithSnapshotter(t *testing.T) {
	// Container with image and snapshotter info
	jsonOutput := `{
    "ID": "test-container",
    "Labels": {
        "app": "web",
        "env": "prod"
    },
    "Image": "docker.io/library/nginx:latest",
    "Runtime": {
        "Name": "io.containerd.runc.v2"
    },
    "Snapshotter": "overlayfs"
}`

	info, err := parseContainerInfo([]byte(jsonOutput))
	require.NoError(t, err)

	assert.Equal(t, "test-container", info.ID)
	assert.Equal(t, "docker.io/library/nginx:latest", info.Image)
	assert.Equal(t, "io.containerd.runc.v2", info.Runtime.Name)
	assert.Equal(t, "overlayfs", info.Snapshotter)
	assert.Len(t, info.Labels, 2)
	assert.Equal(t, "web", info.Labels["app"])
	assert.Equal(t, "prod", info.Labels["env"])
}

func TestParseContainerInfoInvalidJSON(t *testing.T) {
	jsonOutput := `{invalid json}`

	_, err := parseContainerInfo([]byte(jsonOutput))
	assert.Error(t, err)
}

func TestParseNamespaceList(t *testing.T) {
	output := "moby\nk8s.io\ndefault"
	namespaces := parseNamespaceList(output)

	assert.Equal(t, []string{"moby", "k8s.io", "default"}, namespaces)
}

func TestParseNamespaceListSingle(t *testing.T) {
	output := "moby"
	namespaces := parseNamespaceList(output)

	assert.Equal(t, []string{"moby"}, namespaces)
}

func TestParseNamespaceListWithEmptyLines(t *testing.T) {
	output := "moby\n\nk8s.io\n"
	namespaces := parseNamespaceList(output)

	// Empty lines should be filtered out
	assert.Equal(t, []string{"moby", "k8s.io"}, namespaces)
}

func TestParseContainerIDList(t *testing.T) {
	output := `98deb1bb7adca2cce91ae97270238814436c4de3b8b15da64922bea1948f8671
a5e26880c8937d5d0ee37ffc9c3e3605448ff11a6eab018162beac0be6e66919`

	containerIDs := parseContainerIDList(output)

	assert.Equal(t, []string{
		"98deb1bb7adca2cce91ae97270238814436c4de3b8b15da64922bea1948f8671",
		"a5e26880c8937d5d0ee37ffc9c3e3605448ff11a6eab018162beac0be6e66919",
	}, containerIDs)
}

func TestParseContainerIDListSingle(t *testing.T) {
	output := "container1"
	containerIDs := parseContainerIDList(output)

	assert.Equal(t, []string{"container1"}, containerIDs)
}

func TestParseContainerIDListWithEmptyLines(t *testing.T) {
	output := "container1\n\ncontainer2\n"
	containerIDs := parseContainerIDList(output)

	// Empty lines should be filtered out
	assert.Equal(t, []string{"container1", "container2"}, containerIDs)
}

// Note: Integration tests for containerd.containers require a running containerd daemon.
// The implementation uses either:
// - Native Go SDK for local connections (faster, more efficient)
// - CLI commands via the command resource for remote connections (SSH, etc.)
//
// To test manually with containerd running:
//   Local:  cnquery shell local -c "containerd.containers { id status namespace }"
//   Remote: cnquery shell ssh user@host -c "containerd.containers { id status namespace }"
