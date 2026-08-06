// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captured from "podman ps -a --format json" on Podman 6
const podmanTestPs = `[
  {
    "AutoRemove": false,
    "Command": ["sleep", "3600"],
    "CreatedAt": "2 minutes ago",
    "Exited": false,
    "ExitCode": 0,
    "Id": "55a0b364793a3cef9ee0e17e8d5153f9979d827374e57437a9b910924bba0bfd",
    "Image": "localhost/mql-os-verify:latest",
    "ImageID": "8ce27058e51034510c145f57b370ef9df424693c82c57e86af500b141be2ccc8",
    "IsInfra": false,
    "Labels": {"demo": "true"},
    "Mounts": [],
    "Names": ["mql-verify-ctr"],
    "Networks": ["podman"],
    "Pid": 1234,
    "Pod": "",
    "PodName": "",
    "Ports": [
      {"host_ip": "", "container_port": 5432, "host_port": 40441, "range": 1, "protocol": "tcp"}
    ],
    "State": "running",
    "Status": "Up 2 minutes",
    "Created": 1784678819
  }
]`

func TestParsePodmanPs(t *testing.T) {
	entries, err := parsePodmanPs(podmanTestPs)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.Equal(t, "55a0b364793a3cef9ee0e17e8d5153f9979d827374e57437a9b910924bba0bfd", entry.ID)
	assert.Equal(t, []string{"mql-verify-ctr"}, entry.Names)
	assert.Equal(t, "localhost/mql-os-verify:latest", entry.Image)
	assert.Equal(t, []string{"sleep", "3600"}, entry.Command)
	assert.Equal(t, "running", entry.State)
	assert.False(t, entry.IsInfra)
	assert.Equal(t, map[string]string{"demo": "true"}, entry.Labels)
	require.Len(t, entry.Ports, 1)
	assert.Equal(t, int64(5432), entry.Ports[0].ContainerPort)
	assert.Equal(t, int64(40441), entry.Ports[0].HostPort)
	assert.Equal(t, "tcp", entry.Ports[0].Protocol)
}

func TestParsePodmanPs_EmptyOutput(t *testing.T) {
	// some subcommands print nothing rather than an empty array
	for _, out := range []string{"", "  \n", "null", "[]"} {
		entries, err := parsePodmanPs(out)
		require.NoError(t, err, out)
		assert.Empty(t, entries, out)
	}
}

func TestParsePodmanInspect(t *testing.T) {
	entries, err := parsePodmanInspect(`[
  {
    "Id": "55a0b364793a",
    "EffectiveCaps": ["CAP_CHOWN", "CAP_DAC_OVERRIDE"],
    "BoundingCaps": ["CAP_CHOWN", "CAP_DAC_OVERRIDE", "CAP_SYS_TIME"],
    "Config": {"User": "postgres"},
    "HostConfig": {
      "Privileged": false,
      "CapAdd": ["CAP_SYS_TIME"],
      "CapDrop": [],
      "SecurityOpt": ["seccomp=unconfined"],
      "ReadonlyRootfs": true,
      "NetworkMode": "bridge",
      "PidMode": "private",
      "UsernsMode": "",
      "RestartPolicy": {"Name": "always"}
    }
  }
]`)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.False(t, entry.HostConfig.Privileged)
	assert.Equal(t, []string{"CAP_SYS_TIME"}, entry.HostConfig.CapAdd)
	// the engine folds a dropped capability into the resulting set instead of
	// echoing the request back, so the bounding set is the one worth reading
	assert.Empty(t, entry.HostConfig.CapDrop)
	assert.Len(t, entry.BoundingCaps, 3)
	assert.Equal(t, []string{"CAP_CHOWN", "CAP_DAC_OVERRIDE"}, entry.EffectiveCaps)
	assert.Equal(t, []string{"seccomp=unconfined"}, entry.HostConfig.SecurityOpt)
	assert.True(t, entry.HostConfig.ReadonlyRootfs)
	assert.Equal(t, "postgres", entry.Config.User)
	assert.Equal(t, "always", entry.HostConfig.RestartPolicy.Name)
}

