package config

import "go.mondoo.com/cnquery/providers-sdk/v1/plugin"

var Config = plugin.Provider{
	Name:    "gcp",
	ID:      "go.mondoo.com/cnquery/providers/gcp",
	Version: "9.0.0",
	Connectors: []plugin.Connector{
		{
			Name:    "gcp",
			Use:     "gcp",
			Short:   "a Google Cloud Platform (GCP) organization, project or folder",
			MinArgs: 0,
			MaxArgs: 0,
			// Discovery: []string{
			// 	"containers",
			// 	"container-images",
			// },
			Flags: []plugin.Flag{
				{
					Long: "credentials-path",
					Type: plugin.FlagType_String,
					Desc: "The path to the service account credentials to access the APIs with",
				},
			},
		},
	},
}

// func ScanGcpCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:   "gcp",
// 		Short: docs.GetShort("gcp"),
// 		Long:  docs.GetLong("gcp"),
// 		Args:  cobra.ExactArgs(0),
// 		PreRun: func(cmd *cobra.Command, args []string) {
// 			preRun(cmd, args)
// 			viper.BindPFlag("project-id", cmd.Flags().Lookup("project-id"))
// 			viper.BindPFlag("organization-id", cmd.Flags().Lookup("organization-id"))
// 			viper.BindPFlag("credentials-path", cmd.Flags().Lookup("credentials-path"))
// 		},
// 		Run: runFn,
// 	}
// 	commonCmdFlags(cmd)
// 	cmd.Flags().String("project-id", "", "specify the GCP project ID to scan")
// 	cmd.Flags().MarkHidden("project-id")
// 	cmd.Flags().MarkDeprecated("project-id", "--project-id is deprecated in favor of scan gcp project")
// 	cmd.Flags().String("organization-id", "", "specify the GCP organization ID to scan")
// 	cmd.Flags().MarkHidden("organization-id")
// 	cmd.Flags().MarkDeprecated("organization-id", "--organization-id is deprecated in favor of scan gcp org")
// 	cmd.Flags().String("credentials-path", "", "The path to the service account credentials to access the APIs with")
// 	return cmd
// }
