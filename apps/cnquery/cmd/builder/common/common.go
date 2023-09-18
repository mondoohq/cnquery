package common

import (
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	discovery_common "go.mondoo.com/cnquery/motor/discovery/common"
)

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

type (
	CommonFlagsFn  func(cmd *cobra.Command)
	CommonPreRunFn func(cmd *cobra.Command, args []string)
	RunFn          func(cmd *cobra.Command, args []string)
)

func LocalProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "local",
		Short:  docs.GetShort("local"),
		Long:   docs.GetLong("local"),
		Args:   cobra.ExactArgs(0),
		PreRun: preRun,
		Run:    runFn,
	}
	commonCmdFlags(cmd)
	return cmd
}

func MockProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "mock PATH",
		Short:  docs.GetShort("mock"),
		Long:   docs.GetLong("mock"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Flags().Set("path", args[0])
			runFn(cmd, args)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func VagrantCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "vagrant HOST",
		Short:  docs.GetShort("vagrant"),
		Long:   docs.GetLong("vagrant"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run:    runFn,
	}
	commonCmdFlags(cmd)
	return cmd
}

func TerraformProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "terraform PATH",
		Short:  docs.GetShort("terraform"),
		Long:   docs.GetLong("terraform"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Flags().Set("path", args[0])
			runFn(cmd, args)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func TerraformProviderStateCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "state PATH",
		Short:  "Scan a Terraform state file (json)",
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Flags().Set("path", args[0])
			runFn(cmd, args)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func TerraformProviderPlanCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "plan PATH",
		Short:  "Scan a Terraform plan file (json)",
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Flags().Set("path", args[0])
			runFn(cmd, args)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func SshProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "ssh user@host",
		Short:  docs.GetShort("ssh"),
		Long:   docs.GetLong("ssh"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run:    runFn,
	}
	commonCmdFlags(cmd)
	return cmd
}

func WinrmProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "winrm user@host",
		Short:  docs.GetShort("winrm"),
		Long:   docs.GetLong("winrm"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run:    runFn,
	}
	commonCmdFlags(cmd)
	return cmd
}

func ContainerProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "container ID",
		Short:  docs.GetShort("container"),
		Long:   docs.GetLong("container"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run:    runFn,
	}
	commonCmdFlags(cmd)
	return cmd
}

func ContainerImageProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image ID",
		Short: docs.GetShort("container-image"),
		Long:  docs.GetLong("container-image"),
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
			viper.BindPFlag("disable-cache", cmd.Flags().Lookup("disable-cache"))
		},
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().Bool("disable-cache", false, "Disable the in-memory cache for images. WARNING: This will slow down scans significantly.")
	return cmd
}

func ContainerTarProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "tar path",
		Short:  docs.GetShort("container-tar"),
		Long:   docs.GetLong("container-tar"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				cmd.Flags().Set("path", args[0])
			}
			runFn(cmd, args)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func ContainerRegistryProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Aliases: []string{"cr"},
		Use:     "registry TARGET",
		Short:   docs.GetShort("container-registry"),
		Long:    docs.GetLong("container-registry"),
		Args:    cobra.ExactArgs(1),
		PreRun:  preRun,
		Run:     runFn,
	}
	commonCmdFlags(cmd)
	return cmd
}

func DockerProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "docker ID",
		Short:  docs.GetShort("docker"),
		Long:   docs.GetLong("docker"),
		Args:   cobra.MaximumNArgs(1),
		PreRun: preRun,
		Run: func(cmd *cobra.Command, args []string) {
			discover, err := cmd.Flags().GetString("discover")
			if err != nil {
				log.Error().Err(err).Msg("failed to retrieve discover flag")
				return
			}

			// If no target is provided and the discovery flag is empty or auto, then error out since there is nothing to scan.
			if len(args) == 0 && (len(discover) == 0 || strings.Contains(discover, discovery_common.DiscoveryAuto)) {
				log.Error().Msg("either a target or a discovery flag different from \"auto\" must be provided for docker scans")
				return
			}

			runFn(cmd, args)
		},
	}
	commonCmdFlags(cmd)
	return cmd
}

func DockerContainerProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "container ID",
		Short:  docs.GetShort("docker-container"),
		Long:   docs.GetLong("docker-container"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run:    runFn,
	}
	commonCmdFlags(cmd)
	return cmd
}

func DockerImageProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image ID",
		Short: docs.GetShort("docker-image"),
		Long:  docs.GetLong("docker-image"),
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
			viper.BindPFlag("disable-cache", cmd.Flags().Lookup("disable-cache"))
		},
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().Bool("disable-cache", false, "Disable the in-memory cache for images. WARNING: This will slow down scans significantly.")
	return cmd
}

func KubernetesProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "k8s (optional MANIFEST path)",
		Aliases: []string{"kubernetes"},
		Short:   docs.GetShort("kubernetes"),
		Long:    docs.GetLong("kubernetes"),
		Args:    cobra.MaximumNArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
			viper.BindPFlag("namespaces-exclude", cmd.Flags().Lookup("namespaces-exclude"))
			viper.BindPFlag("namespaces", cmd.Flags().Lookup("namespaces"))
			viper.BindPFlag("context", cmd.Flags().Lookup("context"))
		},
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				cmd.Flags().Set("path", args[0])
			}
			runFn(cmd, args)
		},
	}
	commonCmdFlags(cmd)

	cmd.Flags().String("context", "", "Target a Kubernetes context.")
	cmd.Flags().String("namespaces-exclude", "", "Filter out Kubernetes objects in the matching namespaces.")
	cmd.Flags().String("namespaces", "", "Only include Kubernetes object in the matching namespaces.")
	return cmd
}

func AwsProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
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
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("profile", "", "Pick a named AWS profile to use.")
	cmd.Flags().String("region", "", "AWS region to scan.")
	cmd.Flags().String("role-arn", "", "Role ARN to use for assume-role.")
	cmd.Flags().String("external-id", "", "External ID to use for assume-role.")
	return cmd
}

func AwsEc2ProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
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

func AwsEc2ConnectProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "instance-connect user@host",
		Short:  docs.GetShort("aws-ec2-connect"),
		Long:   docs.GetLong("aws-ec2-connect"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run:    runFn,
	}
	commonCmdFlags(cmd)
	return cmd
}

func AwsEc2EbsProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "ebs INSTANCEID",
		Short:  docs.GetShort("aws-ec2-ebs-instance"),
		Long:   docs.GetLong("aws-ec2-ebs-instance"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run:    runFn,
	}
	commonCmdFlags(cmd)
	return cmd
}

func AwsEc2EbsVolumeProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "volume VOLUMEID",
		Short:  docs.GetShort("aws-ec2-ebs-volume"),
		Long:   docs.GetLong("aws-ec2-ebs-volume"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run:    runFn,
	}
	commonCmdFlags(cmd)
	return cmd
}

func AwsEc2EbsSnapshotProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "snapshot SNAPSHOTID",
		Short:  docs.GetShort("aws-ec2-ebs-snapshot"),
		Long:   docs.GetLong("aws-ec2-ebs-snapshot"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run:    runFn,
	}
	commonCmdFlags(cmd)
	return cmd
}

func AwsEc2SsmProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "ssm user@host",
		Short:  docs.GetShort("aws-ec2-ssm"),
		Long:   docs.GetLong("aws-ec2-ssm"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run:    runFn,
	}
	commonCmdFlags(cmd)
	return cmd
}

func AzureProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "azure",
		Short: docs.GetShort("azure"),
		Long:  docs.GetLong("azure"),
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
			viper.BindPFlag("subscription", cmd.Flags().Lookup("subscription"))
		},
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("tenant-id", "", "Directory (tenant) ID of the service principal.")
	cmd.Flags().String("client-id", "", "Application (client) ID of the service principal.")
	cmd.Flags().String("client-secret", "", "Secret for application.")
	cmd.Flags().String("certificate-path", "", "Path (in PKCS #12/PFX or PEM format) to the authentication certificate.")
	cmd.Flags().String("certificate-secret", "", "Passphrase for the authentication certificate file.")
	cmd.Flags().String("subscription", "", "ID of the Azure subscription to scan.")
	cmd.Flags().String("subscriptions", "", "Comma-separated list of Azure subscriptions to include.")
	cmd.Flags().String("subscriptions-exclude", "", "Comma-separated list of Azure subscriptions to exclude.")

	return cmd
}

func ScanGcpCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gcp",
		Short: docs.GetShort("gcp"),
		Long:  docs.GetLong("gcp"),
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
			viper.BindPFlag("project-id", cmd.Flags().Lookup("project-id"))
			viper.BindPFlag("organization-id", cmd.Flags().Lookup("organization-id"))
			viper.BindPFlag("credentials-path", cmd.Flags().Lookup("credentials-path"))
		},
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("project-id", "", "specify the GCP project ID to scan")
	cmd.Flags().MarkHidden("project-id")
	cmd.Flags().MarkDeprecated("project-id", "--project-id is deprecated in favor of scan gcp project")
	cmd.Flags().String("organization-id", "", "specify the GCP organization ID to scan")
	cmd.Flags().MarkHidden("organization-id")
	cmd.Flags().MarkDeprecated("organization-id", "--organization-id is deprecated in favor of scan gcp org")
	cmd.Flags().String("credentials-path", "", "The path to the service account credentials to access the APIs with")
	return cmd
}

func ScanGcpOrgCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
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
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("credentials-path", "", "The path to the service account credentials to access the APIs with")
	return cmd
}

func ScanGcpProjectCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project PROJECT-ID",
		Short: docs.GetShort("gcp-project"),
		Long:  docs.GetLong("gcp-project"),
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
			viper.BindPFlag("credentials-path", cmd.Flags().Lookup("credentials-path"))
		},
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("credentials-path", "", "The path to the service account credentials to access the APIs with")
	return cmd
}

func ScanGcpComputeInstanceCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instance INSTANCE-NAME",
		Short: docs.GetShort("gcp-compute-instance"),
		Long:  docs.GetLong("gcp-compute-instance"),
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
			viper.BindPFlag("project-id", cmd.Flags().Lookup("project-id"))
			viper.BindPFlag("zone", cmd.Flags().Lookup("zone"))
			viper.BindPFlag("credentials-path", cmd.Flags().Lookup("credentials-path"))
			viper.BindPFlag("create-snapshot", cmd.Flags().Lookup("create-snapshot"))
		},
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("project-id", "", "specify the GCP project ID where the target instance is located")
	cmd.Flags().String("zone", "", "specify the GCP zone where the target instance is located")
	cmd.Flags().String("credentials-path", "", "The path to the service account credentials to access the APIs with")
	cmd.Flags().Bool("create-snapshot", false, "create a new snapshot instead of using the latest available snapshot")
	return cmd
}

func ScanGcpComputeSnapshotCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot SNAPSHOT-NAME",
		Short: docs.GetShort("gcp-compute-snapshot"),
		Long:  docs.GetLong("gcp-compute-snapshot"),
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
			viper.BindPFlag("project-id", cmd.Flags().Lookup("project-id"))
			viper.BindPFlag("zone", cmd.Flags().Lookup("zone"))
			viper.BindPFlag("credentials-path", cmd.Flags().Lookup("credentials-path"))
		},
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("project-id", "", "specify the GCP project ID where the target instance is located")
	cmd.Flags().String("zone", "", "specify the GCP zone where the target instance is located")
	cmd.Flags().String("credentials-path", "", "The path to the service account credentials to access the APIs with")
	return cmd
}

func ScanGcpFolderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "folder FOLDER-ID",
		Short: docs.GetShort("gcp-folder"),
		Long:  docs.GetLong("gcp-folder"),
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
			viper.BindPFlag("credentials-path", cmd.Flags().Lookup("credentials-path"))
		},
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("credentials-path", "", "The path to the service account credentials to access the APIs with")
	return cmd
}

func ScanGcpGcrCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gcr PROJECT",
		Short: docs.GetShort("gcp-gcr"),
		Long:  docs.GetLong("gcp-gcr"),
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("repository", cmd.Flags().Lookup("repository"))
			preRun(cmd, args)
		},
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("repository", "", "specify the GCR repository to scan")
	return cmd
}

func OciProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oci",
		Short: docs.GetShort("oci"),
		Long:  docs.GetLong("oci"),
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
		},
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("tenancy", "", "The tenancy's OCID")
	cmd.Flags().String("user", "", "The user's OCID")
	cmd.Flags().String("region", "", "The selected region")
	cmd.Flags().String("key-path", "", "The path to the private key, that will be used for authentication")
	cmd.Flags().String("fingerprint", "", "The fingerprint of the private key")
	cmd.Flags().String("key-secret", "", "The passphrase for private key, that will be used for authentication")
	// either we require all of the params, needed to build an OCI connection or we default to using the default provider there
	cmd.MarkFlagsRequiredTogether("tenancy", "user", "region", "key-path", "fingerprint")
	return cmd
}

func VsphereProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "vsphere user@host",
		Short:  docs.GetShort("vsphere"),
		Long:   docs.GetLong("vsphere"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run:    runFn,
	}
	commonCmdFlags(cmd)
	return cmd
}

func VsphereVmProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "vm user@host",
		Short:  docs.GetShort("vsphere-vm"),
		Long:   docs.GetLong("vsphere-vm"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run:    runFn,
	}
	commonCmdFlags(cmd)
	return cmd
}

func ScanGithubCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
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

func GithubProviderOrganizationCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: docs.GetShort("github-org"),
		Long:  docs.GetLong("github-org"),
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("token", cmd.Flags().Lookup("token"))
			preRun(cmd, args)
		},
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("token", "", "GitHub personal access tokens")
	return cmd
}

func GithubProviderRepositoryCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: docs.GetShort("github-repo"),
		Long:  docs.GetLong("github-repo"),
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("token", cmd.Flags().Lookup("token"))
			preRun(cmd, args)
		},
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("token", "", "GitHub personal access token")
	return cmd
}

func GithubProviderUserCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: docs.GetShort("github-user"),
		Long:  docs.GetLong("github-user"),
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("token", cmd.Flags().Lookup("token"))
			preRun(cmd, args)
		},
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("token", "", "GitHub personal access token")
	return cmd
}

func GitlabProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gitlab",
		Short: docs.GetShort("gitlab"),
		Long:  docs.GetLong("gitlab"),
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("token", cmd.Flags().Lookup("token"))
			viper.BindPFlag("group", cmd.Flags().Lookup("group"))
			viper.BindPFlag("project", cmd.Flags().Lookup("project"))
			preRun(cmd, args)
		},
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("group", "", "a GitLab group")
	cmd.MarkFlagRequired("group")
	cmd.Flags().String("project", "", "a GitLab project")
	cmd.Flags().String("token", "", "GitLab personal access token")
	return cmd
}

func Ms365ProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ms365",
		Aliases: []string{"microsoft365"},
		Short:   docs.GetShort("ms365"),
		Long:    docs.GetLong("ms365"),
		Args:    cobra.ExactArgs(0),
		PreRun:  preRun,
		Run:     runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("tenant-id", "", "directory (tenant) ID of the service principal")
	cmd.MarkFlagRequired("tenant-id")
	cmd.Flags().String("client-id", "", "application (client) ID of the service principal")
	cmd.MarkFlagRequired("client-id")
	cmd.Flags().String("client-secret", "", "secret for application")
	cmd.Flags().String("certificate-path", "", "Path (in PKCS #12/PFX or PEM format) to the authentication certificate")
	cmd.Flags().String("certificate-secret", "", "passphrase for certificate file")
	cmd.Flags().String("datareport", "", "set the MS365 datareport for the scan")
	return cmd
}

func HostProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "host HOST",
		Short:  docs.GetShort("host"),
		Long:   docs.GetLong("host"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run:    runFn,
	}
	commonCmdFlags(cmd)
	return cmd
}

func AristaProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "arista user@host",
		Short:  docs.GetShort("arista"),
		Long:   docs.GetLong("arista"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run:    runFn,
	}
	commonCmdFlags(cmd)
	return cmd
}

func ScanOktaCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
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
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("organization", "", "specify the Okta organization to scan")
	cmd.Flags().String("token", "", "Okta access token")
	return cmd
}

func ScanGoogleWorkspaceCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
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
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("customer-id", "", "Specify the Google Workspace customer id to scan")
	cmd.Flags().String("impersonated-user-email", "", "The impersonated user's email with access to the Admin APIs")
	cmd.Flags().String("credentials-path", "", "The path to the service account credentials to access the APIs with")

	return cmd
}

func ScanSlackCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slack",
		Short: docs.GetShort("slack"),
		Long:  docs.GetLong("slack"),
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("token", cmd.Flags().Lookup("token"))
			preRun(cmd, args)
		},
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("token", "", "Slack API token")
	return cmd
}

func ScanVcdCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
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
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("user", "", "vCloud Director user")
	cmd.Flags().String("host", "", "vCloud Director Host")
	cmd.Flags().String("organization", "", "vCloud Director Organization (optional)")
	return cmd
}

func ScanFilesystemCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "filesystem",
		Aliases: []string{"fs"},
		Short:   docs.GetShort("filesystem"),
		Long:    docs.GetLong("filesystem"),
		Args:    cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.Flags().Set("path", args[0])
			preRun(cmd, args)
		},
		Run: runFn,
	}
	commonCmdFlags(cmd)
	return cmd
}

func ScanOpcUACmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "opcua",
		Short: docs.GetShort("opcua"),
		Long:  docs.GetLong("opcua"),
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("endpoint", cmd.Flags().Lookup("endpoint"))
			preRun(cmd, args)
		},
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("endpoint", "", "OPC UA service endpoint")
	return cmd
}
