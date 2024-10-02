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
		log.Info().Str("path", outPath).Strs("providers", conf.Providers()).Msg("(1/3) configured builtin providers")

		buildProviders(conf.Builtin)
		log.Info().Strs("providers", conf.Providers()).Msg("(2/3) built providers")

		rewireDependencies(conf.Builtin)
		log.Info().Str("path", outPath).Strs("providers", conf.Providers()).Msg("(3/3) rewired dependencies/files")
	},
}

var editProvidersCmd = &cobra.Command{
	Use:   "edit-providers NAME1 NAME2:REMOTE:GOPACKAGE [-f config]",
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
			log.Warn().Strs("providers", conf.Providers()).Msg("overwrite existing providers in config")
		}

		// set new providers
		for _, arg := range args {
			if !strings.Contains(arg, ":") {
				conf.Builtin = append(conf.Builtin, Builtin{Name: arg})
				continue
			}

			parts := strings.Split(arg, ":")
			if len(parts) != 3 {
				log.Fatal().Str("provider", arg).Msg("invalid provider format, must be NAME:REMOTE:GOPACKAGE")
			}

			conf.Builtin = append(conf.Builtin, Builtin{Name: parts[0], Remote: parts[1], GoPackage: parts[2]})
		}
		slices.SortFunc(conf.Builtin, func(a, b Builtin) int {
			return strings.Compare(a.Name, b.Name)
		})
		conf.Builtin = slices.Compact(conf.Builtin)

		log.Info().Strs("providers", conf.Providers()).Msg("configured providers")

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

	for _, provider := range conf.Builtin {
		// imports cannot contain dashes
		trimProvider := strings.Replace(provider.Name, "-", "", -1)
		imports += fmt.Sprintf("\t%sconf \"%s/config\"\n", trimProvider, provider.GoPackage)
		imports += fmt.Sprintf("\t%s \"%s/provider\"\n", trimProvider, provider.GoPackage)

		infos += fmt.Sprintf(
			"//go:embed %s.resources.json\n"+
				"var %sInfo []byte\n",
			provider.Name, trimProvider)

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
`, trimProvider, trimProvider, trimProvider, trimProvider, provider.Name, trimProvider, trimProvider)
	}

	res := fmt.Sprintf(template, imports, infos, configs)
	return format.Source([]byte(res))
}

const template = `// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
//
// This file is auto-generated by 'make providers/config'
// and configured via 'providers.yaml'; DO NOT EDIT.

package providers

import (
	_ "embed"
	// osconf "go.mondoo.com/cnquery/v11/providers/os/config"
	// os "go.mondoo.com/cnquery/v11/providers/os/provider"
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

func buildProviders(providers []Builtin) {
	for i, provider := range providers {
		cmd := exec.Command("make", "providers/build/"+provider.Name)
		if provider.Remote != "" {
			cmd = exec.Command("make", "provider/generate")
			cmd.Dir = provider.Remote
		}

		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		log.Debug().Str("provider", provider.Name).Msg("build provider " + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(providers)))
		if err := cmd.Run(); err != nil {
			fmt.Println(out.String())
			log.Error().Err(err).Str("provider", provider.Name).Msg("failed to build provider")
		}

		// inefficient copy...
		src := provider.Resource()
		raw, err := os.ReadFile(src)
		if err != nil {
			log.Fatal().Err(err).Str("src", src).Msg("failed to read resources json")
		}

		dst := provider.Dist()
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

func rewireDependencies(providers []Builtin) {
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
		goModPath := provider.GoMod()
		_, err := os.Stat(goModPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				log.Info().Str("provider", provider.Name).Msg("skipping provider without go.mod")
				continue
			} else {
				log.Fatal().Err(err).Str("provider", provider.Name).Msg("failed to stat provider go.mod")
			}
		}

		goModContent, err := os.ReadFile(goModPath)
		if err != nil {
			log.Fatal().Err(err).Str("provider", provider.Name).Msg("failed to read provider go.mod")
		}
		goMod, err := modfile.Parse(goModPath, goModContent, nil)
		if err != nil {
			log.Fatal().Err(err).Str("provider", provider.Name).Msg("failed to parse provider go.mod")
		}

		// we don't care about the specific version for dev
		deps += "\n\t" + provider.GoPackage + " v0.0.0"
		if provider.Remote != "" {
			replace += "\nreplace " + provider.GoPackage + " => " + provider.Remote
		} else {
			replace += "\nreplace " + provider.GoPackage + " => ./providers/" + provider.Name
		}
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
