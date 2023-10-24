// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/lr"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/lr/docs"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/resources"
	"sigs.k8s.io/yaml"
)

func init() {
	markdownCmd.Flags().String("pack-name", "", "name of the resource pack")
	markdownCmd.Flags().String("description", "", "description of the resource pack")
	markdownCmd.Flags().String("docs-file", "", "optional docs yaml to enrich the resource information")
	markdownCmd.Flags().StringP("output", "o", ".build", "optional docs yaml to enrich the resource information")
	rootCmd.AddCommand(markdownCmd)
}

const frontMatterTemplate = `---
title: {{ .PackName }} Resource Pack - MQL Resources
id: {{ .ID }}.pack
sidebar_label: {{ .PackName }} Resource Pack
displayed_sidebar: MQL
description: {{ .Description }}
---
`

var markdownCmd = &cobra.Command{
	Use:   "markdown",
	Short: "generates markdown files",
	Long:  `parse an LR file and generates a markdown file`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		raw, err := os.ReadFile(args[0])
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}

		outputDir, err := cmd.Flags().GetString("output")
		if err != nil {
			log.Fatal().Err(err).Msg("no output directory provided")
		}

		res, err := lr.Parse(string(raw))
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}

		schema, err := lr.Schema(res)
		if err != nil {
			log.Error().Err(err).Msg("failed to generate schema")
		}

		var lrDocsData docs.LrDocs
		docsFilepath, _ := cmd.Flags().GetString("docs-file")
		_, err = os.Stat(docsFilepath)
		if err == nil {
			content, err := os.ReadFile(docsFilepath)
			if err != nil {
				log.Fatal().Err(err).Msg("could not read file " + docsFilepath)
			}
			err = yaml.Unmarshal(content, &lrDocsData)
			if err != nil {
				log.Fatal().Err(err).Msg("could not load yaml data")
			}

			log.Info().Int("resources", len(lrDocsData.Resources)).Msg("loaded docs from " + docsFilepath)
		} else {
			log.Info().Msg("no docs file provided")
		}

		// to ensure we generate the same markdown, we sort the resources first
		sort.SliceStable(res.Resources, func(i, j int) bool {
			return res.Resources[i].ID < res.Resources[j].ID
		})

		// generate resource map for hyperlink generation and table of content
		resourceHrefMap := map[string]bool{}
		for i := range res.Resources {
			resource := res.Resources[i]
			resourceHrefMap[resource.ID] = true
		}

		// render all resources incl. fields and examples
		r := &lrSchemaRenderer{
			resourceHrefMap: resourceHrefMap,
		}

		// render readme
		packName, _ := cmd.Flags().GetString("pack-name")
		err = os.MkdirAll(outputDir, 0o755)
		if err != nil {
			log.Fatal().Err(err).Msg("could not create directory: " + outputDir)
		}
		description, _ := cmd.Flags().GetString("description")
		err = os.MkdirAll(outputDir, 0o755)
		if err != nil {
			log.Fatal().Err(err).Msg("could not create directory: " + outputDir)
		}
		err = os.WriteFile(filepath.Join(outputDir, "README.md"), []byte(r.renderToc(packName, description, res.Resources, schema)), 0o600)
		if err != nil {
			log.Fatal().Err(err).Msg("could not write file")
		}

		for i := range res.Resources {
			resource := res.Resources[i]
			var docs *docs.LrDocsEntry
			var ok bool
			if lrDocsData.Resources != nil {
				docs, ok = lrDocsData.Resources[resource.ID]
				if !ok {
					log.Warn().Msg("no docs found for resource " + resource.ID)
				}
			}

			err = os.WriteFile(filepath.Join(outputDir, strings.ToLower(resource.ID+".md")), []byte(r.renderResourcePage(resource, schema, docs)), 0o600)
			if err != nil {
				log.Fatal().Err(err).Msg("could not write file")
			}
		}
	},
}

var reNonID = regexp.MustCompile(`[^A-Za-z0-9-]+`)

type lrSchemaRenderer struct {
	resourceHrefMap map[string]bool
}

func toID(s string) string {
	s = reNonID.ReplaceAllString(s, ".")
	s = strings.ToLower(s)
	return strings.Trim(s, ".")
}

func (l *lrSchemaRenderer) renderToc(packName string, description string, resources []*lr.Resource, schema *resources.Schema) string {
	builder := &strings.Builder{}

	// render front matter
	tpl, _ := template.New("frontmatter").Parse(frontMatterTemplate)
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	err := tpl.Execute(writer, struct {
		PackName    string
		Description string
		ID          string
	}{
		PackName:    packName,
		Description: description,
		ID:          toID(packName),
	})
	if err != nil {
		panic(err)
	}
	writer.Flush()
	builder.WriteString(buf.String())
	builder.WriteString("\n")

	// render content
	builder.WriteString("# Mondoo " + packName + " Resource Pack Reference\n\n")
	builder.WriteString("In this pack:\n\n")
	rows := [][]string{}

	for i := range resources {
		resource := resources[i]
		rows = append(rows, []string{"[" + resource.ID + "](" + mdRef(resource.ID) + ")", strings.Join(sanitizeComments([]string{schema.Resources[resource.ID].Title}), " ")})
	}

	table := tablewriter.NewWriter(builder)
	table.SetHeader([]string{"ID", "Description"})
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.SetAutoWrapText(false)
	table.AppendBulk(rows)
	table.Render()
	builder.WriteString("\n")

	return builder.String()
}

// trimColon removes any : from the string since colons are not allowed in markdown front matter
func trimColon(s string) string {
	return strings.ReplaceAll(s, ":", "")
}

