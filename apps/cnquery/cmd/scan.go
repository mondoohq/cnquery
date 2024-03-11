// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v10"
	"go.mondoo.com/cnquery/v10/cli/reporter"
	"go.mondoo.com/cnquery/v10/explorer"
	"go.mondoo.com/cnquery/v10/explorer/scan"
	"go.mondoo.com/cnquery/v10/mqlc"
	"go.mondoo.com/cnquery/v10/providers"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
)

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.Flags().StringP("output", "o", "compact", "Set output format: "+reporter.AllFormats())
	scanCmd.Flags().BoolP("json", "j", false, "Run the query and return the object in a JSON structure.")
	scanCmd.Flags().String("platform-id", "", "Select a specific target asset by providing its platform ID.")

	scanCmd.Flags().String("inventory-file", "", "Set the path to the inventory file.")
	scanCmd.Flags().Bool("inventory-ansible", false, "Set the inventory format to Ansible.")
	scanCmd.Flags().Bool("inventory-domainlist", false, "Set the inventory format to domain list.")

	// bundles, packs & incognito mode
	scanCmd.Flags().Bool("incognito", false, "Run in incognito mode. Do not report scan results to  Mondoo Platform.")
	scanCmd.Flags().StringSlice("querypack", nil, "Set the query packs to execute. This requires `querypack-bundle`. You can specify multiple UIDs.")
	scanCmd.Flags().StringSliceP("querypack-bundle", "f", nil, "Path to local query pack file")
	// flag completion command
	scanCmd.RegisterFlagCompletionFunc("querypack", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getQueryPacksForCompletion(), cobra.ShellCompDirectiveDefault
	})
	scanCmd.Flags().String("asset-name", "", "User-override for the asset name")
	scanCmd.Flags().StringToString("annotation", nil, "Add an annotation to the asset.") // user-added, editable
	scanCmd.Flags().StringToString("props", nil, "Custom values for properties")

	// v6 should make detect-cicd and category flag public
	scanCmd.Flags().Bool("detect-cicd", true, "Try to detect CI/CD environments. If detected, set the asset category to 'cicd'.")
	scanCmd.Flags().String("category", "inventory", "Set the category for the assets to 'inventory|cicd'.")
	scanCmd.Flags().MarkHidden("category")
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan assets with one or more query packs",
	Long: `
This command scans an asset using a query pack. For example, you can scan
the local system with its pre-configured query pack:

		$ cnquery scan local

To manually configure a query pack, use this:

		$ cnquery scan local -f bundle.mql.yaml --incognito

`,
	PreRun: func(cmd *cobra.Command, args []string) {
		// Special handling for users that want to see what output options are
		// available. We have to do this before printing the help because we
		// don't have a target connection or provider.
		output, _ := cmd.Flags().GetString("output")
		if output == "help" {
			fmt.Println("Available output formats: " + reporter.AllFormats())
			os.Exit(0)
		}

		_ = viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))
		_ = viper.BindPFlag("inventory-file", cmd.Flags().Lookup("inventory-file"))
		_ = viper.BindPFlag("inventory-ansible", cmd.Flags().Lookup("inventory-ansible"))
		_ = viper.BindPFlag("inventory-domainlist", cmd.Flags().Lookup("inventory-domainlist"))
		_ = viper.BindPFlag("querypack-bundle", cmd.Flags().Lookup("querypack-bundle"))
		_ = viper.BindPFlag("detect-cicd", cmd.Flags().Lookup("detect-cicd"))
		_ = viper.BindPFlag("asset-name", cmd.Flags().Lookup("asset-name"))
		_ = viper.BindPFlag("category", cmd.Flags().Lookup("category"))

		_ = viper.BindPFlag("annotations", cmd.Flags().Lookup("annotations"))
		_ = viper.BindPFlag("props", cmd.Flags().Lookup("props"))

		// for all assets
		_ = viper.BindPFlag("incognito", cmd.Flags().Lookup("incognito"))
		_ = viper.BindPFlag("insecure", cmd.Flags().Lookup("insecure"))
		_ = viper.BindPFlag("querypacks", cmd.Flags().Lookup("querypack"))
		_ = viper.BindPFlag("sudo.active", cmd.Flags().Lookup("sudo"))
		_ = viper.BindPFlag("record", cmd.Flags().Lookup("record"))

		_ = viper.BindPFlag("output", cmd.Flags().Lookup("output"))
		_ = viper.BindPFlag("json", cmd.Flags().Lookup("json"))
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return []string{"yml", "yaml", "json"}, cobra.ShellCompDirectiveFilterFileExt
		}
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	},
	// we have to initialize an empty run so it shows up as a runnable command in --help
	Run: func(cmd *cobra.Command, args []string) {},
}

var scanCmdRun = func(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) {
	conf, err := getCobraScanConfig(cmd, runtime, cliRes)
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

	if report != nil && len(report.Errors) > 0 {
		os.Exit(1)
	}
}

// helper method to retrieve the list of query packs for autocomplete
func getQueryPacksForCompletion() []string {
	querypackList := []string{}

	// TODO: autocompletion
	sort.Strings(querypackList)

	return querypackList
}

type scanConfig struct {
	Features       cnquery.Features
	Inventory      *inventory.Inventory
	Output         string
	QueryPackPaths []string
	QueryPackNames []string
	Props          map[string]string
	Bundle         *explorer.Bundle
	runtime        *providers.Runtime

	IsIncognito bool
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

		conf := mqlc.NewConfig(c.runtime.Schema(), cnquery.DefaultFeatures)
		_, err = bundle.CompileExt(context.Background(), explorer.BundleCompileConf{
			CompilerConfig: conf,
			// We don't care about failing queries for local runs. We may only
			// process a subset of all the queries in the bundle. When we receive
			// things from the server, upstream can filter things for us. But running
			// them locally requires us to do it in here.
			RemoveFailing: true,
		})
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
	if config.runtime.UpstreamConfig != nil {
		opts = append(opts, scan.WithUpstream(config.runtime.UpstreamConfig))
	}
	opts = append(opts, scan.WithRecording(config.runtime.Recording()))

	scanner := scan.NewLocalScanner(opts...)
	ctx := cnquery.SetFeatures(context.Background(), config.Features)

	if config.IsIncognito {
		return scanner.RunIncognito(
			ctx,
			&scan.Job{
				Inventory:        config.Inventory,
				Bundle:           config.Bundle,
				QueryPackFilters: config.QueryPackNames,
				Props:            config.Props,
			})
	}
	return scanner.Run(
		ctx,
		&scan.Job{
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

func dedupe[T string | int](sliceList []T) []T {
	allKeys := make(map[T]bool)
	list := []T{}
	for _, item := range sliceList {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}
