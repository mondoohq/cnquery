// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/logger"
	"sigs.k8s.io/yaml"
)

type ProvidersConf struct {
	Builtin []string `json:"builtin"`
}

var rootCmd = &cobra.Command{
	Use:   "configure [-f config] [-o file]",
	Short: "configure providers for cnquery",
	Run: func(cmd *cobra.Command, args []string) {
		confPath, err := cmd.Flags().GetString("file")
		if err != nil {
			log.Fatal().Err(err).Msg("Can't get --file")
		}
		outPath, err := cmd.Flags().GetString("output")
		if err != nil {
			log.Fatal().Err(err).Msg("Can't get --output")
		}

		raw, err := os.ReadFile(confPath)
		if err != nil {
			log.Fatal().Err(err).Str("path", confPath).Msg("failed to read config file")
		}

		var conf ProvidersConf
		err = yaml.Unmarshal(raw, &conf)
		if err != nil {
			log.Fatal().Err(err).Str("path", confPath).Msg("failed to parse config file")
		}

		builtinGo, err := genBuiltinGo(conf)

		if err = os.WriteFile(outPath, []byte(builtinGo), 0o644); err != nil {
			log.Fatal().Err(err).Str("path", outPath).Msg("failed to write output")
		}
		log.Info().Str("path", outPath).Strs("providers", conf.Builtin).Msg("(1/3) configured builtin providers")

		buildProviders(conf.Builtin)
		log.Info().Strs("providers", conf.Builtin).Msg("(2/3) built providers")

		rewireDependencies(conf.Builtin)
		log.Info().Str("path", outPath).Strs("providers", conf.Builtin).Msg("(3/3) rewired dependencies/files")
	},
}

func genBuiltinGo(conf ProvidersConf) (string, error) {
	var imports string
	var infos string
	var configs string

	for _, provider := range conf.Builtin {
		imports += fmt.Sprintf("\t%sconf \"go.mondoo.com/cnquery/providers/%s/config\"\n", provider, provider)
		imports += fmt.Sprintf("\t%s \"go.mondoo.com/cnquery/providers/%s/provider\"\n", provider, provider)
		infos += fmt.Sprintf(
			"//go:embed %s.resources.json\n"+
				"var %sInfo []byte\n",
			provider, provider)
		configs += fmt.Sprintf(`
	builtinProviders[%sconf.Config.ID] = &builtinProvider{
		Runtime: &RunningProvider{
			Name:     %sconf.Config.Name,
			ID:       %sconf.Config.ID,
			Plugin:   %s.Init(),
			Schema:   MustLoadSchema("%s", %sInfo),
			isClosed: false,
		},
		Config: &%sconf.Config,
	}
`, provider, provider, provider, provider, provider, provider, provider)
	}

	return fmt.Sprintf(template, imports, infos, configs), nil
}

const template = `// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
//
// This file is auto-generated by 'make providers/config'
// and configured via 'providers.yaml'

package providers

import (
	_ "embed"
	// osconf "go.mondoo.com/cnquery/providers/os/config"
	// os "go.mondoo.com/cnquery/providers/os/provider"
%s)

// //go:embed os/resources/os.resources.json
// var osInfo []byte

%s

func init() {
	// builtinProviders[osconf.Config.ID] = &builtinProvider{
	// 	Runtime: &RunningProvider{
	// 		Name:     osconf.Config.Name,
	// 		ID:       osconf.Config.ID,
	// 		Plugin:   os.Init(),
	// 		Schema:   MustLoadSchema("os", osInfo),
	// 		isClosed: false,
	// 	},
	// 	Config: &osconf.Config,
	// }
%s
}
`

func buildProviders(providers []string) {
	for i, provider := range providers {
		cmd := exec.Command("make", "providers/build/"+provider)
		log.Debug().Str("provider", provider).Msg("build provider " + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(providers)))
		if err := cmd.Run(); err != nil {
			log.Fatal().Err(err).Str("provider", provider).Msg("failed to build provider")
		}

		// inefficient copy...
		src := "providers/" + provider + "/resources/" + provider + ".resources.json"
		raw, err := os.ReadFile(src)
		if err != nil {
			log.Fatal().Err(err).Str("src", src).Msg("failed to read resources json")
		}

		dst := "providers/" + provider + ".resources.json"
		err = os.WriteFile(dst, raw, 0o644)
		if err != nil {
			log.Fatal().Err(err).Str("dst", dst).Msg("failed to write resources json")
		}
	}
}

var (
	reBuiltinReplace = regexp.MustCompile(`replace go.mondoo.com/cnquery/providers/.* => ./providers/.*`)
	reBuiltinDep     = regexp.MustCompile(`go.mondoo.com/cnquery/providers/.*`)
)

func rewireDependencies(providers []string) {
	raw, err := os.ReadFile("go.mod")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read go.mod")
	}

	raws := string(raw)
	raws = reBuiltinReplace.ReplaceAllString(raws, "")
	raws = reBuiltinDep.ReplaceAllString(raws, "")

	deps := ""
	replace := ""
	for _, provider := range providers {
		// we don't care about the specific version for dev
		deps += "\n\tgo.mondoo.com/cnquery/providers/" + provider + " v0.0.0"
		replace += "\nreplace go.mondoo.com/cnquery/providers/" + provider + " => ./providers/" + provider
	}
	if deps != "" {
		raws = strings.Replace(raws, "require (", "require ("+deps, 1)
		raws = strings.Replace(raws, "module go.mondoo.com/cnquery", "module go.mondoo.com/cnquery\n"+replace, 1)
	}

	err = os.WriteFile("go.mod", []byte(raws), 0o644)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to write go.mod")
	}

	cmd := exec.Command("go", "mod", "tidy")
	log.Debug().Msg("go mod tidy")
	if err := cmd.Run(); err != nil {
		log.Fatal().Err(err).Msg("failed to go mod tidy")
	}
}

func init() {
	rootCmd.Flags().StringP("file", "f", "providers.yaml", "config file for providers")
	rootCmd.Flags().StringP("output", "o", "providers/builtin_dev.go", "output go-file for builtin dev providers")
}

func main() {
	logger.CliCompactLogger(logger.LogOutputWriter)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
