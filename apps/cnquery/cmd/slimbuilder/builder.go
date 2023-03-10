package builder

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder/common"
)

type AssetType int64

const (
	UnknownAssetType AssetType = iota
	DefaultAssetType
	TerraformHclAssetType
	TerraformStateAssetType
	TerraformPlanAssetType
	Ec2InstanceConnectAssetType
	Ec2ebsInstanceAssetType
	Ec2ebsVolumeAssetType
	Ec2ebsSnapshotAssetType
	GcpOrganizationAssetType
	GcpProjectAssetType
	GcpFolderAssetType
	GcrContainerRegistryAssetType
	GithubOrganizationAssetType
	GithubRepositoryAssetType
	GithubUserAssetType
)

func NewSlimProviderCommand(opts CommandOpts) *cobra.Command {
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
				opts.Run(cmd, args)
				return
			}

			log.Info().Msg("no provider specified, defaulting to local.\n  Use --help for a list of available providers.")
			opts.Run(cmd, args)
		},
	}
	opts.CommonFlags(cmd)
	buildCmd(cmd, opts.CommonFlags, opts.CommonPreRun, opts.Run, opts.Docs)
	return cmd
}

// CommandOpts is a helper command to create a cobra.Command
type CommandOpts struct {
	Use               string
	Aliases           []string
	Short             string
	Long              string
	Run               common.RunFn
	CommonFlags       common.CommonFlagsFn
	CommonPreRun      common.CommonPreRunFn
	Docs              common.CommandsDocs
	PreRun            func(cmd *cobra.Command, args []string)
	ValidArgsFunction func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
}

func buildCmd(baseCmd *cobra.Command, commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn common.RunFn, docs common.CommandsDocs) {
	// subcommands
	baseCmd.AddCommand(azureProviderCmd(commonCmdFlags, preRun, runFn, docs))
}

func azureProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn common.RunFn, docs common.CommandsDocs) *cobra.Command {
	cmd := common.AzureProviderCmd(commonCmdFlags, preRun, runFn, docs)

	return cmd
}
