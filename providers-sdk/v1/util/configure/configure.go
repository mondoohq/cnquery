// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/v11/logger"
	"golang.org/x/mod/modfile"
	"sigs.k8s.io/yaml"
)

type ProvidersConf struct {
	Builtin []string `json:"builtin"`
}

func init() {
	rootCmd.Flags().StringP("file", "f", "providers.yaml", "config file for providers")
	rootCmd.Flags().StringP("output", "o", "providers/builtin_dev.go", "output go-file for builtin dev providers")

	editProvidersCmd.Flags().StringP("file", "f", "providers.yaml", "config file for providers")
	rootCmd.AddCommand(editProvidersCmd)
}

func main() {
	logger.CliCompactLogger(logger.LogOutputWriter)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
		if err != nil {
			log.Fatal().Err(err).Str("path", confPath).Msg("failed to generate builtin go")
		}

		if err = os.WriteFile(outPath, builtinGo, 0o644); err != nil {
			log.Fatal().Err(err).Str("path", outPath).Msg("failed to write output")
		}
		log.Info().Str("path", outPath).Strs("providers", conf.Builtin).Msg("(1/3) configured builtin providers")

		buildProviders(conf.Builtin)
		log.Info().Strs("providers", conf.Builtin).Msg("(2/3) built providers")

		rewireDependencies(conf.Builtin)
		log.Info().Str("path", outPath).Strs("providers", conf.Builtin).Msg("(3/3) rewired dependencies/files")
	},
}

var editProvidersCmd = &cobra.Command{
	Use:   "edit-providers NAME1 NAME2 [-f config]",
	Short: "adds a provider to the config",
	Run: func(cmd *cobra.Command, args []string) {
		confPath, err := cmd.Flags().GetString("file")
		if err != nil {
			log.Fatal().Err(err).Msg("Can't get --file")
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

		if len(conf.Builtin) > 0 {
			log.Warn().Strs("providers", conf.Builtin).Msg("overwrite existing providers in config")
		}

		// set new providers
		conf.Builtin = args
		slices.Sort(conf.Builtin)
		conf.Builtin = slices.Compact(conf.Builtin)

		log.Info().Strs("providers", conf.Builtin).Msg("configured providers")

		raw, err = yaml.Marshal(conf)
		if err != nil {
			log.Fatal().Err(err).Str("path", confPath).Msg("failed to marshal updated config")
		}

		err = os.WriteFile(confPath, raw, 0o700)
		if err != nil {
			log.Fatal().Err(err).Str("path", confPath).Msg("failed to write config file")
		}
	},
}

func genBuiltinGo(conf ProvidersConf) ([]byte, error) {
	var imports string
	var infos string
	var configs string

	osIsActive := false
	for _, provider := range conf.Builtin {
		if provider == "os" {
			osIsActive = true
		}

		// imports cannot contain dashes
		trimProvider := strings.Replace(provider, "-", "", -1)

		imports += fmt.Sprintf("\t%sconf \"go.mondoo.com/cnquery/v11/providers/%s/config\"\n", trimProvider, provider)
		imports += fmt.Sprintf("\t%s \"go.mondoo.com/cnquery/v11/providers/%s/provider\"\n", trimProvider, provider)

		providerFilename := provider + ".resources.json"
		// We still have some special handling around the (builtin) os provider
		// which should be removed long-term...
		if provider == "os" {
			providerFilename = provider + "/resources/" + providerFilename
		}
		infos += fmt.Sprintf(
			"//go:embed %s\n"+
				"var %sInfo []byte\n",
			providerFilename, trimProvider)

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
`, trimProvider, trimProvider, trimProvider, trimProvider, provider, trimProvider, trimProvider)
	}

	if !osIsActive {
		imports += "\t// osconf \"go.mondoo.com/cnquery/v11/providers/os/config\"\n"
		imports += "\t// os \"go.mondoo.com/cnquery/v11/providers/os/provider\"\n"
		// IMPORTANT: Still special handling for the OS provider!
		// This is only relevant for the example. If we streamline the os provider
		// this will be removed.
		infos += "// //go:embed os/resources/os.resources.json\n" +
			"// var osInfo []byte\n"
		configs += `
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
`
	}

	res := fmt.Sprintf(template, imports, infos, configs)
	return format.Source([]byte(res))
}

const template = `// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
//
// This file is auto-generated by 'make providers/config'
// and configured via 'providers.yaml'. For example, you can inline via:
//
//   builtin: [os, aws, mondoo]

package providers

import (
	_ "embed"
%s)

%s

func init() {%s
}
`

func buildProviders(providers []string) {
	for i, provider := range providers {
		cmd := exec.Command("make", "providers/build/"+provider)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		log.Debug().Str("provider", provider).Msg("build provider " + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(providers)))
		if err := cmd.Run(); err != nil {
			fmt.Println(out.String())
			log.Error().Err(err).Str("provider", provider).Msg("failed to build provider")
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
	reBuiltinReplace = regexp.MustCompile(`replace go.mondoo.com/cnquery/v11/providers/.* => ./providers/.*`)
	reBuiltinDep     = regexp.MustCompile(`go.mondoo.com/cnquery/v11/providers/.*`)
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
		_, err := os.Stat("providers/" + provider + "/go.mod")
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				log.Info().Str("provider", provider).Msg("skipping provider without go.mod")
				continue
			} else {
				log.Fatal().Err(err).Str("provider", provider).Msg("failed to stat provider go.mod")
			}
		}

		goModContent, err := os.ReadFile("providers/" + provider + "/go.mod")
		if err != nil {
			log.Fatal().Err(err).Str("provider", provider).Msg("failed to read provider go.mod")
		}
		goMod, err := modfile.Parse(fmt.Sprintf("%s/go.mod", provider), goModContent, nil)
		if err != nil {
			log.Fatal().Err(err).Str("provider", provider).Msg("failed to parse provider go.mod")
		}

		// we don't care about the specific version for dev
		deps += "\n\tgo.mondoo.com/cnquery/v11/providers/" + provider + " v0.0.0"
		replace += "\nreplace go.mondoo.com/cnquery/v11/providers/" + provider + " => ./providers/" + provider
		// if the provider has any specific pinned replacements, we also add those to allow compiling
		for _, r := range goMod.Replace {
			// special case: we don't want to pull in provider's 'replace go.mondoo.com/cnquery/v11 => ../..' in.
			if r.Old.Path == "go.mondoo.com/cnquery/v11" {
				continue
			}
			// TODO: maybe we also use the modfile module to update the go.mod we're modifying
			replace += "\nreplace " + r.Old.Path + " => " + r.New.Path + " " + r.New.Version
		}
	}
	if deps != "" {
		raws = strings.Replace(raws, "require (", "require ("+deps, 1)
		raws = strings.Replace(raws, "module go.mondoo.com/cnquery/v11", "module go.mondoo.com/cnquery/v11\n"+replace, 1)
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