func (l *lrSchemaRenderer) renderResourcePage(resource *lr.Resource, schema *resources.Schema, docs *docs.LrDocsEntry) string {
	builder := &strings.Builder{}

	builder.WriteString("---\n")
	builder.WriteString("title: " + resource.ID + "\n")
	builder.WriteString("id: " + resource.ID + "\n")
	builder.WriteString("sidebar_label: " + resource.ID + "\n")
	builder.WriteString("displayed_sidebar: MQL\n")

	headerDesc := strings.Join(sanitizeComments([]string{schema.Resources[resource.ID].Title}), " ")
	if headerDesc != "" {
		builder.WriteString("description: " + trimColon(headerDesc) + "\n")
	}
	builder.WriteString("---\n")
	builder.WriteString("\n")

	builder.WriteString("# ")
	builder.WriteString(resource.ID)
	builder.WriteString("\n\n")

	if docs != nil && docs.Platform != nil && (len(docs.Platform.Name) > 0 || len(docs.Platform.Family) > 0) {
		builder.WriteString("**Supported platform**\n\n")
		for r := range docs.Platform.Name {
			builder.WriteString(fmt.Sprintf("- %s", docs.Platform.Name[r]))
			builder.WriteString("\n")
		}
		for r := range docs.Platform.Family {
			builder.WriteString(fmt.Sprintf("- %s", docs.Platform.Name[r]))
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	if docs != nil && len(docs.Maturity) > 0 {
		builder.WriteString("**Maturity**\n\n")
		builder.WriteString(docs.Maturity)
		builder.WriteString("\n\n")
	}

	if schema.Resources[resource.ID].Title != "" {
		builder.WriteString("**Description**\n\n")
		builder.WriteString(strings.Join(sanitizeComments([]string{schema.Resources[resource.ID].Title}), "\n"))
		builder.WriteString("\n\n")
	}

	if docs != nil && docs.Docs != nil && docs.Docs.Description != "" {
		builder.WriteString(docs.Docs.Description)
		builder.WriteString("\n\n")
	}

	inits := resource.GetInitFields()
	// generate the constructor
	if len(inits) > 0 {
		builder.WriteString("**Init**\n\n")
		for j := range inits {
			init := inits[j]

			for a := range init.Args {
				arg := init.Args[a]
				builder.WriteString(resource.ID + "(" + arg.ID + " " + renderLrType(arg.Type, l.resourceHrefMap) + ")")
				builder.WriteString("\n")
			}
		}
		builder.WriteString("\n")
	}

	if resource.ListType != nil {
		builder.WriteString("**List**\n\n")
		builder.WriteString("[]" + resource.ListType.Type.Type)
		builder.WriteString("\n\n")
	}

	basicFields := []*lr.BasicField{}
	comments := [][]string{}
	for _, f := range resource.Body.Fields {
		if f.BasicField != nil {
			basicFields = append(basicFields, f.BasicField)
			comments = append(comments, f.Comments)
		}
	}
	// generate the fields markdown table
	// NOTE: list types may not have any fields
	if len(basicFields) > 0 {
		builder.WriteString("**Fields**\n\n")
		rows := [][]string{}

		for k := range basicFields {
			field := basicFields[k]
			rows = append(rows, []string{
				field.ID, renderLrType(field.Type, l.resourceHrefMap),
				strings.Join(sanitizeComments(comments[k]), ", "),
			})
		}

		table := tablewriter.NewWriter(builder)
		table.SetHeader([]string{"ID", "Type", "Description"})
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")
		table.SetAutoWrapText(false)
		table.AppendBulk(rows)
		table.Render()
		builder.WriteString("\n")
	}

	if docs != nil && len(docs.Snippets) > 0 {
		builder.WriteString("**Examples**\n\n")
		for si := range docs.Snippets {
			snippet := docs.Snippets[si]
			builder.WriteString(snippet.Title)
			builder.WriteString("\n\n")
			builder.WriteString("```coffee\n")
			builder.WriteString(strings.TrimSpace(snippet.Query))
			builder.WriteString("\n```\n\n")
		}
		builder.WriteString("\n")
	}

	if docs != nil && len(docs.Resources) > 0 {
		builder.WriteString("**Resources**\n\n")
		for r := range docs.Resources {
			builder.WriteString(fmt.Sprintf("- [%s](%s)", docs.Resources[r].Title, docs.Resources[r].Url))
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	if docs != nil && len(docs.Refs) > 0 {
		builder.WriteString("**References**\n\n")
		for r := range docs.Refs {
			builder.WriteString(fmt.Sprintf("- [%s](%s)", docs.Refs[r].Title, docs.Refs[r].Url))
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

func anchore(name string) string {
	name = strings.Replace(name, ".", "", -1)
	return strings.ToLower(name)
}

func mdRef(name string) string {
	return strings.ToLower(name) + ".md"
}

func renderLrType(t lr.Type, resourceHrefMap map[string]bool) string {
	switch {
	case t.SimpleType != nil:
		_, ok := resourceHrefMap[t.SimpleType.Type]
		if ok {
			return "[" + t.SimpleType.Type + "](" + mdRef(t.SimpleType.Type) + ")"
		}
		return t.SimpleType.Type
	case t.ListType != nil:
		// we need a space between [] and the link, otherwise some markdown link parsers do not render the links properly
		// related to https://github.com/facebook/docusaurus/issues/4801
		return "&#91;&#93;" + renderLrType(t.ListType.Type, resourceHrefMap)
	case t.MapType != nil:
		return "map[" + t.MapType.Key.Type + "]" + renderLrType(t.MapType.Value, resourceHrefMap)
	default:
		return "?"
	}
}

func sanitizeComments(c []string) []string {
	for i := range c {
		c[i] = strings.TrimPrefix(c[i], "// ")
	}

	return c
}
