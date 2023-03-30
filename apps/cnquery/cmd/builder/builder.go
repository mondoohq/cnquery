package builder

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder/common"
	"go.mondoo.com/cnquery/motor/providers"
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

type (
	runFn func(cmd *cobra.Command, args []string, provider providers.ProviderType, assetType AssetType)
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
				opts.Run(cmd, args, providers.ProviderType_UNKNOWN, DefaultAssetType)
				return
			}

			log.Info().Msg("no provider specified, defaulting to local.\n  Use --help for a list of available providers.")
			opts.Run(cmd, args, providers.ProviderType_LOCAL_OS, DefaultAssetType)
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
	Run               runFn
	CommonFlags       common.CommonFlagsFn
	CommonPreRun      common.CommonPreRunFn
	Docs              common.CommandsDocs
	PreRun            func(cmd *cobra.Command, args []string)
	ValidArgsFunction func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
}

func buildCmd(baseCmd *cobra.Command, commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) {
	// REMEMBER: when adding new commands, to also add entries in the slimbuilder package so that
	// the mondoo wrapper can also pick up the new subcommands

	containerCmd := containerProviderCmd(commonCmdFlags, preRun, runFn, docs)
	containerImageCmd := containerImageProviderCmd(commonCmdFlags, preRun, runFn, docs)
	containerCmd.AddCommand(containerImageCmd)
	containerRegistryCmd := containerRegistryProviderCmd(commonCmdFlags, preRun, runFn, docs)
	containerCmd.AddCommand(containerRegistryCmd)
	containerTarCmd := containerTarProviderCmd(commonCmdFlags, preRun, runFn, docs)
	containerCmd.AddCommand(containerTarCmd)

	dockerCmd := dockerProviderCmd(commonCmdFlags, preRun, runFn, docs)
	dockerImageCmd := dockerImageProviderCmd(commonCmdFlags, preRun, runFn, docs)
	dockerCmd.AddCommand(dockerImageCmd)
	dockerContainerCmd := dockerContainerProviderCmd(commonCmdFlags, preRun, runFn, docs)
	dockerCmd.AddCommand(dockerContainerCmd)

	// aws subcommand
	awsCmd := awsProviderCmd(commonCmdFlags, preRun, runFn, docs)
	awsEc2 := awsEc2ProviderCmd(commonCmdFlags, preRun, runFn, docs)
	awsCmd.AddCommand(awsEc2)

	awsEc2Connect := awsEc2ConnectProviderCmd(commonCmdFlags, preRun, runFn, docs)
	awsEc2.AddCommand(awsEc2Connect)

	awsEc2EbsCmd := awsEc2EbsProviderCmd(commonCmdFlags, preRun, runFn, docs)
	awsEc2EbsVolumeCmd := awsEc2EbsVolumeProviderCmd(commonCmdFlags, preRun, runFn, docs)
	awsEc2EbsCmd.AddCommand(awsEc2EbsVolumeCmd)
	awsEc2EbsSnapshotCmd := awsEc2EbsSnapshotProviderCmd(commonCmdFlags, preRun, runFn, docs)
	awsEc2EbsCmd.AddCommand(awsEc2EbsSnapshotCmd)
	awsEc2.AddCommand(awsEc2EbsCmd)

	awsEc2Ssm := awsEc2SsmProviderCmd(commonCmdFlags, preRun, runFn, docs)
	awsEc2.AddCommand(awsEc2Ssm)

	// gcp subcommand
	gcpCmd := scanGcpCmd(commonCmdFlags, preRun, runFn, docs)
	gcpGcrCmd := scanGcpGcrCmd(commonCmdFlags, preRun, runFn, docs)
	gcpCmd.AddCommand(gcpGcrCmd)
	gcpCmd.AddCommand(scanGcpOrgCmd(commonCmdFlags, preRun, runFn, docs))
	gcpCmd.AddCommand(scanGcpProjectCmd(commonCmdFlags, preRun, runFn, docs))
	gcpCmd.AddCommand(scanGcpFolderCmd(commonCmdFlags, preRun, runFn, docs))

	// vsphere subcommand
	vsphereCmd := vsphereProviderCmd(commonCmdFlags, preRun, runFn, docs)
	vsphereVmCmd := vsphereVmProviderCmd(commonCmdFlags, preRun, runFn, docs)
	vsphereCmd.AddCommand(vsphereVmCmd)

	// github subcommand
	githubCmd := scanGithubCmd(commonCmdFlags, preRun, runFn, docs)
	githubOrgCmd := githubProviderOrganizationCmd(commonCmdFlags, preRun, runFn, docs)
	githubCmd.AddCommand(githubOrgCmd)
	githubRepositoryCmd := githubProviderRepositoryCmd(commonCmdFlags, preRun, runFn, docs)
	githubCmd.AddCommand(githubRepositoryCmd)
	githubUserCmd := githubProviderUserCmd(commonCmdFlags, preRun, runFn, docs)
	githubCmd.AddCommand(githubUserCmd)

	// terraform subcommand
	terraformCmd := terraformProviderCmd(commonCmdFlags, preRun, runFn, docs)
	terraformPlanCmd := terraformProviderPlanCmd(commonCmdFlags, preRun, runFn, docs)
	terraformCmd.AddCommand(terraformPlanCmd)
	terraformStateCmd := terraformProviderStateCmd(commonCmdFlags, preRun, runFn, docs)
	terraformCmd.AddCommand(terraformStateCmd)

	// subcommands
	baseCmd.AddCommand(localProviderCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(mockProviderCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(vagrantCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(terraformCmd)
	baseCmd.AddCommand(sshProviderCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(winrmProviderCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(containerCmd)
	baseCmd.AddCommand(dockerCmd)
	baseCmd.AddCommand(kubernetesProviderCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(awsCmd)
	baseCmd.AddCommand(azureProviderCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(gcpCmd)
	baseCmd.AddCommand(vsphereCmd)
	baseCmd.AddCommand(githubCmd)
	baseCmd.AddCommand(gitlabProviderCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(ms365ProviderCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(hostProviderCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(aristaProviderCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(scanOktaCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(scanGoogleWorkspaceCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(scanSlackCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(scanVcdCmd(commonCmdFlags, preRun, runFn, docs))
}

func localProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_LOCAL_OS, DefaultAssetType)
	}
	cmd := common.LocalProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func mockProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_MOCK, DefaultAssetType)
	}
	cmd := common.MockProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func vagrantCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_VAGRANT, DefaultAssetType)
	}
	cmd := common.VagrantCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func terraformProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_TERRAFORM, TerraformHclAssetType)
	}
	cmd := common.TerraformProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func terraformProviderStateCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_TERRAFORM, TerraformStateAssetType)
	}
	cmd := common.TerraformProviderStateCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func terraformProviderPlanCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_TERRAFORM, TerraformPlanAssetType)
	}
	cmd := common.TerraformProviderPlanCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func sshProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_SSH, DefaultAssetType)
	}
	cmd := common.SshProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func winrmProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_WINRM, DefaultAssetType)
	}
	cmd := common.WinrmProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func containerProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_DOCKER, DefaultAssetType)
	}
	cmd := common.ContainerProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func containerImageProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_DOCKER_ENGINE_IMAGE, DefaultAssetType)
	}

	cmd := common.ContainerImageProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func containerTarProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_TAR, DefaultAssetType)
	}

	cmd := common.ContainerTarProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func containerRegistryProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_CONTAINER_REGISTRY, DefaultAssetType)
	}
	cmd := common.ContainerRegistryProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func dockerProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_DOCKER, DefaultAssetType)
	}

	cmd := common.DockerProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func dockerContainerProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_DOCKER_ENGINE_CONTAINER, DefaultAssetType)
	}

	cmd := common.DockerContainerProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func dockerImageProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_DOCKER_ENGINE_IMAGE, DefaultAssetType)
	}

	cmd := common.DockerImageProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func kubernetesProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_K8S, DefaultAssetType)
	}

	cmd := common.KubernetesProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func awsProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_AWS, DefaultAssetType)
	}

	cmd := common.AwsProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func awsEc2ProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := common.AwsEc2ProviderCmd(commonCmdFlags, preRun, nil, docs)
	return cmd
}

