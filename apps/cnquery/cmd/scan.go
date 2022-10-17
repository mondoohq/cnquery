package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder"
	cnquery_config "go.mondoo.com/cnquery/apps/cnquery/cmd/config"
	"go.mondoo.com/cnquery/cli/components"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/cli/execruntime"
	"go.mondoo.com/cnquery/cli/inventoryloader"
	"go.mondoo.com/cnquery/cli/reporter"
	"go.mondoo.com/cnquery/cli/theme"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/explorer/scan"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/upstream"
	"go.mondoo.com/ranger-rpc"
)

func init() {
	rootCmd.AddCommand(scanCmd)
}

var scanCmd = builder.NewProviderCommand(builder.CommandOpts{
	Use:     "scan",
	Aliases: []string{"explore"},
	Short:   "Scans assets with one or more query packs",
	Long: `
This command scans an asset given a query pack. For example, you can scan
the local system with its pre-configured query pack:

    $ cnquery scan local

To manually configure a query pack, use this:

    $ cnquery scan local -f bundle.mql.yaml --incognito

	`,
	Docs: builder.CommandsDocs{
		Entries: map[string]builder.CommandDocsEntry{
			"local": {
				Short: "Scan a local target",
			},
			"mock": {
				Short: "Scan a mock target (a simulated asset)",
				Long: `Scan a mock target, i.e. a simulated asset, whose data was recorded beforehand.
Provide the recording with mock data as an argument:

    cnquery scan container ubuntu:latest --record
    cnquery scan mock recording-20220519173543.toml
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
				Short: "Scan a container, an image, or a registry",
				Long: `Scan a container, a container image, or a container registry. By default
we will try to auto-detect the container or image from the provided ID, even
if it's not the full ID:

    cnquery scan container b62b276baab6
    cnquery scan container b62
    cnquery scan container ubuntu:latest

You can also explicitly request the scan of an image or a container registry:

    cnquery scan container image ubuntu:20.04
    cnquery scan container registry harbor.lunalectric.com/project/repository
`,
			},
			"container-image": {
				Short: "Scan a container image",
			},
			"container-registry": {
				Short: "Scan a container registry",
				Long: `Scan a container registry. Supports more parameters for different registries:

    cnquery scan container registry harbor.lunalectric.com/project/repository
    cnquery scan container registry yourname.azurecr.io
    cnquery scan container registry 123456789.dkr.ecr.us-east-1.amazonaws.com/repository
`,
			},
			"docker": {
				Short: "Scan a Docker container or image",
				Long: `Scan a Docker container or image by automatically detecting the provided ID.
You can also specify a subcommand to narrow the scan to containers or images.

    cnquery scan docker b62b276baab6

    cnquery scan docker container b62b
    cnquery scan docker image ubuntu:latest
`,
			},
			"docker-container": {
				Short: "Scan a Docker container",
				Long: `Scan a Docker container. Can be specified as the container ID (e.g. b62b276baab6)
or container name (e.g. elated_poincare).`,
			},
			"docker-image": {
				Short: "Scan a Docker image",
				Long: `Scan a Docker image. Can be specified as the image ID (e.g. b6f507652425)
or the image name (e.g. ubuntu:latest).`,
			},
			"kubernetes": {
				Short: "Scan a Kubernetes cluster",
			},
			"aws": {
				Short: "Scan an AWS account or instance",
				Long: `Scan an AWS account or EC2 instance. It will use your local AWS configuration
for the account scan. See the subcommands to scan EC2 instances.`,
			},
			"aws-ec2": {
				Short: "Scan an AWS instance using one of the available connectors",
			},
			"aws-ec2-connect": {
				Short: "Scan an AWS instance using EC2 Instance Connect",
			},
			"aws-ec2-ebs-instance": {
				Short: "Scan an AWS instance using an EBS volume scan (requires AWS host)",
				Long: `Scan an AWS instance using an EBS volume scan. This requires that the
scan be executed on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ebs-volume": {
				Short: "Scan a specific AWS volume using the EBS volume scan functionality (requires AWS host)",
				Long: `Scan a specific AWS volume using an EBS volume scan. This requires that the
scan be executed on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ebs-snapshot": {
				Short: "Scan a specific AWS snapshot using the EBS volume scan functionality (requires AWS host)",
				Long: `Scan a specific AWS snapshot using an EBS volume scan. This requires that the
scan be executed on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ssm": {
				Short: "Scan an AWS instance using the AWS Systems Manager to connect",
			},
			"azure": {
				Short: "Scan a Microsoft Azure account or instance",
				Long: `Scan a Microsoft Azure account or instance. It will use your local Azure
configuration for the account scan. To scan your Azure compute, you need to
configure your Azure credentials and have SSH access to your instances.`,
			},
			"gcp": {
				Short: "Scan a Google Cloud Platform (GCP) account",
			},
			"gcp-gcr": {
				Short: "Scan a Google Container Registry (GCR)",
			},
			"vsphere": {
				Short: "Scan a VMware vSphere API endpoint",
			},
			"vsphere-vm": {
				Short: "Scan a VMware vSphere VM",
			},
			"github": {
				Short: "Scan a GitHub organization or repository",
			},
			"github-org": {
				Short: "Scan a GitHub organization",
			},
			"github-repo": {
				Short: "Scan a GitHub repository",
			},
			"gitlab": {
				Short: "Scan a GitLab group",
			},
			"ms365": {
				Short: "Scan a Microsoft 365 endpoint",
				Long: `
Here is an example run for Microsoft 365:

    $ cnquery scan ms365 --tenant-id {tenant id} --client-id {client id} --client-secret {client secret}

This example connects to Microsoft 365 using the PKCS #12 formatted certificate:

    $ cnquery scan ms365 --tenant-id {tenant id} --client-id {client id} --certificate-path {certificate.pfx} --certificate-secret {certificate secret}
    $ cnquery scan ms365 --tenant-id {tenant id} --client-id {client id} --certificate-path {certificate.pfx} --ask-pass
`,
			},
			"host": {
				Short: "Scan a host endpoint",
			},
			"arista": {
				Short: "Scan an Arista endpoint",
			},
		},
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return []string{"yml", "yaml", "json"}, cobra.ShellCompDirectiveFilterFileExt
		}
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	},
	CommonFlags: func(cmd *cobra.Command) {
		// inventories for multi-asset scan
		cmd.Flags().String("inventory-file", "", "Set the path to inventory file")
		cmd.Flags().Bool("inventory-ansible", false, "Set the inventory format to Ansible")
		cmd.Flags().Bool("inventory-domainlist", false, "Set the inventory format to domain list")

		// bundles, packs & incognito mode
		cmd.Flags().Bool("incognito", false, "Run in incognito mode. Do not report scan results to the Mondoo platform.")
		cmd.Flags().StringSlice("querypack", nil, "Set the query packs to be executed (requires incognito mode), multiple query packs can be specified")
		cmd.Flags().StringSliceP("querypack-bundle", "f", nil, "Path to local query pack file")
		// flag completion command
		cmd.RegisterFlagCompletionFunc("querypack", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return getQueryPacksForCompletion(), cobra.ShellCompDirectiveDefault
		})

		// individual asset flags
		cmd.Flags().StringP("password", "p", "", "Password e.g. for ssh/winrm")
		cmd.Flags().Bool("ask-pass", false, "Ask for connection password")
		cmd.Flags().StringP("identity-file", "i", "", "Select a file from which the identity (private key) for public key authentication is read")
		cmd.Flags().String("id-detector", "", "User-override for platform id detection mechanism, supported are "+strings.Join(providers.AvailablePlatformIdDetector(), ", "))

		cmd.Flags().String("path", "", "Path to a local file or directory that the connection should use")
		cmd.Flags().StringToString("option", nil, "Additional connection options, multiple options can be passed in via --option key=value")
		cmd.Flags().String("discover", common.DiscoveryAuto, "Enable the discovery of nested assets. Supported are 'all|auto|instances|host-instances|host-machines|container|container-images|pods|cronjobs|statefulsets|deployments|jobs|replicasets|daemonsets'")
		cmd.Flags().StringToString("discover-filter", nil, "Additional filter for asset discovery")
		cmd.Flags().StringToString("annotation", nil, "Add an annotation to the asset") // user-added, editable

		// global asset flags
		cmd.Flags().Bool("insecure", false, "Disable TLS/SSL checks or SSH hostkey config")
		cmd.Flags().Bool("sudo", false, "Elevate privileges using sudo")
		cmd.Flags().Bool("record", false, "Record all backend calls")
		cmd.Flags().MarkHidden("record")

		// v6 should make detect-cicd and category flag public, default for "detect-cicd" should switch to true
		cmd.Flags().Bool("detect-cicd", true, "Attempt to detect CI/CD environments and sets the asset category to 'cicd' if detected")
		cmd.Flags().String("category", "fleet", "Sets the category for the assets 'fleet|cicd'")
		cmd.Flags().MarkHidden("category")

		// output rendering
		cmd.Flags().StringP("output", "o", "compact", "Set output format: "+reporter.AllFormats())
		cmd.Flags().BoolP("json", "j", false, "Set output to JSON (shorthand)")
		cmd.Flags().Bool("no-pager", false, "Disable interactive scan output pagination")
		cmd.Flags().String("pager", "", "Enable scan output pagination with custom pagination command (default 'less -R')")
	},
	CommonPreRun: func(cmd *cobra.Command, args []string) {
		// multiple assets mapping
		viper.BindPFlag("inventory-file", cmd.Flags().Lookup("inventory-file"))
		viper.BindPFlag("inventory-ansible", cmd.Flags().Lookup("inventory-ansible"))
		viper.BindPFlag("inventory-domainlist", cmd.Flags().Lookup("inventory-domainlist"))
		viper.BindPFlag("querypack-bundle", cmd.Flags().Lookup("querypack-bundle"))
		viper.BindPFlag("id-detector", cmd.Flags().Lookup("id-detector"))
		viper.BindPFlag("detect-cicd", cmd.Flags().Lookup("detect-cicd"))
		viper.BindPFlag("category", cmd.Flags().Lookup("category"))

		// for all assets
		viper.BindPFlag("incognito", cmd.Flags().Lookup("incognito"))
		viper.BindPFlag("insecure", cmd.Flags().Lookup("insecure"))
		viper.BindPFlag("querypacks", cmd.Flags().Lookup("querypack"))
		viper.BindPFlag("sudo.active", cmd.Flags().Lookup("sudo"))

		viper.BindPFlag("output", cmd.Flags().Lookup("output"))
		// the logic is that noPager takes precedence over pager if both are sent
		viper.BindPFlag("no_pager", cmd.Flags().Lookup("no-pager"))
		viper.BindPFlag("pager", cmd.Flags().Lookup("pager"))
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		// Special handling for users that want to see what output options are
		// available. We have to do this before printing the help because we
		// don't have a target connection or provider.
		output, _ := cmd.Flags().GetString("output")
		if output == "help" {
			fmt.Println("Available output formats: " + reporter.AllFormats())
			os.Exit(0)
		}
	},
	Run: func(cmd *cobra.Command, args []string, provider providers.ProviderType, assetType builder.AssetType) {
		conf, err := getCobraScanConfig(cmd, args, provider, assetType)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to prepare config")
		}

		err = conf.loadBundles()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to resolve query packs")
		}

		report := RunScan(conf)
		printReports(report, conf, cmd)
	},
})

