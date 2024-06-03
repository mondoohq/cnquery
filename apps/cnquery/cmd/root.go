// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v11/cli/config"
	cli_errors "go.mondoo.com/cnquery/v11/cli/errors"
	"go.mondoo.com/cnquery/v11/cli/execruntime"
	"go.mondoo.com/cnquery/v11/cli/inventoryloader"
	cliproviders "go.mondoo.com/cnquery/v11/cli/providers"
	"go.mondoo.com/cnquery/v11/cli/theme"
	"go.mondoo.com/cnquery/v11/logger"
	"go.mondoo.com/cnquery/v11/providers"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream"
)

const (
	askForPasswordValue = ">passwordisnotset<"
	rootCmdDesc         = "cnquery is a cloud-native tool for querying your entire infrastructure.\n"

	// we send a 78 exit code to prevent systemd service from restart
	ConfigurationErrorCode = 78
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cnquery",
	Short: "cnquery CLI",
	Long:  theme.DefaultTheme.Landing + "\n\n" + rootCmdDesc,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initLogger()
	},
}

func BuildRootCmd() (*cobra.Command, error) {
	err := cliproviders.AttachCLIs(
		rootCmd,
		&cliproviders.Command{
			Command: shellCmd,
			Run:     shellRun,
			Action:  "Interactive shell with ",
		},
		&cliproviders.Command{
			Command: RunCmd,
			Run:     RunCmdRun,
			Action:  "Run a query with ",
		},
		&cliproviders.Command{
			Command: scanCmd,
			Run:     scanCmdRun,
			Action:  "Scan ",
		},
		&cliproviders.Command{
			Command:             sbomCmd,
			Run:                 sbomCmdRun,
			Action:              "Collect a software bill of materials (SBOM) for ",
			SupportedConnectors: []string{"docker", "container", "filesystem", "local", "ssh", "vagrant", "winrm", "sbom"},
		},
	)
	return rootCmd, err
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	_, err := BuildRootCmd()
	if err != nil {
		log.Error().Msg(err.Error())
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		if cErr, ok := err.(*cli_errors.CommandError); ok {
			if cErr.HasError() {
				log.Error().Msg(err.Error())
			}
			os.Exit(cErr.ExitCode())
		}

		log.Error().Msg(err.Error())
		os.Exit(1)
	}
}

func init() {
	// NOTE: we need to call this super early, otherwise the CLI color output on Windows is broken for the first lines
	// since the log instance is already initialized, replace default zerolog color output with our own
	// use color logger by default
	logger.CliCompactLogger(logger.LogOutputWriter)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// TODO harmonize with initLogger, which is called later and attached to the command
	// here we set the log level only by environment variable
	envLevel, ok := logger.GetEnvLogLevel()
	if ok {
		logger.Set(envLevel)
	}

	config.DefaultConfigFile = "mondoo.yml"

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().String("log-level", "info", "Set log level: error, warn, info, debug, trace")
	rootCmd.PersistentFlags().String("api-proxy", "", "Set proxy for communications with Mondoo Platform API")
	rootCmd.PersistentFlags().Bool("auto-update", true, "Enable automatic provider installation and update")
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("api_proxy", rootCmd.PersistentFlags().Lookup("api-proxy"))
	viper.BindPFlag("auto_update", rootCmd.PersistentFlags().Lookup("auto-update"))
	viper.BindEnv("features")

	config.Init(rootCmd)
}

func initLogger() {
	// environment variables always over-write custom flags
	envLevel, ok := logger.GetEnvLogLevel()
	if ok {
		logger.Set(envLevel)
		return
	}

	// retrieve log-level from flags
	level := viper.GetString("log-level")
	if v := viper.GetBool("verbose"); v {
		level = "debug"
	}
	logger.Set(level)
}

var reMdName = regexp.MustCompile(`/([^/]+)\.md$`)

