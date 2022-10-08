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
	"go.mondoo.com/cnquery/cli/components"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/cli/inventoryloader"
	"go.mondoo.com/cnquery/cli/reporter"
	"go.mondoo.com/cnquery/cli/theme"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/explorer/scan"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/providers"
)

func init() {
	rootCmd.AddCommand(exploreCmd)
}

var exploreCmd = builder.NewProviderCommand(builder.CommandOpts{
	Use:   "explore",
	Short: "Explore assets with one or more query packs",
	Long: `
This command explores an asset given a query pack. For example, you can explore
the local system with its pre-configured query pack:

    $ cnquery explore local

To manually configure a query pack, use this:

    $ cnquery explore local -f bundle.mql.yaml --incognito

	`,
	Docs: builder.CommandsDocs{
		Entries: map[string]builder.CommandDocsEntry{
			"local": {
				Short: "Explore a local target",
			},
			"mock": {
				Short: "Explore a mock target (a simulated asset)",
				Long: `Explore a mock target, i.e. a simulated asset, whose data was recorded beforehand.
Provide the recording with mock data as an argument:

    cnquery explore container ubuntu:latest --record
    cnquery explore mock recording-20220519173543.toml
`,
			},
			"vagrant": {
				Short: "Explore a Vagrant host",
			},
			"terraform": {
				Short: "Explore all Terraform files in a path (.tf files)",
			},
			"ssh": {
				Short: "Explore a SSH target",
			},
			"winrm": {
				Short: "Explore a WinRM target",
			},
			"container": {
				Short: "Explore a container, an image, or a registry",
				Long: `Explore a container, a container image, or a container registry. By default
we will try to auto-detect the container or image from the provided ID, even
if it's not the full ID:

    cnquery explore container b62b276baab6
    cnquery explore container b62
    cnquery explore container ubuntu:latest

You can also explicitly request the scan of an image or a container registry:

    cnquery explore container image ubuntu:20.04
    cnquery explore container registry harbor.yourdomain.com/project/repository
`,
			},
			"container-image": {
				Short: "Explore a container image",
			},
			"container-registry": {
				Short: "Explore a container registry",
				Long: `Explore a container registry. Supports more parameters for different registries:

    cnquery explore container registry harbor.yourdomain.com/project/repository
    cnquery explore container registry yourname.azurecr.io
    cnquery explore container registry 123456789.dkr.ecr.us-east-1.amazonaws.com/repository
`,
			},
			"docker": {
				Short: "Explore a Docker container or image",
				Long: `Explore a Docker container or image by automatically detecting the provided ID.
You can also specify a subcommand to narrow the scan to containers or images.

    cnquery explore docker b62b276baab6

    cnquery explore docker container b62b
    cnquery explore docker image ubuntu:latest
`,
			},
			"docker-container": {
				Short: "Explore a Docker container",
				Long: `Explore a Docker container. Can be specified as the container ID (e.g. b62b276baab6)
or container name (e.g. elated_poincare).`,
			},
			"docker-image": {
				Short: "Explore a Docker image",
				Long: `Explore a Docker image. Can be specified as the image ID (e.g. b6f507652425)
or the image name (e.g. ubuntu:latest).`,
			},
			"kubernetes": {
				Short: "Explore a Kubernetes cluster",
			},
			"aws": {
				Short: "Explore an AWS account or instance",
				Long: `Explore an AWS account or EC2 instance. It will use your local AWS configuration
for the account scan. See the subcommands to scan EC2 instances.`,
			},
			"aws-ec2": {
				Short: "Explore an AWS instance using one of the available connectors",
			},
			"aws-ec2-connect": {
				Short: "Explore an AWS instance using EC2 Instance Connect",
			},
			"aws-ec2-ebs-instance": {
				Short: "Explore an AWS instance using an EBS volume scan (requires AWS host)",
				Long: `Explore an AWS instance using an EBS volume scan. This requires that the
scan be executed on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ebs-volume": {
				Short: "Explore a specific AWS volume using the EBS volume scan functionality (requires AWS host)",
				Long: `Explore a specific AWS volume using an EBS volume scan. This requires that the
scan be executed on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ebs-snapshot": {
				Short: "Explore a specific AWS snapshot using the EBS volume scan functionality (requires AWS host)",
				Long: `Explore a specific AWS snapshot using an EBS volume scan. This requires that the
scan be executed on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ssm": {
				Short: "Explore an AWS instance using the AWS Systems Manager to connect",
			},
			"azure": {
				Short: "Explore a Microsoft Azure account or instance",
				Long: `Explore a Microsoft Azure account or instance. It will use your local Azure
configuration for the account scan. To scan your Azure compute, you need to
configure your Azure credentials and have SSH access to your instances.`,
			},
			"gcp": {
				Short: "Explore a Google Cloud Platform (GCP) account",
			},
			"gcp-gcr": {
				Short: "Explore a Google Container Registry (GCR)",
			},
			"vsphere": {
				Short: "Explore a VMware vSphere API endpoint",
			},
			"vsphere-vm": {
				Short: "Explore a VMware vSphere VM",
			},
			"github": {
				Short: "Explore a GitHub organization or repository",
			},
			"github-org": {
				Short: "Explore a GitHub organization",
			},
			"github-repo": {
				Short: "Explore a GitHub repository",
			},
			"gitlab": {
				Short: "Explore a GitLab group",
			},
			"ms365": {
				Short: "Explore a Microsoft 365 endpoint",
				Long: `
Here is an example run for Microsoft 365:

    $ cnquery explore ms365 --tenant-id {tennant id} --client-id {client id} --client-secret {client secret}

This example connects to Microsoft 365 using the PKCS #12 formatted certificate:

    $ cnquery explore ms365 --tenant-id {tennant id} --client-id {client id} --certificate-path {certificate.pfx} --certificate-secret {certificate secret}
    $ cnquery explore ms365 --tenant-id {tennant id} --client-id {client id} --certificate-path {certificate.pfx} --ask-pass
`,
			},
			"host": {
				Short: "Explore a host endpoint",
			},
			"arista": {
				Short: "Explore an Arista endpoint",
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
		// FIXME: remove in v7.0 vv
		// we are moving over to the new scan syntax
		cmd.Flags().StringP("connection", "t", "", "set the method used to connect to the asset. supported connections are 'local://', 'docker://' and 'ssh://'")
		// ^^

		// inventories for multi-asset scan
		cmd.Flags().String("inventory-file", "", "path to inventory file")
		cmd.Flags().String("inventory", "", "inventory file")
		cmd.Flags().MarkDeprecated("inventory", "use new `inventory-file` flag instead")
		cmd.Flags().Bool("inventory-ansible", false, "set inventory format to ansible")
		cmd.Flags().Bool("ansible-inventory", false, "set inventory format to ansible")
		cmd.Flags().MarkDeprecated("ansible-inventory", "use the new flag `inventory-ansible` instead")
		cmd.Flags().Bool("inventory-domainlist", false, "set inventory format to domain list")
		cmd.Flags().Bool("domainlist-inventory", false, "set inventory format to domain list")
		cmd.Flags().MarkDeprecated("domainlist-inventory", "use the new flag `inventory-domainlist` instead")

		// bundles, packs & incognito mode
		cmd.Flags().Bool("incognito", false, "incognito mode. do not report scan results to the Mondoo platform.")
		cmd.Flags().StringSlice("querypack", nil, "list of query packs to be executed (requires incognito mode), multiple query packs can be specified")
		cmd.Flags().StringSliceP("querypack-bundle", "f", nil, "path to local query pack file")
		// flag completion command
		cmd.RegisterFlagCompletionFunc("querypack", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return getQueryPacksForCompletion(), cobra.ShellCompDirectiveDefault
		})

		// individual asset flags
		cmd.Flags().StringP("password", "p", "", "password e.g. for ssh/winrm")
		cmd.Flags().Bool("ask-pass", false, "ask for connection password")
		cmd.Flags().StringP("identity-file", "i", "", "selects a file from which the identity (private key) for public key authentication is read")
		cmd.Flags().String("id-detector", "", "user-override for platform id detection mechanism, supported are "+strings.Join(providers.AvailablePlatformIdDetector(), ", "))

		cmd.Flags().String("path", "", "path to a local file or directory that the connection should use")
		cmd.Flags().StringToString("option", nil, "addition connection options, multiple options can be passed in via --option key=value")
		cmd.Flags().String("discover", "", "enable the discovery of nested assets. Supported are 'all|instances|host-instances|host-machines|container|container-images|pods|cronjobs|statefulsets|deployments|jobs|replicasets|daemonsets'")
		cmd.Flags().StringToString("discover-filter", nil, "additional filter for asset discovery")
		cmd.Flags().StringToString("annotation", nil, "add an annotation to the asset") // user-added, editable

		// global asset flags
		cmd.Flags().Bool("insecure", false, "disable TLS/SSL checks or SSH hostkey config")
		cmd.Flags().Bool("sudo", false, "run with sudo")
		cmd.Flags().Bool("record", false, "record backend calls")
		cmd.Flags().MarkHidden("record")

		// v6 should make detect-cicd and category flag public, default for "detect-cicd" should switch to true
		cmd.Flags().Bool("detect-cicd", true, "attempt to detect CI/CD environments and sets the asset category to 'cicd' if detected")
		cmd.Flags().String("category", "fleet", "sets the category for the assets 'fleet|cicd'")
		cmd.Flags().MarkHidden("category")

		// output rendering
		cmd.Flags().StringP("output", "o", "compact", "set output format: "+reporter.AllFormats())
		cmd.Flags().Bool("no-pager", false, "disable interactive scan output pagination")
		cmd.Flags().String("pager", "", "enable scan output pagination with custom pagination command. default is 'less -R'")
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

		// deprecated flags
		viper.BindPFlag("inventory", cmd.Flags().Lookup("inventory"))
		viper.BindPFlag("ansible-inventory", cmd.Flags().Lookup("ansible-inventory"))
		viper.BindPFlag("domainlist-inventory", cmd.Flags().Lookup("domainlist-inventory"))

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

		// if users supply an inventory, we want to continue with it and move forward
		hasInventory := false
		if x, _ := cmd.Flags().GetString("inventory-file"); x != "" {
			hasInventory = true
		}
		if x, _ := cmd.Flags().GetString("inventory-ansible"); x != "" {
			hasInventory = true
		}
		if x, _ := cmd.Flags().GetString("inventory-domainlist"); x != "" {
			hasInventory = true
		}

		// FIXME: remove in v7.0 vv
		// We are still supporting the --connection flag throughout v6.x and will remove
		// it after. Remember to migrate the zero-state here
		connection, _ := cmd.Flags().GetString("connection")
		// Since we support the fallback for --connection, we check if it was provided
		// first before we print the help in case no subcommand was called
		if connection == "" && !hasInventory {
			cmd.Help()
			fmt.Print(`
Please run one of the subcommands to specify the target system. For example:

    $ cnquery explore local

`)
			os.Exit(1)
		}
		// ^^
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

type exploreConfig struct {
	Features       cnquery.Features
	Inventory      *v1.Inventory
	Output         string
	QueryPackPaths []string
	QueryPackNames []string
	Bundle         *explorer.Bundle

	IsIncognito bool
	DoRecord    bool
}

func getCobraScanConfig(cmd *cobra.Command, args []string, provider providers.ProviderType, assetType builder.AssetType) (*exploreConfig, error) {
	conf := exploreConfig{
		Features:       cnquery.DefaultFeatures,
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

	// TODO: DETECT CI/CD
	// TODO: SERVICE CREDENTIALS

	if len(conf.QueryPackPaths) > 0 && !conf.IsIncognito {
		log.Warn().Msg("Scanning with local bundles will switch into --incognito mode by default. Your results will not be sent upstream.")
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

func (c *exploreConfig) loadBundles() error {
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

func RunScan(config *exploreConfig) *explorer.ReportCollection {
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

func printReports(report *explorer.ReportCollection, conf *exploreConfig, cmd *cobra.Command) {
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
