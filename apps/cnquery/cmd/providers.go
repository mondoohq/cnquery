// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/v10/cli/theme"
	"go.mondoo.com/cnquery/v10/providers"
	"go.mondoo.com/cnquery/v10/utils/sortx"
)

func init() {
	rootCmd.AddCommand(ProvidersCmd)
	ProvidersCmd.AddCommand(listProvidersCmd)
	ProvidersCmd.AddCommand(installProviderCmd)

	installProviderCmd.Flags().StringP("file", "f", "", "install a provider via a file")
	installProviderCmd.Flags().String("url", "", "install a provider via URL")
}

var ProvidersCmd = &cobra.Command{
	Use:    "providers",
	Short:  "Providers add connectivity to all assets.",
	Long:   `Manage your providers. List and install new ones or update existing ones.`,
	PreRun: func(cmd *cobra.Command, args []string) {},
	Run: func(cmd *cobra.Command, args []string) {
		list()
	},
}

var listProvidersCmd = &cobra.Command{
	Use:    "list",
	Short:  "List all providers on the system.",
	Long:   "",
	PreRun: func(cmd *cobra.Command, args []string) {},
	Run: func(cmd *cobra.Command, args []string) {
		list()
	},
}

var installProviderCmd = &cobra.Command{
	Use:    "install <NAME[@VERSION]>",
	Short:  "Install or update a provider.",
	Long:   "",
	PreRun: func(cmd *cobra.Command, args []string) {},
	Run: func(cmd *cobra.Command, args []string) {
		// Explicit installs of files will ignore version recommendations.
		// So we just take them and roll with it.
		path, _ := cmd.Flags().GetString("file")
		if path != "" {
			installProviderFile(path)
			return
		}

		url, _ := cmd.Flags().GetString("url")
		if url != "" {
			installProviderUrl(url)
			return
		}

		if len(args) == 0 {
			log.Fatal().Msg("no provider specified, use the NAME[@VERSION] format to pass in a provider name")
		}

		// if no url or file is specified, we default to installing by name from the default upstream
		installProviderByName(args[0])
	},
}

func installProviderByName(name string) {
	parts := strings.Split(name, "@")
	if len(parts) > 2 {
		log.Fatal().Msg("invalid provider name")
	}
	name = parts[0]
	version := ""
	if len(parts) == 2 {
		// trim the v prefix, allowing users to specify both 9.0.0 and v9.0.0
		version = strings.TrimPrefix(parts[1], "v")
	}
	installed, err := providers.Install(name, version)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to install")
	}
	providers.PrintInstallResults([]*providers.Provider{installed})
}

func installProviderUrl(u string) {
	if i := strings.Index(u, "://"); i == -1 {
		u = "http://" + u
	}
	uUrl, err := url.Parse(u)
	if err != nil {
		log.Fatal().Err(err).Msg("invalid url")
	}

	res, err := http.Get(uUrl.String())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to install")
	}

	installed, err := providers.InstallIO(res.Body, providers.InstallConf{
		Dst: providers.HomePath,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to install")
	}
	providers.PrintInstallResults(installed)
}

func installProviderFile(path string) {
	installed, err := providers.InstallFile(path, providers.InstallConf{
		Dst: providers.HomePath,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to install")
	}
	providers.PrintInstallResults(installed)
}

func list() {
	list, err := providers.ListAll()
	if err != nil {
		log.Error().Err(err).Msg("failed to list providers")
	}

	printProviders(list)
}

func printProviders(p []*providers.Provider) {
	if len(p) == 0 {
		log.Info().Msg("No providers found.")
		fmt.Println("No providers found.")
		if providers.SystemPath == "" && providers.HomePath == "" {
			fmt.Println("No paths for providers detected.")
		} else {
			fmt.Println("Was checking: " + providers.SystemPath)
		}
	}

	paths := map[string][]*providers.Provider{}
	for i := range p {
		provider := p[i]
		if provider.Path == "" {
			paths["builtin"] = append(paths["builtin"], provider)
			continue
		}
		dir := filepath.Dir(provider.Path)
		paths[dir] = append(paths[dir], provider)
	}

	printProviderPath("builtin", paths["builtin"], false)
	printProviderPath(providers.HomePath, paths[providers.HomePath], true)
	printProviderPath(providers.SystemPath, paths[providers.SystemPath], true)
	delete(paths, "builtin")
	delete(paths, providers.HomePath)
	delete(paths, providers.SystemPath)

	keys := sortx.Keys(paths)
	for _, path := range keys {
		printProviderPath(path, paths[path], true)
	}

	fmt.Println()
}

func printProviderPath(path string, list []*providers.Provider, printEmpty bool) {
	if list == nil {
		if printEmpty && path != "" {
			fmt.Println("")
			log.Info().Msg(path + " has no providers")
		}
		return
	}

	fmt.Println()
	log.Info().Msg(path + " (found " + strconv.Itoa(len(list)) + " providers)")
	fmt.Println()

	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})

	for i := range list {
		printProvider(list[i])
	}
}

func printProvider(p *providers.Provider) {
	conns := make([]string, len(p.Connectors))
	for i := range p.Connectors {
		conns[i] = theme.DefaultTheme.Secondary(p.Connectors[i].Name)
	}

	name := theme.DefaultTheme.Primary(p.Name)
	supports := ""
	if len(conns) != 0 {
		supports = " with connectors: " + strings.Join(conns, ", ")
	}

	fmt.Println("  " + name + " " + p.Version + supports)
}