func GenerateMarkdown(dir string) error {
	rootCmd.DisableAutoGenTag = true

	// We need to remove our fancy logo from the markdown output,
	// since it messes with the formatting.
	rootCmd.Long = rootCmdDesc

	files := []string{}
	err := doc.GenMarkdownTreeCustom(rootCmd, dir, func(s string) string {
		files = append(files, s)

		titles := reMdName.FindStringSubmatch(s)
		if len(titles) == 0 {
			return ""
		}
		title := strings.ReplaceAll(titles[1], "_", " ")

		return "---\n" +
			"id: " + titles[1] + "\n" +
			"title: " + title + "\n" +
			"---\n\n"
	}, func(s string) string { return s })
	if err != nil {
		return err
	}

	// we need to remove the first headline, since it is doubled with the
	// headline from the ID. Really annoying, all this needs a rewrite.
	for i := range files {
		file := files[i]
		raw, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		if !strings.HasPrefix(string(raw), "---\nid:") {
			continue
		}

		start := strings.Index(string(raw), "\n## ")
		if start < 0 {
			continue
		}

		end := start
		for i := start + 3; i < len(raw); i++ {
			if raw[i] == '\n' {
				end = i
				break
			}
		}

		res := append(raw[0:start], raw[end:]...)
		err = os.WriteFile(file, res, 0o644)
		if err != nil {
			return err
		}
	}

	return nil
}

func getCobraScanConfig(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) (*scanConfig, error) {
	opts, err := config.Read()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load config")
	}

	config.DisplayUsedConfig()

	props := viper.GetStringMapString("props")

	// merge the config and the user-provided annotations with the latter having precedence
	optAnnotations := opts.Annotations
	if optAnnotations == nil {
		optAnnotations = map[string]string{}
	}

	assetName := viper.GetString("asset-name")
	if assetName != "" && cliRes.Asset != nil {
		cliRes.Asset.Name = assetName
	}

	traceId := viper.GetString("trace-id")
	if traceId != "" && cliRes.Asset != nil {
		cliRes.Asset.TraceId = traceId
	}

	inv, err := inventoryloader.ParseOrUse(cliRes.Asset, viper.GetBool("insecure"), optAnnotations)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse inventory")
	}

	// TODO: We currently deduplicate this here because it leads to errors down the line,
	// if the same querypack is added more than once. Fix this properly downstream.
	querypackPaths := dedupe(viper.GetStringSlice("querypack-bundle"))

	conf := scanConfig{
		Features:       opts.GetFeatures(),
		IsIncognito:    viper.GetBool("incognito"),
		Inventory:      inv,
		QueryPackPaths: querypackPaths,
		QueryPackNames: viper.GetStringSlice("querypacks"),
		Props:          props,
		runtime:        runtime,
	}

	// determine the output format
	output := viper.GetString("output")
	// --json takes precedence
	if ok := viper.GetBool("json"); ok {
		output = "json"
	}
	conf.Output = output

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
		conf.Inventory.ApplyCategory(inventory.AssetCategory_CATEGORY_CICD)
	}

	serviceAccount := opts.GetServiceCredential()
	if serviceAccount != nil {
		log.Info().Msg("using service account credentials")
		conf.runtime.UpstreamConfig = &upstream.UpstreamConfig{
			SpaceMrn:    opts.GetParentMrn(),
			ApiEndpoint: opts.UpstreamApiEndpoint(),
			ApiProxy:    opts.APIProxy,
			Incognito:   conf.IsIncognito,
			Creds:       serviceAccount,
		}
		providers.DefaultRuntime().UpstreamConfig = conf.runtime.UpstreamConfig
	} else {
		log.Warn().Msg("No credentials provided. Switching to --incognito mode.")
		conf.IsIncognito = true
	}

	if len(conf.QueryPackPaths) > 0 && !conf.IsIncognito {
		log.Warn().Msg("Scanning with local bundles will switch into --incognito mode by default. Your results will not be sent upstream.")
		conf.IsIncognito = true
	}

	// print headline when its not printed to yaml
	if output == "" {
		fmt.Fprintln(os.Stdout, theme.DefaultTheme.Welcome)
	}

	return &conf, nil
}
