package builder

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	GcrContainerRegistryAssetType
	GithubOrganizationAssetType
	GithubRepositoryAssetType
)

type (
	commonFlagsFn  func(cmd *cobra.Command)
	commonPreRunFn func(cmd *cobra.Command, args []string)
	runFn          func(cmd *cobra.Command, args []string, provider providers.ProviderType, assetType AssetType)
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
			opts.Run(cmd, args, providers.ProviderType_LOCAL_OS, UnknownAssetType)
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
	CommonFlags       commonFlagsFn
	CommonPreRun      commonPreRunFn
	Docs              CommandsDocs
	PreRun            func(cmd *cobra.Command, args []string)
	ValidArgsFunction func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
}

func buildCmd(baseCmd *cobra.Command, commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) {
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
	amsCmd := awsProviderCmd(commonCmdFlags, preRun, runFn, docs)
	awsEc2 := awsEc2ProviderCmd(commonCmdFlags, preRun, runFn, docs)
	amsCmd.AddCommand(awsEc2)

	awsEc2Connect := awsEc2ConnectProviderCmd(commonCmdFlags, preRun, runFn, docs)
	awsEc2.AddCommand(awsEc2Connect)

	awsEc2EbsCmd := awsEc2EbsProviderCmd(commonCmdFlags, preRun, runFn, docs)
	awsEc2.AddCommand(awsEc2EbsCmd)

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

	// terraform subcommand
	terraformCmd := terraformProviderCmd(commonCmdFlags, preRun, runFn, docs)
	terrafromPlanCmd := terraformProviderPlanCmd(commonCmdFlags, preRun, runFn, docs)
	terraformCmd.AddCommand(terrafromPlanCmd)
	terrafromStateCmd := terraformProviderStateCmd(commonCmdFlags, preRun, runFn, docs)
	terraformCmd.AddCommand(terrafromStateCmd)

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
	baseCmd.AddCommand(amsCmd)
	baseCmd.AddCommand(azureProviderCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(gcpCmd)
	baseCmd.AddCommand(vsphereCmd)
	baseCmd.AddCommand(githubCmd)
	baseCmd.AddCommand(gitlabProviderCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(ms365ProviderCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(hostProviderCmd(commonCmdFlags, preRun, runFn, docs))
	baseCmd.AddCommand(aristaProviderCmd(commonCmdFlags, preRun, runFn, docs))
}

type CommandsDocs struct {
	Entries map[string]CommandDocsEntry
}

type CommandDocsEntry struct {
	Short string
	Long  string
}

func (c CommandsDocs) GetShort(id string) string {
	e, ok := c.Entries[id]
	if ok {
		return e.Short
	}
	return ""
}

func (c CommandsDocs) GetLong(id string) string {
	e, ok := c.Entries[id]
	if ok {
		return e.Long
	}
	return ""
}

func localProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
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

func mockProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
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

func vagrantCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
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

func terraformProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
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

func terraformProviderStateCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "state PATH",
		Short:  "Scan all Terraform state file (json)",
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

func terraformProviderPlanCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "plan PATH",
		Short:  "Scan all Terraform plan file (json)",
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

func sshProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
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

func winrmProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
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

func containerProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "container ID",
		Short:  docs.GetShort("container"),
		Long:   docs.GetLong("container"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_DOCKER, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func containerImageProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "image ID",
		Short:  docs.GetShort("container-image"),
		Long:   docs.GetLong("container-image"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_DOCKER_ENGINE_IMAGE, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func containerRegistryProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Aliases: []string{"cr"},
		Use:     "registry TARGET",
		Short:   docs.GetShort("container-registry"),
		Long:    docs.GetLong("container-registry"),
		Args:    cobra.ExactArgs(1),
		PreRun:  preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_CONTAINER_REGISTRY, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func dockerProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "docker ID",
		Short:  docs.GetShort("docker"),
		Long:   docs.GetLong("docker"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_DOCKER, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func dockerContainerProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "container ID",
		Short:  docs.GetShort("docker-container"),
		Long:   docs.GetLong("docker-container"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_DOCKER_ENGINE_CONTAINER, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func dockerImageProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "image ID",
		Short:  docs.GetShort("docker-image"),
		Long:   docs.GetLong("docker-image"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_DOCKER_ENGINE_IMAGE, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func kubernetesProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "k8s (optional MANIFEST path)",
		Aliases: []string{"kubernetes"},
		Short:   docs.GetShort("kubernetes"),
		Long:    docs.GetLong("kubernetes"),
		Args:    cobra.MaximumNArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
			viper.BindPFlag("namespace", cmd.Flags().Lookup("namespace"))
			viper.BindPFlag("context", cmd.Flags().Lookup("context"))
		},
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				cmd.Flags().Set("path", args[0])
			}
			runFn(cmd, args, providers.ProviderType_K8S, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().Bool("all-namespaces", false, "DEPRECATED: list the resources across all namespaces.")
	cmd.Flags().String("namespace", "", "target a kubernetes namespace")
	cmd.Flags().String("context", "", "target a kubernetes context")
	return cmd
}

func awsProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aws",
		Short: docs.GetShort("aws"),
		Long:  docs.GetLong("aws"),
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
			viper.BindPFlag("project", cmd.Flags().Lookup("project"))
			viper.BindPFlag("region", cmd.Flags().Lookup("region"))
		},
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_AWS, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("profile", "", "pick a named AWS profile to use")
	cmd.Flags().String("region", "", "the AWS region to scan")
	return cmd
}

func awsEc2ProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ec2 SUBCOMMAND",
		Short: docs.GetShort("aws-ec2"),
		Long:  docs.GetLong("aws-ec2"),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			os.Exit(0)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func awsEc2ConnectProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "instance-connect user@host",
		Short:  docs.GetShort("aws-ec2-connect"),
		Long:   docs.GetLong("aws-ec2-connect"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_SSH, Ec2InstanceConnectAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func awsEc2EbsProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "ebs INSTANCEID",
		Short:  docs.GetShort("aws-ec2-ebs-instance"),
		Long:   docs.GetLong("aws-ec2-ebs-instance"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_AWS_EC2_EBS, Ec2ebsInstanceAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func awsEc2EbsVolumeProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "volume VOLUMEID",
		Short:  docs.GetShort("aws-ec2-ebs-volume"),
		Long:   docs.GetLong("aws-ec2-ebs-volume"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_AWS_EC2_EBS, Ec2ebsVolumeAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func awsEc2EbsSnapshotProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "snapshot SNAPSHOTID",
		Short:  docs.GetShort("aws-ec2-ebs-snapshot"),
		Long:   docs.GetLong("aws-ec2-ebs-snapshot"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_AWS_EC2_EBS, Ec2ebsSnapshotAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func awsEc2SsmProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "ssm user@host",
		Short:  docs.GetShort("aws-ec2-ssm"),
		Long:   docs.GetLong("aws-ec2-ssm"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_AWS_SSM_RUN_COMMAND, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func azureProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "azure",
		Short: docs.GetShort("azure"),
		Long:  docs.GetLong("azure"),
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
			viper.BindPFlag("subscription", cmd.Flags().Lookup("subscription"))
		},
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_AZURE, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("subscription", "", "the Azure subscription ID to scan")
	return cmd
}

func scanGcpCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gcp",
		Short: docs.GetShort("gcp"),
		Long:  docs.GetLong("gcp"),
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
			viper.BindPFlag("project", cmd.Flags().Lookup("project"))
			viper.BindPFlag("organization", cmd.Flags().Lookup("organization"))
		},
		Run: func(cmd *cobra.Command, args []string) {
			runFn(cmd, args, providers.ProviderType_GCP, DefaultAssetType)
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("project", "", "specify the GCP project to scan")
	cmd.Flags().String("organization", "", "specify the GCP organization to scan")
	return cmd
}

func scanGcpGcrCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
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

func vsphereProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
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

func vsphereVmProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
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

func scanGithubCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
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

func githubProviderOrganizationCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
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
	cmd.Flags().String("token", "", "GitHub access tokens")
	return cmd
}

func githubProviderRepositoryCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
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
	cmd.Flags().String("token", "", "GitHub access tokens")
	return cmd
}

func gitlabProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
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
			runFn(cmd, args, providers.ProviderType_GITHUB, DefaultAssetType) // TODO: does not indicate individual assets
		},
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("group", "", "a GitLab group to scan")
	cmd.MarkFlagRequired("group")
	cmd.Flags().String("token", "", "GitHub access tokens")
	return cmd
}

func ms365ProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
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

func hostProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
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

func aristaProviderCmd(commonCmdFlags commonFlagsFn, preRun commonPreRunFn, runFn runFn, docs CommandsDocs) *cobra.Command {
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
