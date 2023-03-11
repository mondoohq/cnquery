package builder

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder/common"
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
	containerCmd := common.ContainerProviderCmd(commonCmdFlags, preRun, runFn, docs)
	containerImageCmd := common.ContainerImageProviderCmd(commonCmdFlags, preRun, runFn, docs)
	containerCmd.AddCommand(containerImageCmd)
	containerRegistryCmd := common.ContainerRegistryProviderCmd(commonCmdFlags, preRun, runFn, docs)
	containerCmd.AddCommand(containerRegistryCmd)

	dockerCmd := common.DockerProviderCmd(commonCmdFlags, preRun, runFn, docs)
	dockerImageCmd := common.DockerImageProviderCmd(commonCmdFlags, preRun, runFn, docs)
	dockerCmd.AddCommand(dockerImageCmd)
	dockerContainerCmd := common.DockerContainerProviderCmd(commonCmdFlags, preRun, runFn, docs)
	dockerCmd.AddCommand(dockerContainerCmd)

	// aws subcommand
	awsCmd := common.AwsProviderCmd(commonCmdFlags, preRun, runFn, docs)
	awsEc2 := common.AwsEc2ProviderCmd(commonCmdFlags, preRun, runFn, docs)
	awsCmd.AddCommand(awsEc2)

	awsEc2Connect := common.AwsEc2ConnectProviderCmd(commonCmdFlags, preRun, runFn, docs)
	awsEc2.AddCommand(awsEc2Connect)

	awsEc2EbsCmd := common.AwsEc2EbsProviderCmd(commonCmdFlags, preRun, runFn, docs)
	awsEc2EbsVolumeCmd := common.AwsEc2EbsVolumeProviderCmd(commonCmdFlags, preRun, runFn, docs)
	awsEc2EbsCmd.AddCommand(awsEc2EbsVolumeCmd)
	awsEc2EbsSnapshotCmd := common.AwsEc2EbsSnapshotProviderCmd(commonCmdFlags, preRun, runFn, docs)
	awsEc2EbsCmd.AddCommand(awsEc2EbsSnapshotCmd)
	awsEc2.AddCommand(awsEc2EbsCmd)

	awsEc2Ssm := common.AwsEc2SsmProviderCmd(commonCmdFlags, preRun, runFn, docs)
	awsEc2.AddCommand(awsEc2Ssm)

	// subcommands
	baseCmd.AddCommand(containerCmd)
	baseCmd.AddCommand(dockerCmd)
	baseCmd.AddCommand(common.KubernetesProviderCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(awsCmd)
	baseCmd.AddCommand(common.AzureProviderCmd(commonCmdFlags, preRun, runFn, docs))
}
