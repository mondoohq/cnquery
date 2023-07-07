package cmd

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/providers"
	"go.mondoo.com/cnquery/providers/proto"
	"go.mondoo.com/cnquery/shared"
	run "go.mondoo.com/cnquery/shared/proto"
)

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.Flags().BoolP("json", "j", false, "Run the query and return the object in a JSON structure.")
	scanCmd.Flags().String("platform-id", "", "Select a specific target asset by providing its platform ID.")
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan assets with one or more query packs.",
	Long: `
This command scans an asset using a query pack. For example, you can scan
the local system with its pre-configured query pack:

		$ cnquery scan local

To manually configure a query pack, use this:

		$ cnquery scan local -f bundle.mql.yaml --incognito

`,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return []string{"yml", "yaml", "json"}, cobra.ShellCompDirectiveFilterFileExt
		}
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	},
	CommonFlags: func(cmd *cobra.Command) {
		// inventories for multi-asset scan
		cmd.Flags().String("inventory-file", "", "Set the path to the inventory file.")
		cmd.Flags().Bool("inventory-ansible", false, "Set the inventory format to Ansible.")
		cmd.Flags().Bool("inventory-domainlist", false, "Set the inventory format to domain list.")

		// bundles, packs & incognito mode
		cmd.Flags().Bool("incognito", false, "Run in incognito mode. Do not report scan results to  Mondoo Platform.")
		cmd.Flags().StringSlice("querypack", nil, "Set the query packs to execute. This requires `querypack-bundle`. You can specify multiple UIDs.")
		cmd.Flags().StringSliceP("querypack-bundle", "f", nil, "Path to local query pack file")
		// flag completion command
		cmd.RegisterFlagCompletionFunc("querypack", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return getQueryPacksForCompletion(), cobra.ShellCompDirectiveDefault
		})

		// individual asset flags
		cmd.Flags().StringP("password", "p", "", "Password, such as for SSH/WinRM.")
		cmd.Flags().Bool("ask-pass", false, "Ask for connection password.")
		cmd.Flags().StringP("identity-file", "i", "", "Select a file from which to read the identity (private key) for public key authentication.")
		cmd.Flags().String("id-detector", "", "User override for platform ID detection mechanism. Supported: "+strings.Join(providers.AvailablePlatformIdDetector(), ", "))
		cmd.Flags().String("asset-name", "", "User-override for the asset name")
		cmd.Flags().StringToString("props", nil, "Custom values for properties")

		cmd.Flags().String("path", "", "Path to a local file or directory for the connection to use.")
		cmd.Flags().StringToString("option", nil, "Additional connection options. You can pass multiple options using `--option key=value`.")
		cmd.Flags().String("discover", discovery_common.DiscoveryAuto, "Enable the discovery of nested assets. Supported: 'all|auto|instances|host-instances|host-machines|container|container-images|pods|cronjobs|statefulsets|deployments|jobs|replicasets|daemonsets'")
		cmd.Flags().StringToString("discover-filter", nil, "Additional filter for asset discovery.")
		cmd.Flags().StringToString("annotation", nil, "Add an annotation to the asset.") // user-added, editable

		// global asset flags
		cmd.Flags().Bool("insecure", false, "Disable TLS/SSL checks or SSH hostkey config.")
		cmd.Flags().Bool("sudo", false, "Elevate privileges using sudo.")
		cmd.Flags().Bool("record", false, "Record all backend calls.")
		cmd.Flags().MarkHidden("record")

		// v6 should make detect-cicd and category flag public, default for "detect-cicd" should switch to true
		cmd.Flags().Bool("detect-cicd", true, "Try to detect CI/CD environments. If detected, set the asset category to 'cicd'.")
		cmd.Flags().String("category", "fleet", "Set the category for the assets to 'fleet|cicd'.")
		cmd.Flags().MarkHidden("category")

		// output rendering
		cmd.Flags().StringP("output", "o", "compact", "Set output format: "+reporter.AllFormats())
		cmd.Flags().BoolP("json", "j", false, "Set output to JSON (shorthand).")
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
		viper.BindPFlag("record", cmd.Flags().Lookup("record"))

		viper.BindPFlag("output", cmd.Flags().Lookup("output"))
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

		report, err := RunScan(conf)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to run scan")
		}

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
	Props          map[string]string
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

	props, err := cmd.Flags().GetStringToString("props")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse props")
	}

	conf := scanConfig{
		Features:       opts.GetFeatures(),
		IsIncognito:    viper.GetBool("incognito"),
		DoRecord:       viper.GetBool("record"),
		QueryPackPaths: viper.GetStringSlice("querypack-bundle"),
		QueryPackNames: viper.GetStringSlice("querypacks"),
		Props:          props,
	}

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
	if iddetector, _ := cmd.Flags().GetString("id-detector"); iddetector != "" {
		flagAsset.IdDetector = []string{iddetector}
	}
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

	var serviceAccount *upstream.ServiceAccountCredentials
	if !conf.IsIncognito {
		serviceAccount = opts.GetServiceCredential()
		if serviceAccount != nil {
			httpClient, err := opts.GetHttpClient()
			if err != nil {
				log.Error().Err(err).Msg("error while setting up httpclient")
				os.Exit(ConfigurationErrorCode)
			}
			certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
			if err != nil {
				log.Error().Err(err).Msg("could not initialize client authentication")
				os.Exit(ConfigurationErrorCode)
			}
			plugins := []ranger.ClientPlugin{certAuth}
			// determine information about the client
			sysInfo, err := sysinfo.GatherSystemInfo()
			if err != nil {
				log.Warn().Err(err).Msg("could not gather client information")
			}
			plugins = append(plugins, defaultRangerPlugins(sysInfo, opts.GetFeatures())...)
			log.Info().Msg("using service account credentials")
			conf.UpstreamConfig = &resources.UpstreamConfig{
				SpaceMrn:    opts.GetParentMrn(),
				ApiEndpoint: opts.UpstreamApiEndpoint(),
				Plugins:     plugins,
				HttpClient:  httpClient,
			}
		}
	}

	if len(conf.QueryPackPaths) > 0 && !conf.IsIncognito {
		log.Warn().Msg("Scanning with local bundles will switch into --incognito mode by default. Your results will not be sent upstream.")
		conf.IsIncognito = true
	}

	if serviceAccount == nil && !conf.IsIncognito {
		log.Warn().Msg("No credentials provided. Switching to --incognito mode.")
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

		_, err = bundle.Compile(context.Background())
		if err != nil {
			return errors.Wrap(err, "failed to compile bundle")
		}

		c.Bundle = bundle
		return nil
	}

	return nil
}

