package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/cli/theme"
	"go.mondoo.com/cnquery/providers"
	"go.mondoo.com/cnquery/sortx"
)

func init() {
	rootCmd.AddCommand(providersCmd)
	providersCmd.AddCommand(listProvidersCmd)
	providersCmd.AddCommand(installProviderCmd)

	installProviderCmd.Flags().StringP("file", "f", "", "install a provider via a file")
}

var providersCmd = &cobra.Command{
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
	Use:    "install <NAME>",
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

		log.Fatal().Msg("cannot install providers by name yet")
	},
}

func installProviderFile(path string) {
	providers, err := providers.InstallFile(path, providers.InstallConf{
		Dst: providers.HomePath,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to install")
	}

	for i := range providers {
		provider := providers[i]
		log.Info().
			Str("version", provider.Version).
			Str("path", provider.Path).
			Msg("successfully installed " + provider.Name + " provider")
	}
}

func list() {
	list, err := providers.ListAll()
	if err != nil {
		log.Error().Err(err).Msg("failed to list providers")
	}

	for _, v := range list {
		if v.Path == "" {
			continue
		}
		if err := v.LoadJSON(); err != nil {
			log.Error().Err(err).
				Str("provider", v.Name).
				Str("path", v.Path).
				Msg("failed to load provider")
		}
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
		if printEmpty {
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
