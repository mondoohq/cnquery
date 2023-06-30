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
)

func init() {
	rootCmd.AddCommand(providersCmd)
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

func list() {
	list, err := providers.List()
	if err != nil {
		log.Error().Err(err).Msg("failed to list providers")
	}

	printProviders(list)
}

func printProviders(p providers.Providers) {
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
	for _, provider := range p {
		dir := filepath.Dir(provider.Path)
		paths[dir] = append(paths[dir], provider)
	}

	for path, list := range paths {
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

	if _, ok := paths[providers.SystemPath]; !ok {
		fmt.Println("")
		log.Info().Msg(providers.SystemPath + " has no providers")
	}
	if _, ok := paths[providers.HomePath]; !ok {
		fmt.Println("")
		log.Info().Msg(providers.HomePath + " has no providers")
	}

	fmt.Println()
}

func printProvider(p *providers.Provider) {
	conns := make([]string, len(p.Connectors))
	for i := range p.Connectors {
		conns[i] = theme.DefaultTheme.Secondary(p.Connectors[i].Name)
	}

	name := theme.DefaultTheme.Primary(p.Name)
	ps := strings.Join(conns, ", ")
	fmt.Println("  " + name + " provides: " + ps)
}
