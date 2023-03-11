package builder

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder2/common"
)

type (
	runFn func(cmd *cobra.Command, args []string)
)

func NewProviderCommand(opts CommandOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:     opts.Use,
		Aliases: opts.Aliases,
		Short:   opts.Short,
		Long:    opts.Long,
		PreRun: func(cmd *cobra.Command, args []string) {
			if opts.PreRun != nil {
				opts.PreRun(cmd, args)
			}
			if opts.CommonPreRun != nil {
				opts.CommonPreRun(cmd, args)
			}
		},
		ValidArgsFunction: opts.ValidArgsFunction,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				log.Error().Msg("provider " + args[0] + " does not exist")
				cmd.Help()
				os.Exit(1)
			}

			if viper.GetString("inventory-file") != "" {
				// when the user provided an inventory file, users do not need to provide a provider
				opts.SubcommandFnMap["inventory-file"](cmd, args)
				return
			}

			log.Info().Msg("no provider specified, defaulting to local.\n  Use --help for a list of available providers.")
			opts.SubcommandFnMap["local"](cmd, args)
		},
	}
	opts.CommonFlags(cmd)
	buildCmd(cmd, opts.CommonFlags, opts.CommonPreRun, opts.Docs, opts.SubcommandFnMap)
	return cmd
}

// CommandOpts is a helper command to create a cobra.Command
type CommandOpts struct {
	Use               string
	Aliases           []string
	Short             string
	Long              string
	CommonFlags       common.CommonFlagsFn
	CommonPreRun      common.CommonPreRunFn
	Docs              common.CommandsDocs
	PreRun            func(cmd *cobra.Command, args []string)
	ValidArgsFunction func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
	SubcommandFnMap   common.SubcommandFnMap
}

func buildCmd(baseCmd *cobra.Command, commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, docs common.CommandsDocs, subcmdFnMap common.SubcommandFnMap) {
	// subcommands
	baseCmd.AddCommand(localProviderCmd(commonCmdFlags, preRun, docs, subcmdFnMap))
	baseCmd.AddCommand(azureProviderCmd(commonCmdFlags, preRun, docs, subcmdFnMap))
}

func localProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, docs common.CommandsDocs, subcmdFnMap common.SubcommandFnMap) *cobra.Command {
	fn := subcmdFnMap["local"]
	cmd := common.LocalProviderCmd(commonCmdFlags, preRun, fn, docs)
	return cmd
}

func azureProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, docs common.CommandsDocs, subcmdFnMap common.SubcommandFnMap) *cobra.Command {
	fn := subcmdFnMap["azure"]
	cmd := common.AzureProviderCmd(commonCmdFlags, preRun, fn, docs)

	return cmd
}
