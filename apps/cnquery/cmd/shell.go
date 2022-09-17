package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder"
	"go.mondoo.com/cnquery/cli/assetlist"
	"go.mondoo.com/cnquery/cli/shell"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery"
	"go.mondoo.com/cnquery/motor/inventory"
	"go.mondoo.com/cnquery/motor/providers"
	provider_resolver "go.mondoo.com/cnquery/motor/providers/resolver"
)

func init() {
	rootCmd.AddCommand(shellCmd)
}

var shellCmd = builder.NewProviderCommand(builder.CommandOpts{
	Use:   "shell",
	Short: "Interactive query shell for MQL",
	Long:  `Allows for the interactive exploration of mondoo queries`,
	CommonFlags: func(cmd *cobra.Command) {
		cmd.Flags().StringP("password", "p", "", "connection password e.g. for ssh/winrm")
		cmd.Flags().Bool("ask-pass", false, "ask for connection password")

		cmd.Flags().StringP("command", "c", "", "a command to run in the shell")
		cmd.Flags().StringP("identity-file", "i", "", "Selects a file from which the identity (private key) for public key authentication is read.")
		cmd.Flags().Bool("insecure", false, "disables TLS/SSL checks or SSH hostkey config")
		cmd.Flags().Bool("sudo", false, "runs with sudo")
		cmd.Flags().String("platform-id", "", "select an specific asset by providing the platform id for the target")
		cmd.Flags().Bool("instances", false, "also scan instances (only applies to api targets like aws, azure or gcp)")
		cmd.Flags().Bool("host-machines", false, "also scan host machines like ESXi server")
		cmd.Flags().Bool("record", false, "records backend calls")
		cmd.Flags().MarkHidden("record")

		cmd.Flags().String("path", "", "path to a local file or directory that the connection should use")
		cmd.Flags().StringToString("option", nil, "addition connection options, multiple options can be passed in via --option key=value")
		cmd.Flags().String("discover", "", "enables the discovery of nested assets. Supported are 'all|instances|host-instances|host-machines|container|container-images'")
		cmd.Flags().StringToString("discover-filter", nil, "additional filter for asset discovery")
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

    mondoo shell container ubuntu:latest --record
    mondoo shell mock recording-20220519173543.toml
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

    mondoo shell container b62b276baab6
    mondoo shell container b62
    mondoo shell container ubuntu:latest

You can also explicitly connect to an image or a container registry:

    mondoo shell container image ubuntu:20.04
    mondoo shell container registry harbor.yourdomain.com/project/repository
`,
			},
			"container-image": {
				Short: "Connec to a container image",
			},
			"container-registry": {
				Short: "Connect to a container registry",
				Long: `Connect to a container registry. Supports more parameters for different registries:

    mondoo shell cr harbor.yourdomain.com/project/repository
    mondoo shell cr yourname.azurecr.io
    mondoo shell cr 123456789.dkr.ecr.us-east-1.amazonaws.com/repository
`,
			},
			"docker": {
				Short: "Connect to a Docker container or image",
				Long: `Connect to a Docker container or image by automatically detecting the provided ID.
You can also specify a subcommand to narrow the scan to containers or images.

    mondoo shell docker b62b276baab6

    mondoo shell docker container b62b
    mondoo shell docker image ubuntu:latest
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
			"gitlab-org": {
				Short: "Connect to a GitHub organization",
			},
			"github-user": {
				Short: "Connect to a GitHub user",
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

    $ mondoo shell ms365 --tenant-id {tennant id} --client-id {client id} --client-secret {client secret}

This example connects to Microsoft 365 using the PKCS #12 formatted certificate:

    $ mondoo shell ms365 --tenant-id {tennant id} --client-id {client id} --certificate-path {certificate.pfx} --certificate-secret {certificate secret}
    $ mondoo shell ms365 --tenant-id {tennant id} --client-id {client id} --certificate-path {certificate.pfx} --ask-pass
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
		ctx := discovery.InitCtx(context.Background())

		// check if the user used --password without a value
		askPass, err := cmd.Flags().GetBool("ask-pass")
		if err == nil && askPass {
			askForPassword("Enter password: ", cmd.Flags())
		}

		flagAsset := builder.ParseTargetAsset(cmd, args, provider, assetType)

		// Note: this flags is not stored in the config and not retrieved from Viper
		command, err := cmd.Flags().GetString("command")
		if err != nil {
			log.Error().Err(err).Msg("failed to read command flags")
		}

		// determine the scan config from pipe or args
		v1Inventory, err := getInventory(flagAsset, viper.GetBool("insecure"))
		if err != nil {
			log.Fatal().Err(err).Msg("could not load configuration")
		}
		features := cnquery.DefaultFeatures

		log.Info().Msgf("discover related assets for %d asset(s)", len(v1Inventory.Spec.Assets))
		im, err := inventory.New(inventory.WithInventory(v1Inventory))
		if err != nil {
			log.Fatal().Err(err).Msg("could not load asset information")
		}
		assetErrors := im.Resolve(ctx)
		if len(assetErrors) > 0 {
			for a := range assetErrors {
				log.Error().Err(assetErrors[a]).Str("asset", a.Name).Msg("could not connect to asset")
			}
			log.Fatal().Msg("could not resolve assets")
		}

		assetList := im.GetAssets()
		log.Debug().Msgf("resolved %d assets", len(assetList))

		if len(assetList) == 0 {
			log.Fatal().Msg("could not find an asset that we can connect to")
		}

		var connectAsset *asset.Asset
		selectedPlatformID := viper.GetString("platform-id")

		if len(assetList) == 1 {
			connectAsset = assetList[0]
		} else if len(assetList) > 1 && selectedPlatformID != "" {
			connectAsset, err = filterAssetByPlatformID(assetList, selectedPlatformID)
			if err != nil {
				log.Fatal().Err(err).Send()
			}
		} else if len(assetList) > 1 {
			r := &assetlist.SimpleRender{}
			fmt.Println(r.Render(assetList))
			log.Fatal().Msg("cannot connect to more than one asset, use --platform-id to select a specific asset")
		}

		record, _ := cmd.Flags().GetBool("record")
		if record {
			log.Info().Msg("enable recording of platform calls")
		}

		backend, err := provider_resolver.OpenAssetConnection(ctx, connectAsset, im.GetCredential, record)
		if err != nil {
			log.Fatal().Err(err).Msg("could not connect to asset")
		}

		// when we close the shell, we need to close the backend and store the recording
		onCloseHandler := func() {
			// store tracked commands and files
			if backend.IsRecording() {
				filename := "recording-" + time.Now().Format("20060102150405") + ".toml"
				log.Info().Str("filename", filename).Msg("store recordings")
				data := backend.Recording()
				ioutil.WriteFile(filename, data, 0o700)
			}

			// close backend connection
			backend.Close()
		}

		shellOptions := []shell.ShellOption{}
		shellOptions = append(shellOptions, shell.WithOnCloseListener(onCloseHandler))
		shellOptions = append(shellOptions, shell.WithFeatures(features))

		sh, err := shell.New(backend, shellOptions...)
		if err != nil {
			log.Error().Err(err).Msg("failed to initialize Mondoo Shell")
		}
		sh.RunInteractive(command)
	},
})
