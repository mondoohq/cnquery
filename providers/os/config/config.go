package config

import "go.mondoo.com/cnquery/providers/plugin"

var Config = plugin.Provider{
	Name: "os",
	Connectors: []plugin.Connector{
		{
			Name:    "local",
			Use:     "local",
			Short:   "your local system",
			MinArgs: 0,
			MaxArgs: 0,
			Discovery: []string{
				"containers",
				"container-images",
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
	},
}
