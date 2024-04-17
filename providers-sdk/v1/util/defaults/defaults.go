// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/v11/logger"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
)

var rootCmd = &cobra.Command{
	Use:   "defaults [-o file]",
	Short: "configure provider defaults",
	Run: func(cmd *cobra.Command, args []string) {
		outPath, err := cmd.Flags().GetString("output")
		if err != nil {
			log.Fatal().Err(err).Msg("Can't get --output")
		}

		providers, err := getProviders()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get providers")
		}

		var configs []*plugin.Provider
		for name, path := range providers {
			provider, err := getConfig(name, path)
			if err != nil {
				log.Error().Err(err).Str("provider", name).Msg("failed to get provider config")
			} else {
				configs = append(configs, provider)
			}
		}

		sort.Slice(configs, func(i, j int) bool {
			return configs[i].Name < configs[j].Name
		})

		res, err := goGen(configs)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to generate go code for provider defaults")
		}

		if err = os.WriteFile(outPath, res, 0o644); err != nil {
			log.Fatal().Err(err).Str("path", outPath).Msg("failed to save file")
		} else {
			log.Info().Str("path", outPath).Msg("updated")
		}
	},
}

func goGen(configs []*plugin.Provider) ([]byte, error) {
	var body strings.Builder

	for i := range configs {
		conf := configs[i]

		var conns strings.Builder
		for j := range conf.Connectors {
			conn := conf.Connectors[j]
			conns.WriteString(fmt.Sprintf(connectorTemplate, conn.Name, conn.Use, conn.Short))
		}
		body.WriteString(fmt.Sprintf(providerTemplate, conf.Name, conf.Name, conf.ID, conf.ConnectionTypes, conns.String()))
	}

	res := fmt.Sprintf(template, body.String())
	return format.Source([]byte(res))
}

const template = `// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
//
// This file is auto-generated by 'make providers/defaults'

package providers

import "go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"

// DefaultProviders are useful when working in air-gapped environments
// to tell users what providers are used for common connections, when there
// is no other way to find out.
var DefaultProviders Providers = map[string]*Provider{
	%s
}
`

const providerTemplate = `
	%#v: {
		Provider: &plugin.Provider{
			Name:            %#v,
			ID:              %#v,
			ConnectionTypes: %#v,
			Connectors: []plugin.Connector{
				%s
			},
		},
	},
`

const connectorTemplate = `
				{
					Name:  %#v,
					Use:   %#v,
					Short: %#v,
				},
`

func getProviders() (map[string]string, error) {
	dir, err := os.ReadDir("./providers")
	if err != nil {
		return nil, err
	}

	res := map[string]string{}
	for i := range dir {
		entry := dir[i]
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		res[name] = filepath.Join("./providers", name)
	}
	return res, nil
}

func getConfig(providerName string, providerPath string) (*plugin.Provider, error) {
	log.Info().Str("provider", providerName).Msg("generate provider config")
	cmd := exec.Command("go", "run", "./gen/main.go", ".")
	cmd.Dir = providerPath
	_, err := cmd.Output()
	if err != nil {
		return nil, errors.New("failed to run:" + cmd.String() + " in " + providerPath)
	}

	raw, err := os.ReadFile(filepath.Join(providerPath, "dist", providerName+".json"))
	if err != nil {
		return nil, err
	}

	var res plugin.Provider
	err = json.Unmarshal(raw, &res)
	return &res, err
}

func init() {
	rootCmd.Flags().StringP("output", "o", "providers/defaults.go", "output go-file for defaults of providers")
}

func main() {
	logger.CliCompactLogger(logger.LogOutputWriter)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
