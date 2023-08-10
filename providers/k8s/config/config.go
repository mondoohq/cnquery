package config

import "go.mondoo.com/cnquery/providers-sdk/v1/plugin"

var Config = plugin.Provider{
	Name:    "k8s",
	ID:      "go.mondoo.com/cnquery/providers/k8s",
	Version: "9.0.0",
	Connectors: []plugin.Connector{
		{
			Name:      "k8s",
			Aliases:   []string{"kubernetes"},
			Use:       "k8s (optional MANIFEST path)",
			Short:     "a Kubernetes cluster or local manifest file(s).",
			MinArgs:   0,
			MaxArgs:   1,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "context",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Target a Kubernetes context.",
				},
				{
					Long:    "namespaces-exclude",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Filter out Kubernetes objects in the matching namespaces.",
				},
				{
					Long:    "namespaces",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Only include Kubernetes object in the matching namespaces.",
				},
			},
		},
	},
}

// cmd := &cobra.Command{
// 	Use:     "k8s (optional MANIFEST path)",
// 	Aliases: []string{"kubernetes"},
// 	Short:   docs.GetShort("kubernetes"),
// 	Long:    docs.GetLong("kubernetes"),
// 	Args:    cobra.MaximumNArgs(1),
// 	PreRun: func(cmd *cobra.Command, args []string) {
// 		preRun(cmd, args)
// 		viper.BindPFlag("namespaces-exclude", cmd.Flags().Lookup("namespaces-exclude"))
// 		viper.BindPFlag("namespaces", cmd.Flags().Lookup("namespaces"))
// 		viper.BindPFlag("context", cmd.Flags().Lookup("context"))
// 	},
// 	Run: func(cmd *cobra.Command, args []string) {
// 		if len(args) > 0 {
// 			cmd.Flags().Set("path", args[0])
// 		}
// 		runFn(cmd, args)
// 	},
// }
// commonCmdFlags(cmd)

// cmd.Flags().String("context", "", "Target a Kubernetes context.")
// cmd.Flags().String("namespaces-exclude", "", "Filter out Kubernetes objects in the matching namespaces.")
// cmd.Flags().String("namespaces", "", "Only include Kubernetes object in the matching namespaces.")
