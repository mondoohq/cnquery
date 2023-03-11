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
	containerCmd := containerProviderCmd(commonCmdFlags, preRun, runFn, docs)
	containerImageCmd := containerImageProviderCmd(commonCmdFlags, preRun, runFn, docs)
	containerCmd.AddCommand(containerImageCmd)
	containerRegistryCmd := containerRegistryProviderCmd(commonCmdFlags, preRun, runFn, docs)
	containerCmd.AddCommand(containerRegistryCmd)

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
	cmd := &cobra.Command{
		Use:    "local",
		Short:  docs.GetShort("local"),
		Long:   docs.GetLong("local"),
		Args:   cobra.ExactArgs(0),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_LOCAL_OS, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func mockProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "mock PATH",
		Short:  docs.GetShort("mock"),
		Long:   docs.GetLong("mock"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Flags().Set("path", args[0])
			runFn(cmd, args, providers.ProviderType_MOCK, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func vagrantCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "vagrant HOST",
		Short:  docs.GetShort("vagrant"),
		Long:   docs.GetLong("vagrant"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_VAGRANT, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func terraformProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "terraform PATH",
		Short:  docs.GetShort("terraform"),
		Long:   docs.GetLong("terraform"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Flags().Set("path", args[0])
			runFn(cmd, args, providers.ProviderType_TERRAFORM, TerraformHclAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func terraformProviderStateCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "state PATH",
		Short:  "Scan a Terraform state file (json)",
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Flags().Set("path", args[0])
			runFn(cmd, args, providers.ProviderType_TERRAFORM, TerraformStateAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func terraformProviderPlanCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "plan PATH",
		Short:  "Scan a Terraform plan file (json)",
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Flags().Set("path", args[0])
			runFn(cmd, args, providers.ProviderType_TERRAFORM, TerraformPlanAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func sshProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "ssh user@host",
		Short:  docs.GetShort("ssh"),
		Long:   docs.GetLong("ssh"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_SSH, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func winrmProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "winrm user@host",
		Short:  docs.GetShort("winrm"),
		Long:   docs.GetLong("winrm"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_WINRM, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
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
	cmd := &cobra.Command{
		Use:   "gcp",
		Short: docs.GetShort("gcp"),
		Long:  docs.GetLong("gcp"),
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
			viper.BindPFlag("project", cmd.Flags().Lookup("project"))
			viper.BindPFlag("organization", cmd.Flags().Lookup("organization"))
			viper.BindPFlag("project-id", cmd.Flags().Lookup("project-id"))
			viper.BindPFlag("organization-id", cmd.Flags().Lookup("organization-id"))
			viper.BindPFlag("credentials-path", cmd.Flags().Lookup("credentials-path"))
		},
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_GCP, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("project", "", "specify the GCP project to scan")
	cmd.Flags().MarkHidden("project")
	cmd.Flags().MarkDeprecated("project", "--project is deprecated in favor of --project-id")
	cmd.Flags().String("project-id", "", "specify the GCP project ID to scan")
	cmd.Flags().String("organization", "", "specify the GCP organization to scan")
	cmd.Flags().MarkHidden("organization")
	cmd.Flags().MarkDeprecated("organization", "--organization is deprecated in favor of --organization-id")
	cmd.Flags().String("organization-id", "", "specify the GCP organization ID to scan")
	cmd.Flags().String("credentials-path", "", "The path to the service account credentials to access the APIs with")
	return cmd
}

func scanGcpOrgCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "org ORGANIZATION-ID",
		Aliases: []string{"organization"},
		Short:   docs.GetShort("gcp-org"),
		Long:    docs.GetLong("gcp-org"),
		Args:    cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
			viper.BindPFlag("credentials-path", cmd.Flags().Lookup("credentials-path"))
		},
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_GCP, GcpOrganizationAssetType)
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("credentials-path", "", "The path to the service account credentials to access the APIs with")
	return cmd
}

func scanGcpProjectCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project PROJECT-ID",
		Short: docs.GetShort("gcp-project"),
		Long:  docs.GetLong("gcp-project"),
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
			viper.BindPFlag("credentials-path", cmd.Flags().Lookup("credentials-path"))
		},
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_GCP, GcpProjectAssetType)
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("credentials-path", "", "The path to the service account credentials to access the APIs with")
	return cmd
}

func scanGcpFolderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "folder FOLDER-ID",
		Short: docs.GetShort("gcp-folder"),
		Long:  docs.GetLong("gcp-folder"),
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
			viper.BindPFlag("credentials-path", cmd.Flags().Lookup("credentials-path"))
		},
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_GCP, GcpFolderAssetType)
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("credentials-path", "", "The path to the service account credentials to access the APIs with")
	return cmd
}

func scanGcpGcrCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gcr PROJECT",
		Short: docs.GetShort("gcp-gcr"),
		Long:  docs.GetLong("gcp-gcr"),
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("repository", cmd.Flags().Lookup("repository"))
			preRun(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_CONTAINER_REGISTRY, GcrContainerRegistryAssetType)
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("repository", "", "specify the GCR repository to scan")
	return cmd
}

func vsphereProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "vsphere user@host",
		Short:  docs.GetShort("vsphere"),
		Long:   docs.GetLong("vsphere"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_VSPHERE, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func vsphereVmProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "vm user@host",
		Short:  docs.GetShort("vsphere-vm"),
		Long:   docs.GetLong("vsphere-vm"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_VSPHERE_VM, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func scanGithubCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "github SUBCOMMAND",
		Short: docs.GetShort("github"),
		Long:  docs.GetLong("github"),
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			os.Exit(0)
		},
	}
	return cmd
}

func githubProviderOrganizationCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: docs.GetShort("github-org"),
		Long:  docs.GetLong("github-org"),
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("token", cmd.Flags().Lookup("token"))
			preRun(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_GITHUB, GithubOrganizationAssetType)
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("token", "", "GitHub personal access tokens")
	return cmd
}

func githubProviderRepositoryCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: docs.GetShort("github-repo"),
		Long:  docs.GetLong("github-repo"),
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("token", cmd.Flags().Lookup("token"))
			preRun(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_GITHUB, GithubRepositoryAssetType)
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("token", "", "GitHub personal access token")
	return cmd
}

func githubProviderUserCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: docs.GetShort("github-user"),
		Long:  docs.GetLong("github-user"),
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("token", cmd.Flags().Lookup("token"))
			preRun(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_GITHUB, GithubUserAssetType)
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("token", "", "GitHub personal access token")
	return cmd
}

func gitlabProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gitlab",
		Short: docs.GetShort("gitlab"),
		Long:  docs.GetLong("gitlab"),
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("token", cmd.Flags().Lookup("token"))
			viper.BindPFlag("group", cmd.Flags().Lookup("group"))
			preRun(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_GITLAB, DefaultAssetType) // TODO: does not indicate individual assets
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("group", "", "a GitLab group to scan")
	cmd.MarkFlagRequired("group")
	cmd.Flags().String("token", "", "GitLab personal access token")
	return cmd
}

func ms365ProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ms365",
		Aliases: []string{"microsoft365"},
		Short:   docs.GetShort("ms365"),
		Long:    docs.GetLong("ms365"),
		Args:    cobra.ExactArgs(0),
		PreRun:  preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_MS365, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("tenant-id", "", "directory (tenant) ID of the service principal")
	cmd.Flags().String("client-id", "", "application (client) ID of the service principal")
	cmd.Flags().String("client-secret", "", "secret for application")
	cmd.Flags().String("certificate-path", "", "path to certificate that's used for certificate-based authentication in PKCS 12 format (pfx)")
	cmd.Flags().String("certificate-secret", "", "passphrase for certificate file")
	cmd.Flags().String("datareport", "", "set the MS365 datareport for the scan")
	return cmd
}

func hostProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "host HOST",
		Short:  docs.GetShort("host"),
		Long:   docs.GetLong("host"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_HOST, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func aristaProviderCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "arista user@host",
		Short:  docs.GetShort("arista"),
		Long:   docs.GetLong("arista"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_ARISTAEOS, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func scanOktaCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "okta",
		Short: docs.GetShort("okta"),
		Long:  docs.GetLong("okta"),
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("organization", cmd.Flags().Lookup("organization"))
			viper.BindPFlag("token", cmd.Flags().Lookup("token"))
			preRun(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_OKTA, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("organization", "", "specify the Okta organization to scan")
	cmd.Flags().String("token", "", "Okta access token")
	return cmd
}

func scanGoogleWorkspaceCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "google-workspace",
		Aliases: []string{"googleworkspace"},
		Short:   docs.GetShort("googleworkspace"),
		Long:    docs.GetLong("googleworkspace"),
		Args:    cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("customer-id", cmd.Flags().Lookup("customer-id"))
			viper.BindPFlag("impersonated-user-email", cmd.Flags().Lookup("impersonated-user-email"))
			viper.BindPFlag("credentials-path", cmd.Flags().Lookup("credentials-path"))

			preRun(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_GOOGLE_WORKSPACE, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("customer-id", "", "Specify the Google Workspace customer id to scan")
	cmd.Flags().String("impersonated-user-email", "", "The impersonated user's email with access to the Admin APIs")
	cmd.Flags().String("credentials-path", "", "The path to the service account credentials to access the APIs with")

	return cmd
}

func scanSlackCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slack",
		Short: docs.GetShort("slack"),
		Long:  docs.GetLong("slack"),
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("token", cmd.Flags().Lookup("token"))
			preRun(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_SLACK, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("token", "", "Slack API token")
	return cmd
}

func scanVcdCmd(commonCmdFlags common.CommonFlagsFn, preRun common.CommonPreRunFn, runFn runFn, docs common.CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vcd",
		Short: docs.GetShort("vcd"),
		Long:  docs.GetLong("vcd"),
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("user", cmd.Flags().Lookup("user"))
			viper.BindPFlag("host", cmd.Flags().Lookup("host"))
			viper.BindPFlag("organization", cmd.Flags().Lookup("organization"))
			preRun(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_VCD, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("user", "", "vCloud Director user")
	cmd.Flags().String("host", "", "vCloud Director Host")
	cmd.Flags().String("organization", "", "vCloud Director Organization (optional)")
	return cmd
}
