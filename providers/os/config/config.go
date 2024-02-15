// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/os/resources/discovery/docker_engine"
)

var Config = plugin.Provider{
	Name:    "os",
	ID:      "go.mondoo.com/cnquery/v9/providers/os",
	Version: "10.2.0",
	ConnectionTypes: []string{
		shared.Type_Local.String(),
		shared.Type_SSH.String(),
		shared.Type_Tar.String(),
		shared.Type_DockerSnapshot.String(),
		shared.Type_Vagrant.String(),
		shared.Type_DockerImage.String(),
		shared.Type_DockerContainer.String(),
		shared.Type_DockerRegistry.String(),
		shared.Type_ContainerRegistry.String(),
		shared.Type_RegistryImage.String(),
		shared.Type_FileSystem.String(),
		shared.Type_Winrm.String(),
	},
	Connectors: []plugin.Connector{
		{
			Name:    "local",
			Use:     "local",
			Short:   "your local system",
			MinArgs: 0,
			MaxArgs: 0,
			Discovery: []string{
				docker_engine.DiscoveryContainerRunning,
				docker_engine.DiscoveryContainerImages,
			},
			Flags: []plugin.Flag{
				{
					Long:        "sudo",
					Type:        plugin.FlagType_Bool,
					Default:     "false",
					Desc:        "Elevate privileges with sudo.",
					ConfigEntry: "sudo.active",
				},
				{
					Long:    "id-detector",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "User override for platform ID detection mechanism",
					Option:  plugin.FlagOption_Hidden,
				},
			},
		},
		{
			Name:    "ssh",
			Use:     "ssh user@host",
			Short:   "a remote system via SSH",
			MinArgs: 1,
			MaxArgs: 1,
			Flags: []plugin.Flag{
				{
					Long:    "sudo",
					Type:    plugin.FlagType_Bool,
					Default: "false",
					Desc:    "Elevate privileges with sudo.",
				},
				{
					Long:    "insecure",
					Type:    plugin.FlagType_Bool,
					Default: "false",
					Desc:    "Disable SSH hostkey verification.",
				},
				{
					Long:        "ask-pass",
					Type:        plugin.FlagType_Bool,
					Default:     "false",
					Desc:        "Prompt for connection password.",
					ConfigEntry: "-",
				},
				{
					Long:        "password",
					Short:       "p",
					Type:        plugin.FlagType_String,
					Default:     "",
					Desc:        "Set the connection password for SSH.",
					Option:      plugin.FlagOption_Password,
					ConfigEntry: "-",
				},
				{
					Long:    "identity-file",
					Short:   "i",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Select a file from which to read the identity (private key) for public key authentication.",
				},
				{
					Long:    "id-detector",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "User override for platform ID detection mechanism",
					Option:  plugin.FlagOption_Hidden,
				},
			},
		},
		{
			Name:    "winrm",
			Use:     "winrm user@host",
			Short:   "a remote system via WinRM",
			MinArgs: 1,
			MaxArgs: 1,
			Flags: []plugin.Flag{
				{
					Long:    "insecure",
					Default: "false",
					Desc:    "Disable TLS/SSL checks",
					Type:    plugin.FlagType_Bool,
				},
				{
					Long:    "ask-pass",
					Default: "false",
					Desc:    "Prompt for connection password.",
					Type:    plugin.FlagType_Bool,
				},
				{
					Long:        "password",
					Short:       "p",
					Default:     "false",
					Desc:        "Set the connection password for SSH.",
					Type:        plugin.FlagType_String,
					Option:      plugin.FlagOption_Password,
					ConfigEntry: "-",
				},
				{
					Long:    "id-detector",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "User override for platform ID detection mechanism",
					Option:  plugin.FlagOption_Hidden,
				},
			},
		},
		{
			Name:    "vagrant",
			Use:     "vagrant host",
			Short:   "a Vagrant host",
			MinArgs: 1,
			MaxArgs: 1,
			Flags: []plugin.Flag{
				{
					Long:    "sudo",
					Type:    plugin.FlagType_Bool,
					Default: "false",
					Desc:    "Elevate privileges with sudo.",
				},
				{
					Long:    "id-detector",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "User override for platform ID detection mechanism",
					Option:  plugin.FlagOption_Hidden,
				},
			},
		},
		{
			Name:    "container",
			Use:     "container",
			Short:   "a running container or container image",
			MinArgs: 1,
			MaxArgs: 2,
			Discovery: []string{
				docker_engine.DiscoveryContainerRunning,
				docker_engine.DiscoveryContainerImages,
			},
			Flags: []plugin.Flag{
				{
					Long:        "sudo",
					Type:        plugin.FlagType_Bool,
					Default:     "false",
					Desc:        "Elevate privileges with sudo.",
					ConfigEntry: "sudo.active",
				},
				{
					Long:    "id-detector",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "User override for platform ID detection mechanism",
					Option:  plugin.FlagOption_Hidden,
				},
				{
					Long:    "disable-cache",
					Type:    plugin.FlagType_Bool,
					Default: "false",
					Desc:    "Disable the in-memory cache for images. WARNING: This will slow down scans significantly.",
				},
			},
		},
		{
			Name:    "docker",
			Use:     "docker",
			Short:   "a running Docker container or Docker image",
			MinArgs: 1,
			MaxArgs: 2,
			Discovery: []string{
				docker_engine.DiscoveryContainerRunning,
				docker_engine.DiscoveryContainerImages,
			},
			Flags: []plugin.Flag{
				{
					Long:        "sudo",
					Type:        plugin.FlagType_Bool,
					Default:     "false",
					Desc:        "Elevate privileges with sudo.",
					ConfigEntry: "sudo.active",
				},
				{
					Long:    "id-detector",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "User override for platform ID detection mechanism",
					Option:  plugin.FlagOption_Hidden,
				},
				{
					Long:    "disable-cache",
					Type:    plugin.FlagType_Bool,
					Default: "false",
					Desc:    "Disable the in-memory cache for images. WARNING: This will slow down scans significantly.",
				},
			},
		},
		{
			Name:    "filesystem",
			Aliases: []string{"fs"},
			Use:     "filesystem [flags]",
			Short:   "a mounted file system target",
			MinArgs: 0,
			MaxArgs: 0,
			Flags: []plugin.Flag{
				{
					Long:    "path",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Path to a local file or directory for the connection to use.",
				},
			},
		},
	},
}
