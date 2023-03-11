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

var AllSubcommandFuncs = []func(CommonFlagsFn, CommonPreRunFn, RunFn, CommandsDocs) *cobra.Command{
	ContainerProviderCmd,
	ContainerImageProviderCmd,
	ContainerRegistryProviderCmd,
	DockerProviderCmd,
	DockerContainerProviderCmd,
	DockerImageProviderCmd,
	KubernetesProviderCmd,
	AwsProviderCmd,
	AwsEc2ProviderCmd,
	AwsEc2ConnectProviderCmd,
	AwsEc2EbsProviderCmd,
	AwsEc2EbsVolumeProviderCmd,
	AwsEc2EbsSnapshotProviderCmd,
	AwsEc2SsmProviderCmd,
	AzureProviderCmd,
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
		Use:    "image ID",
		Short:  docs.GetShort("container-image"),
		Long:   docs.GetLong("container-image"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run:    runFn,
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
		Use:    "image ID",
		Short:  docs.GetShort("docker-image"),
		Long:   docs.GetLong("docker-image"),
		Args:   cobra.ExactArgs(1),
		PreRun: preRun,
		Run:    runFn,
	}
	commonCmdFlags(cmd)
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
			// FIXME: DEPRECATED, remove in v8.0 vv
			viper.BindPFlag("namespace", cmd.Flags().Lookup("namespace"))
			// ^^
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
	// FIXME: DEPRECATED, remove in v8.0
	cmd.Flags().String("namespace", "", "filter kubernetes objects by namespace")
	cmd.Flags().MarkHidden("namespace")
	// ^^

	cmd.Flags().String("context", "", "target a Kubernetes context")
	cmd.Flags().String("namespaces-exclude", "", "filter out Kubernetes objects in the matching namespaces")
	cmd.Flags().String("namespaces", "", "only include Kubernetes object in the matching namespaces")
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
	cmd.Flags().String("profile", "", "pick a named AWS profile to use")
	cmd.Flags().String("region", "", "the AWS region to scan")
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
	cmd.Flags().String("tenant-id", "", "Directory (tenant) ID of the service principal")
	cmd.Flags().String("client-id", "", "Application (client) ID of the service principal")
	cmd.Flags().String("client-secret", "", "Secret for application")
	cmd.Flags().String("certificate-path", "", "Path (in PKCS #12/PFX or PEM format) to the authentication certificate")
	cmd.Flags().String("certificate-secret", "", "Passphrase for the authentication certificate file")
	cmd.Flags().String("subscription", "", "ID of the Azure subscription to scan")
	cmd.Flags().String("subscriptions", "", "Comma-separated list of Azure subscriptions to include")
	cmd.Flags().String("subscriptions-exclude", "", "Comma-separated list of Azure subscriptions to exclude")

	return cmd
}