func TestParsePodmanInfo(t *testing.T) {
	info, err := parsePodmanInfo(`{
  "host": {
    "cgroupManager": "systemd",
    "cgroupVersion": "v2",
    "networkBackend": "netavark",
    "ociRuntime": {"name": "crun"},
    "security": {
      "rootless": true,
      "seccompEnabled": true,
      "seccompProfilePath": "/usr/share/containers/seccomp.json",
      "apparmorEnabled": false,
      "selinuxEnabled": true
    }
  },
  "store": {"graphDriverName": "overlay"},
  "version": {"Version": "6.0.1"}
}`)
	require.NoError(t, err)

	assert.True(t, info.Host.Security.Rootless)
	assert.Equal(t, "systemd", info.Host.CgroupManager)
	assert.Equal(t, "v2", info.Host.CgroupVersion)
	assert.Equal(t, "crun", info.Host.OciRuntime.Name)
	assert.Equal(t, "netavark", info.Host.NetworkBackend)
	assert.Equal(t, "overlay", info.Store.GraphDriverName)
	assert.Equal(t, "6.0.1", info.Version.Version)
	assert.True(t, info.Host.Security.SeccompEnabled)
	assert.Equal(t, "/usr/share/containers/seccomp.json", info.Host.Security.SeccompProfilePath)
	assert.False(t, info.Host.Security.ApparmorEnabled)
	assert.True(t, info.Host.Security.SelinuxEnabled)
}

func TestParsePodmanNetworks(t *testing.T) {
	// the network API uses snake_case where the other subcommands use PascalCase
	entries, err := parsePodmanNetworks(`[
  {
    "name": "podman",
    "id": "2f259b",
    "driver": "bridge",
    "network_interface": "podman0",
    "created": "2026-07-21T09:12:33.123456789Z",
    "subnets": [{"subnet": "10.88.0.0/16", "gateway": "10.88.0.1"}],
    "ipv6_enabled": false,
    "internal": false,
    "dns_enabled": true
  }
]`)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.Equal(t, "podman", entry.Name)
	assert.Equal(t, "podman0", entry.NetworkInterface)
	assert.True(t, entry.DNSEnabled)
	assert.False(t, entry.Internal)
	require.Len(t, entry.Subnets, 1)
	assert.Equal(t, "10.88.0.0/16", entry.Subnets[0].Subnet)
	assert.Equal(t, "10.88.0.1", entry.Subnets[0].Gateway)
}

func TestParsePodmanVolumes(t *testing.T) {
	entries, err := parsePodmanVolumes(`[
  {"Name": "data", "Driver": "local", "Mountpoint": "/var/lib/containers/storage/volumes/data/_data",
   "CreatedAt": "2026-07-21T09:12:33.123456789Z", "Labels": {"app": "db"}, "Options": {"type": "tmpfs"},
   "Scope": "local", "Anonymous": false}
]`)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	assert.Equal(t, "data", entries[0].Name)
	assert.Equal(t, "/var/lib/containers/storage/volumes/data/_data", entries[0].Mountpoint)
	assert.Equal(t, map[string]string{"app": "db"}, entries[0].Labels)
	assert.False(t, entries[0].Anonymous)
}

func TestPodmanSplitReference(t *testing.T) {
	tests := []struct {
		title      string
		reference  string
		repository string
		tag        string
	}{
		{"tagged", "docker.io/library/debian:13", "docker.io/library/debian", "13"},
		{"local tagged", "localhost/mql-os-verify:latest", "localhost/mql-os-verify", "latest"},
		{"untagged", "docker.io/library/debian", "docker.io/library/debian", ""},
		{"registry port is not a tag", "registry.local:5000/app", "registry.local:5000/app", ""},
		{"registry port with tag", "registry.local:5000/app:1.2", "registry.local:5000/app", "1.2"},
		{"digest pinned", "docker.io/library/debian@sha256:abc123", "docker.io/library/debian@sha256:abc123", ""},
		{"empty", "", "", ""},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			repository, tag := podmanSplitReference(test.reference)
			assert.Equal(t, test.repository, repository)
			assert.Equal(t, test.tag, tag)
		})
	}
}

func TestPodmanPrimaryName(t *testing.T) {
	assert.Equal(t, "web", podmanPrimaryName([]string{"web", "web-alias"}))
	assert.Empty(t, podmanPrimaryName(nil))
	assert.Empty(t, podmanPrimaryName([]string{}))
}

func TestPodmanUnixTime(t *testing.T) {
	got := podmanUnixTime(1784678819)
	require.NotNil(t, got)
	assert.Equal(t, int64(1784678819), got.Unix())

	// a container that never ran carries no time, and the engine's sentinel for
	// that must not surface as a date in 1970
	assert.Nil(t, podmanUnixTime(0))
	assert.Nil(t, podmanUnixTime(-62135596800))
}

func TestPodmanParseTime(t *testing.T) {
	got := podmanParseTime("2026-07-21T09:12:33.123456789Z")
	require.NotNil(t, got)
	assert.Equal(t, 2026, got.Year())
	assert.Equal(t, time.July, got.Month())

	assert.Nil(t, podmanParseTime(""))
	assert.Nil(t, podmanParseTime("not a time"))
}