func awsEc2ConnectProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_SSH, Ec2InstanceConnectAssetType)
	}

	cmd := common.AwsEc2ConnectProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func awsEc2EbsProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_AWS_EC2_EBS, Ec2ebsInstanceAssetType)
	}
	cmd := common.AwsEc2EbsProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func awsEc2EbsVolumeProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_AWS_EC2_EBS, Ec2ebsVolumeAssetType)
	}

	cmd := common.AwsEc2EbsVolumeProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func awsEc2EbsSnapshotProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_AWS_EC2_EBS, Ec2ebsSnapshotAssetType)
	}
	cmd := common.AwsEc2EbsSnapshotProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func awsEc2SsmProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_AWS_SSM_RUN_COMMAND, DefaultAssetType)
	}
	cmd := common.AwsEc2SsmProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func azureProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_AZURE, DefaultAssetType)
	}
	cmd := common.AzureProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)

	return cmd
}

func scanGcpCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_GCP, DefaultAssetType)
	}
	cmd := common.ScanGcpCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func scanGcpOrgCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_GCP, GcpOrganizationAssetType)
	}
	cmd := common.ScanGcpOrgCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func scanGcpProjectCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_GCP, GcpProjectAssetType)
	}

	cmd := common.ScanGcpProjectCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func scanGcpFolderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_GCP, GcpFolderAssetType)
	}
	cmd := common.ScanGcpFolderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func scanGcpGcrCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_CONTAINER_REGISTRY, GcrContainerRegistryAssetType)
	}
	cmd := common.ScanGcpGcrCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func vsphereProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_VSPHERE, DefaultAssetType)
	}
	cmd := common.VsphereProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func vsphereVmProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_VSPHERE_VM, DefaultAssetType)
	}
	cmd := common.VsphereVmProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func scanGithubCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := common.ScanGithubCmd(commonCmdFlags, preRun, nil, docs)
	return cmd
}

func githubProviderOrganizationCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_GITHUB, GithubOrganizationAssetType)
	}
	cmd := common.GithubProviderOrganizationCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func githubProviderRepositoryCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_GITHUB, GithubRepositoryAssetType)
	}
	cmd := common.GithubProviderRepositoryCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func githubProviderUserCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_GITHUB, GithubUserAssetType)
	}
	cmd := common.GithubProviderUserCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func gitlabProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_GITLAB, DefaultAssetType) // TODO: does not indicate individual assets
	}
	cmd := common.GitlabProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func ms365ProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_MS365, DefaultAssetType)
	}
	cmd := common.Ms365ProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func hostProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_HOST, DefaultAssetType)
	}
	cmd := common.HostProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func aristaProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_ARISTAEOS, DefaultAssetType)
	}
	cmd := common.AristaProviderCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func scanOktaCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_OKTA, DefaultAssetType)
	}
	cmd := common.ScanOktaCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func scanGoogleWorkspaceCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_GOOGLE_WORKSPACE, DefaultAssetType)
	}
	cmd := common.ScanGoogleWorkspaceCmd(commonCmdFlags, preRun, wrapRunFn, docs)

	return cmd
}

func scanSlackCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_SLACK, DefaultAssetType)
	}
	cmd := common.ScanSlackCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}

func scanVcdCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	wrapRunFn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, providers.ProviderType_VCD, DefaultAssetType)
	}
	cmd := common.ScanVcdCmd(commonCmdFlags, preRun, wrapRunFn, docs)
	return cmd
}