func RunScan(config *scanConfig) (*explorer.ReportCollection, error) {
	opts := []scan.ScannerOption{}
	if config.UpstreamConfig != nil {
		opts = append(opts, scan.WithUpstream(config.UpstreamConfig.ApiEndpoint, config.UpstreamConfig.SpaceMrn, config.UpstreamConfig.Plugins, config.UpstreamConfig.HttpClient))
	}

	scanner := scan.NewLocalScanner(opts...)
	ctx := cnquery.SetFeatures(context.Background(), config.Features)

	if config.IsIncognito {
		return scanner.RunIncognito(
			ctx,
			&scan.Job{
				DoRecord:         config.DoRecord,
				Inventory:        config.Inventory,
				Bundle:           config.Bundle,
				QueryPackFilters: config.QueryPackNames,
				Props:            config.Props,
			})
	}
	return scanner.Run(
		ctx,
		&scan.Job{
			DoRecord:         config.DoRecord,
			Inventory:        config.Inventory,
			Bundle:           config.Bundle,
			QueryPackFilters: config.QueryPackNames,
			Props:            config.Props,
		})
}

func printReports(report *explorer.ReportCollection, conf *scanConfig, cmd *cobra.Command) {
	// print the output using the specified output format
	r, err := reporter.New(conf.Output)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	r.IsIncognito = conf.IsIncognito

	if err = r.Print(report, os.Stdout); err != nil {
		log.Fatal().Err(err).Msg("failed to print")
	}
}
