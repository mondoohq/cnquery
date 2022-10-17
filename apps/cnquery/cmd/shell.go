package cmd

import (
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder"
	cnquery_config "go.mondoo.com/cnquery/apps/cnquery/cmd/config"
	"go.mondoo.com/cnquery/cli/components"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/cli/inventoryloader"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/upstream"
	"go.mondoo.com/ranger-rpc"
)

func init() {
	rootCmd.AddCommand(shellCmd)
}

var shellCmd = builder.NewProviderCommand(builder.CommandOpts{
	Use:   "shell",
	Short: "Interactive query shell for MQL",
	Long:  `Allows for the interactive exploration of MQL queries`,
	CommonFlags: func(cmd *cobra.Command) {
		cmd.Flags().StringP("password", "p", "", "Set the connection password e.g. for ssh/winrm")
		cmd.Flags().Bool("ask-pass", false, "Prompt for connection password")

		cmd.Flags().String("query", "", "MQL query to be executed")
		cmd.Flags().MarkHidden("query")
		cmd.Flags().StringP("command", "c", "", "MQL query to be executed in the shell")
		cmd.Flags().StringP("identity-file", "i", "", "Select a file from which the identity (private key) for public key authentication is read.")
		cmd.Flags().Bool("insecure", false, "Disable TLS/SSL checks or SSH hostkey config")
		cmd.Flags().Bool("sudo", false, "Elevate privileges with sudo")
		cmd.Flags().String("platform-id", "", "Select an specific asset by providing the platform id for the target")
		cmd.Flags().Bool("instances", false, "Also scan instances (only applies to api targets like aws, azure or gcp)")
		cmd.Flags().Bool("host-machines", false, "Also scan host machines like ESXi server")

		cmd.Flags().Bool("record", false, "Record all backend calls")
		cmd.Flags().MarkHidden("record")

		cmd.Flags().String("record-file", "", "File path to for the recorded provider calls (only works for operating system providers)")
		cmd.Flags().MarkHidden("record-file")

		cmd.Flags().String("path", "", "Path to a local file or directory that the connection should use")
		cmd.Flags().StringToString("option", nil, "Additional connection options, multiple options can be passed in via --option key=value")
		cmd.Flags().String("discover", common.DiscoveryAuto, "Enable the discovery of nested assets. Supported are 'all|auto|instances|host-instances|host-machines|container|container-images|pods|cronjobs|statefulsets|deployments|jobs|replicasets|daemonsets'")
		cmd.Flags().StringToString("discover-filter", nil, "Additional filter for asset discovery")
	},
	CommonPreRun: func(cmd *cobra.Command, args []string) {
		// for all assets
		viper.BindPFlag("incognito", cmd.Flags().Lookup("incognito"))
		viper.BindPFlag("insecure", cmd.Flags().Lookup("insecure"))
		viper.BindPFlag("policies", cmd.Flags().Lookup("policy"))
		viper.BindPFlag("sudo.active", cmd.Flags().Lookup("sudo"))

		viper.BindPFlag("output", cmd.Flags().Lookup("output"))

		viper.BindPFlag("vault.name", cmd.Flags().Lookup("vault"))
		viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))

		viper.BindPFlag("record", cmd.Flags().Lookup("record"))
		viper.BindPFlag("record-file", cmd.Flags().Lookup("record-file"))
	},
	Docs: builder.CommandsDocs{
		Entries: map[string]builder.CommandDocsEntry{
			"local": {
				Short: "Connect to a local machine",
			},
			"mock": {
				Short: "Connect to mock target (a simulated asset)",
				Long: `Connect to a mock target, i.e. a simulated asset, whose data was recorded beforehand.
Provide the recording with mock data as an argument:

    cnquery shell container ubuntu:latest --record
    cnquery shell mock recording-20220519173543.toml
`,
			},
			"vagrant": {
				Short: "Scan a Vagrant host",
			},
			"terraform": {
				Short: "Scan all Terraform files in a path (.tf files)",
			},
			"ssh": {
				Short: "Scan a SSH target",
			},
			"winrm": {
				Short: "Scan a WinRM target",
			},
			"container": {
				Short: "Connect to a container, an image, or a registry",
				Long: `Connect to a container, a container image, or a container registry. By default
we will try to auto-detect the container or image from the provided ID, even
if it's not the full ID:

    cnquery shell container b62b276baab6
    cnquery shell container b62
    cnquery shell container ubuntu:latest

You can also explicitly connect to an image or a container registry:

    cnquery shell container image ubuntu:20.04
    cnquery shell container registry harbor.lunalectric.com/project/repository
`,
			},
			"container-image": {
				Short: "Connect to a container image",
			},
			"container-registry": {
				Short: "Connect to a container registry",
				Long: `Connect to a container registry. Supports more parameters for different registries:

    cnquery shell container registry harbor.lunalectric.com/project/repository
    cnquery shell container registry yourname.azurecr.io
    cnquery shell container registry 123456789.dkr.ecr.us-east-1.amazonaws.com/repository
`,
			},
			"docker": {
				Short: "Connect to a Docker container or image",
				Long: `Connect to a Docker container or image by automatically detecting the provided ID.
You can also specify a subcommand to narrow the scan to containers or images.

    cnquery shell docker b62b276baab6

    cnquery shell docker container b62b
    cnquery shell docker image ubuntu:latest
`,
			},
			"docker-container": {
				Short: "Connect to a Docker container",
				Long: `Connect to a Docker container. Can be specified as the container ID (e.g. b62b276baab6)
or container name (e.g. elated_poincare).`,
			},
			"docker-image": {
				Short: "Connect to a Docker image",
				Long: `Connect to a Docker image. Can be specified as the image ID (e.g. b6f507652425)
or the image name (e.g. ubuntu:latest).`,
			},
			"kubernetes": {
				Short: "Connect to a Kubernetes cluster or manifest",
			},
			"aws": {
				Short: "Connect to an AWS account or instance",
				Long: `Connect to an AWS account or EC2 instance. It will use your local AWS configuration
for the account scan. See the subcommands to scan EC2 instances.`,
			},
			"aws-ec2": {
				Short: "Connect to an AWS instance using one of the available connectors",
			},
			"aws-ec2-connect": {
				Short: "Connect to an AWS instance using EC2 Instance Connect",
			},
			"aws-ec2-ebs-instance": {
				Short: "Connect to an AWS instance using an EBS volume scan (requires AWS host)",
				Long: `Connect to an AWS instance using an EBS volume scan. This requires that the
scan be executed on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ebs-volume": {
				Short: "Connect to a specific AWS volume using the EBS volume scan functionality (requires AWS host)",
				Long: `Connect to a specific AWS volume using an EBS volume scan. This requires that the
scan be executed on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ebs-snapshot": {
				Short: "Connect to a specific AWS snapshot using the EBS volume scan functionality (requires AWS host)",
				Long: `Connect to a specific AWS snapshot using an EBS volume scan. This requires that the
scan be executed on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ssm": {
				Short: "Connect to an AWS instance using the AWS Systems Manager to connect",
			},
			"azure": {
				Short: "Connect to a Microsoft Azure account or instance",
				Long: `Connect to a Microsoft Azure account or instance. It will use your local Azure
configuration for the account scan. To scan your Azure compute, you need to
configure your Azure credentials and have SSH access to your instances.`,
			},
			"gcp": {
				Short: "Connect to a Google Cloud Platform (GCP) account",
			},
			"gcp-gcr": {
				Short: "Connect to a Google Container Registry (GCR)",
			},
			"vsphere": {
				Short: "Connect to a VMware vSphere API endpoint",
			},
			"vsphere-vm": {
				Short: "Connect to a VMware vSphere VM",
			},
			"github": {
				Short: "Connect to a GitHub organization or repository",
			},
			"github-org": {
				Short: "Connect to a GitHub organization",
			},
			"github-repo": {
				Short: "Connect to a GitHub repository",
			},
			"gitlab": {
				Short: "Connect to a GitLab group",
			},
			"ms365": {
				Short: "Connect to a Microsoft 365 tenant",
				Long: `
This command opens a shell to a Microsoft 365 tenant:

    $ cnquery shell ms365 --tenant-id {tenant id} --client-id {client id} --client-secret {client secret}

This example connects to Microsoft 365 using the PKCS #12 formatted certificate:

    $ cnquery shell ms365 --tenant-id {tenant id} --client-id {client id} --certificate-path {certificate.pfx} --certificate-secret {certificate secret}
    $ cnquery shell ms365 --tenant-id {tenant id} --client-id {client id} --certificate-path {certificate.pfx} --ask-pass
`,
			},
			"host": {
				Short: "Connect to a host endpoint",
			},
			"arista": {
				Short: "Connect to an Arista endpoint",
			},
		},
	},
	Run: func(cmd *cobra.Command, args []string, provider providers.ProviderType, assetType builder.AssetType) {
		conf, err := GetCobraShellConfig(cmd, args, provider, assetType)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to prepare config")
		}

		err = StartShell(conf)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to run query")
		}
	},
})

func GetCobraShellConfig(cmd *cobra.Command, args []string, provider providers.ProviderType, assetType builder.AssetType) (*ShellConfig, error) {
	opts, optsErr := cnquery_config.ReadConfig()
	if optsErr != nil {
		log.Fatal().Err(optsErr).Msg("could not load configuration")
	}

	config.DisplayUsedConfig()

	conf := ShellConfig{
		Features: config.Features,
	}

	// check if the user used --password without a value
	askPass, err := cmd.Flags().GetBool("ask-pass")
	if err == nil && askPass {
		pass, err := components.AskPassword("Enter password: ")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get password")
		}
		cmd.Flags().Set("password", pass)
	}

	conf.Command, _ = cmd.Flags().GetString("command")
	// fallback to --query
	if conf.Command == "" {
		conf.Command, _ = cmd.Flags().GetString("query")
	}

	conf.DoRecord = viper.GetBool("record")

	// determine the scan config from pipe or args
	flagAsset := builder.ParseTargetAsset(cmd, args, provider, assetType)
	conf.Inventory, err = inventoryloader.ParseOrUse(flagAsset, viper.GetBool("insecure"))
	if err != nil {
		return nil, errors.Wrap(err, "could not load configuration")
	}

	conf.PlatformID = viper.GetString("platform-id")

	serviceAccount := opts.GetServiceCredential()
	if serviceAccount != nil {
		certAuth, _ := upstream.NewServiceAccountRangerPlugin(serviceAccount)

		conf.UpstreamConfig = &resources.UpstreamConfig{
			SpaceMrn:    opts.GetParentMrn(),
			ApiEndpoint: opts.UpstreamApiEndpoint(),
			Plugins:     []ranger.ClientPlugin{certAuth},
			// we do not use opts here since we want to ensure the result is not stored when users use the shell
			Incognito: true,
		}
	}

	return &conf, nil
}