// helper method to retrieve the list of query packs for autocomplete
func getQueryPacksForCompletion() []string {
	querypackList := []string{}

	// TODO: autocompletion

	sort.Strings(querypackList)

	return querypackList
}

type scanConfig struct {
	Features       cnquery.Features
	Inventory      *v1.Inventory
	Output         string
	QueryPackPaths []string
	QueryPackNames []string
	Bundle         *explorer.Bundle

	IsIncognito bool
	DoRecord    bool

	UpstreamConfig *resources.UpstreamConfig
}

func getCobraScanConfig(cmd *cobra.Command, args []string, provider providers.ProviderType, assetType builder.AssetType) (*scanConfig, error) {
	opts, optsErr := cnquery_config.ReadConfig()
	if optsErr != nil {
		log.Fatal().Err(optsErr).Msg("could not load configuration")
	}
	config.DisplayUsedConfig()

	// display activated features
	if len(opts.Features) > 0 {
		log.Info().Strs("features", opts.Features).Msg("user activated features")
	}

	conf := scanConfig{
		Features:       opts.GetFeatures(),
		IsIncognito:    viper.GetBool("incognito"),
		DoRecord:       viper.GetBool("record"),
		QueryPackPaths: viper.GetStringSlice("querypack-bundle"),
		QueryPackNames: viper.GetStringSlice("querypack"),
	}
	config.DisplayUsedConfig()

	// if users want to get more information on available output options,
	// print them before executing the scan
	output, _ := cmd.Flags().GetString("output")
	if output == "help" {
		fmt.Println("Available output formats: " + reporter.AllFormats())
		os.Exit(0)
	}

	// --json takes precedence
	if ok, _ := cmd.Flags().GetBool("json"); ok {
		output = "json"
	}
	conf.Output = output

	// check if the user used --password without a value
	askPass, err := cmd.Flags().GetBool("ask-pass")
	if err == nil && askPass {
		pass, err := components.AskPassword("Enter password: ")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get password")
		}
		cmd.Flags().Set("password", pass)
	}

	// determine the scan config from pipe or args
	flagAsset := builder.ParseTargetAsset(cmd, args, provider, assetType)
	conf.Inventory, err = inventoryloader.ParseOrUse(flagAsset, viper.GetBool("insecure"))
	if err != nil {
		return nil, errors.Wrap(err, "could not load configuration")
	}

	// detect CI/CD runs and read labels from runtime and apply them to all assets in the inventory
	runtimeEnv := execruntime.Detect()
	if opts.AutoDetectCICDCategory && runtimeEnv.IsAutomatedEnv() || opts.Category == "cicd" {
		log.Info().Msg("detected ci-cd environment")
		// NOTE: we only apply those runtime environment labels for CI/CD runs to ensure other assets from the
		// inventory are not touched, we may consider to add the data to the flagAsset
		if runtimeEnv != nil {
			runtimeLabels := runtimeEnv.Labels()
			conf.Inventory.ApplyLabels(runtimeLabels)
		}
		conf.Inventory.ApplyCategory(asset.AssetCategory_CATEGORY_CICD)
	}

	serviceAccount := opts.GetServiceCredential()
	if serviceAccount != nil {
		log.Info().Msg("using service account credentials")
		certAuth, _ := upstream.NewServiceAccountRangerPlugin(serviceAccount)

		conf.UpstreamConfig = &resources.UpstreamConfig{
			SpaceMrn:    opts.GetParentMrn(),
			ApiEndpoint: opts.UpstreamApiEndpoint(),
			Plugins:     []ranger.ClientPlugin{certAuth},
		}
	}

	if len(conf.QueryPackPaths) > 0 && !conf.IsIncognito {
		log.Warn().Msg("Scanning with local bundles will switch into --incognito mode by default. Your results will not be sent upstream.")
		conf.IsIncognito = true
	}

	if serviceAccount == nil && !conf.IsIncognito {
		log.Warn().Msg("No credentials provided. Switching to --incogito mode.")
		conf.IsIncognito = true
	}

	// print headline when its not printed to yaml
	if output == "" {
		fmt.Fprintln(os.Stdout, theme.DefaultTheme.Welcome)
	}

	if conf.DoRecord {
		log.Info().Msg("enable recording of platform calls")
	}

	return &conf, nil
}

