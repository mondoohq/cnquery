// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
)

//go:embed template/*
var templates embed.FS

type config struct {
	Path                string
	ProviderID          string
	CamelcaseProviderID string
	ProviderName        string
	GoPackage           string
}

func main() {
	flags := pflag.NewFlagSet("", pflag.ContinueOnError)
	dir := flags.String("path", "", "path directory where you want to generate the files into")
	providerID := flags.String("provider-id", "", "provider id")
	providerName := flags.String("provider-name", "", "provider name")
	goPackage := flags.String("go-package", "", "go package")

	if err := flags.Parse(os.Args); err != nil {
		if err == pflag.ErrHelp {
			os.Exit(0)
		}
		log.Fatal().Err(err).Msg("error: could not parse flags")
	}

	if *dir == "" {
		log.Fatal().Msg("--path is required")
	}

	if *providerID == "" {
		log.Fatal().Msg("--provider-id is required")
	}

	if *providerName == "" {
		log.Fatal().Msg("--provider-name is required")
	}

	if *goPackage == "" {
		log.Fatal().Msg("--go-package is required")
	}

	err := os.MkdirAll(*dir, os.ModePerm)
	if err != nil {
		log.Fatal().Err(err).Msg("could not ensure the provided directory exists")
	}

	err = generateProvider(config{
		Path:                *dir,
		ProviderID:          *providerID,
		ProviderName:        *providerName,
		GoPackage:           *goPackage,
		CamelcaseProviderID: capitalize(*providerID),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("could not generate provider files")
	}
}

func capitalize(str string) string {
	runes := []rune(str)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func generateProvider(cfg config) error {
	return fs.WalkDir(templates, ".", func(sourceFile string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		if filepath.Ext(sourceFile) != ".template" {
			return nil
		}

		base, _ := filepath.Rel("template", sourceFile)
		fmt.Println("Render " + base)

		path := strings.TrimSuffix(base, ".template")
		path = strings.ReplaceAll(path, "providerid", cfg.ProviderID)

		rootDir := filepath.Join(cfg.Path, filepath.Dir(path))
		fmt.Println("Create dir " + rootDir)
		err = os.MkdirAll(rootDir, os.ModePerm)
		if err != nil {
			return err
		}

		destinationFile := filepath.Join(cfg.Path, path)
		fmt.Println("Render file" + destinationFile)

		input, err := fs.ReadFile(templates, sourceFile)
		if err != nil {
			return err
		}

		tmpl, err := template.New(destinationFile).Parse(string(input))
		if err != nil {
			return err
		}

		w, err := os.OpenFile(destinationFile, os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			fmt.Println("Error creating", destinationFile)
			return err
		}
		defer w.Close()

		err = tmpl.Execute(w, cfg)
		if err != nil {
			return err
		}

		return nil
	})
}
