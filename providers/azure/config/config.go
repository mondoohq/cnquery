// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import "go.mondoo.com/cnquery/providers-sdk/v1/plugin"

var Config = plugin.Provider{
	Name:    "azure",
	ID:      "go.mondoo.com/cnquery/providers/azure",
	Version: "9.0.0",
	Connectors: []plugin.Connector{
		{
			Name:      "azure",
			Use:       "azure",
			Short:     "azure",
			MinArgs:   0,
			MaxArgs:   8,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "tenant-id",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Directory (tenant) ID of the service principal.",
					Option:  plugin.FlagOption_Hidden,
				},
				{
					Long:    "client-id",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Application (client) ID of the service principal.",
					Option:  plugin.FlagOption_Hidden,
				},
				{
					Long:    "client-secret",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Secret for application.",
					Option:  plugin.FlagOption_Hidden,
				},
				{
					Long:    "certificate-path",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Path (in PKCS #12/PFX or PEM format) to the authentication certificate.",
					Option:  plugin.FlagOption_Hidden,
				},
				{
					Long:    "certificate-secret",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Passphrase for the authentication certificate file.",
					Option:  plugin.FlagOption_Hidden,
				},
				{
					Long:    "subscription",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "ID of the Azure subscription to scan.",
					Option:  plugin.FlagOption_Hidden,
				},
				{
					Long:    "subscriptions",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Comma-separated list of Azure subscriptions to include.",
					Option:  plugin.FlagOption_Hidden,
				},
				{
					Long:    "subscriptions-exclude",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Comma-separated list of Azure subscriptions to exclude.",
					Option:  plugin.FlagOption_Hidden,
				},
			},
		},
	},
}

// func Ms365ProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:     "ms365",
// 		Aliases: []string{"microsoft365"},
// 		Short:   docs.GetShort("ms365"),
// 		Long:    docs.GetLong("ms365"),
// 		Args:    cobra.ExactArgs(0),
// 		PreRun:  preRun,
// 		Run:     runFn,
// 	}
// 	commonCmdFlags(cmd)
// 	cmd.Flags().String("tenant-id", "", "directory (tenant) ID of the service principal")
// 	cmd.MarkFlagRequired("tenant-id")
// 	cmd.Flags().String("client-id", "", "application (client) ID of the service principal")
// 	cmd.MarkFlagRequired("client-id")
// 	cmd.Flags().String("client-secret", "", "secret for application")
// 	cmd.Flags().String("certificate-path", "", "Path (in PKCS #12/PFX or PEM format) to the authentication certificate")
// 	cmd.Flags().String("certificate-secret", "", "passphrase for certificate file")
// 	cmd.Flags().String("datareport", "", "set the MS365 datareport for the scan")
// 	return cmd
// }