func (c *scanConfig) loadBundles() error {
	if c.IsIncognito {
		if len(c.QueryPackPaths) == 0 {
			return nil
		}

		bundle, err := explorer.BundleFromPaths(c.QueryPackPaths...)
		if err != nil {
			return err
		}

		c.Bundle = bundle
		return nil
	}

	return errors.New("Cannot yet resolve query packs other than incognito")
}

func RunScan(config *scanConfig) *explorer.ReportCollection {
	scanner := scan.NewLocalScanner()

	ctx := cnquery.SetFeatures(context.Background(), config.Features)

	reports, err := scanner.RunIncognito(
		ctx,
		&scan.Job{
			DoRecord:         config.DoRecord,
			Inventory:        config.Inventory,
			Bundle:           config.Bundle,
			QueryPackFilters: config.QueryPackNames,
		})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run scan")
	}

	return reports
}

func printReports(report *explorer.ReportCollection, conf *scanConfig, cmd *cobra.Command) {
	// print the output using the specified output format
	r, err := reporter.New(conf.Output)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	r.UsePager, _ = cmd.Flags().GetBool("pager")
	r.Pager, _ = cmd.Flags().GetString("pager")
	r.IsIncognito = conf.IsIncognito

	if err = r.Print(report, os.Stdout); err != nil {
		log.Fatal().Err(err).Msg("failed to print")
	}
}
