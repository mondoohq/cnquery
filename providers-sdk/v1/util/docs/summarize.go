// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/resources"
)

func main() {
	if len(os.Args) <= 1 {
		log.Fatal().Msg("usage: summarize ./providers")
	}

	path := os.Args[1]
	providersDir, err := os.ReadDir(path)
	if err != nil {
		log.Fatal().Msg("failed to read folder " + path)
	}

	schemas := map[string]*resources.Schema{}
	for i := range providersDir {
		node := providersDir[i]
		if !node.IsDir() {
			continue
		}

		name := node.Name()
		path := filepath.Join(path, name)

		resourcesFile := filepath.Join(path, "dist", name+".resources.json")
		exists, err := exists(resourcesFile)
		if err != nil {
			log.Fatal().Err(err).Str("path", resourcesFile).Msg("can't determine if folder exists")
		}
		if !exists {
			log.Info().Msg("skipping " + name)
			continue
		}

		schema := providers.MustLoadSchemaFromFile(name, resourcesFile)
		schemas[node.Name()] = schema
	}

	combined := resources.Schema{Resources: map[string]*resources.ResourceInfo{}}
	for _, schema := range schemas {
		combined.Add(schema)
	}

	resources := combined.AllResources()

	fields := 0
	for _, resource := range resources {
		for _, v := range resource.Fields {
			if v.IsImplicitResource {
				continue
			}
			fields++
		}
	}

	fmt.Println()
	fmt.Printf("Total providers: %d\n", len(schemas))
	fmt.Printf("Total resources: %d\n", len(resources))
	fmt.Printf("Total fields:    %d\n", fields)
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
