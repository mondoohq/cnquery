// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"encoding/json"
	"go/format"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/mql/providers-sdk/v1/mqlr/lrcore"
	"go.mondoo.com/mql/providers-sdk/v1/resources"
)

var goCmd = &cobra.Command{
	Use:   "go",
	Short: "convert LR file to go",
	Long:  `parse an LR file and convert it to go, saving it in the same location with the suffix .go`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dist, err := cmd.Flags().GetString("dist")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get dist flag")
		}

		failOnDups, _ := cmd.Flags().GetBool("fail-on-duplicates")
		failOnBreaking, _ := cmd.Flags().GetBool("fail-on-breaking")
		headerFile, _ := cmd.Flags().GetString("license-header-file")
		runGoCmd(args[0], dist, headerFile, failOnDups, failOnBreaking)
	},
}

func init() {
	rootCmd.AddCommand(goCmd)
	goCmd.Flags().Bool("fail-on-duplicates", false, "fail if duplicate LR field paths are detected")
	goCmd.Flags().Bool("fail-on-breaking", false, "fail if the schema change is breaking and has no migration (ADR 040 part 5)")
	goCmd.Flags().String("dist", "", "folder for output json generation")
	goCmd.Flags().String("license-header-file", "", "optional file path to read license header from")
}

func runGoCmd(lrFile string, dist string, headerFile string, failOnDups bool, failOnBreaking bool) {
	packageName := path.Base(path.Dir(lrFile))
	res, err := lrcore.Resolve(lrFile, func(path string) ([]byte, error) {
		return os.ReadFile(path)
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to resolve")
		return
	}

	dups := res.GetDuplicates()
	if failOnDups && len(dups) > 0 {
		log.Fatal().Int("count", len(dups)).Strs("paths", dups).Msg("duplicate field paths detected, exiting")
	} else if len(dups) > 0 {
		log.Warn().Int("count", len(dups)).Strs("paths", dups).Msg("duplicate field paths detected")
	}

	// add license header
	var headerTpl *template.Template
	if headerFile != "" {
		headerRaw, err := os.ReadFile(headerFile)
		if err != nil {
			log.Fatal().Err(err).Msg("could not read license header file")
		}
		headerTpl, err = template.New("license_header").Parse(string(headerRaw))
		if err != nil {
			log.Fatal().Err(err).Msg("could not parse license header template")
		}
	}

	collector := lrcore.NewCollector(lrFile)
	goCode, err := lrcore.Go(packageName, res, collector, headerTpl)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to compile go code")
	}

	fmtGoData, err := format.Source([]byte(goCode))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to format go code")
	}
	err = os.WriteFile(lrFile+".go", fmtGoData, 0o644)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to write to go file")
	}

	schema, err := lrcore.Schema(res)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to generate schema")
	}

	// Auto-detect .lr.versions file and inject version metadata into the schema
	versionsPath := strings.TrimSuffix(lrFile, ".lr") + ".lr.versions"
	versions, err := lrcore.ReadVersions(versionsPath)
	if err == nil {
		lrcore.InjectVersions(schema, versions)
	} else if os.IsNotExist(err) {
		log.Info().Str("path", versionsPath).Msg("no versions file found, ignoring")
	} else {
		log.Fatal().Err(err).Str("path", versionsPath).Msg("failed to read versions file")
	}

	schemaData, err := json.Marshal(schema)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to generate schema json")
	}

	base := path.Base(lrFile)
	base = strings.TrimSuffix(base, ".lr")

	dst := strings.TrimSuffix(lrFile, ".lr") + ".resources.json"

	// ADR 040 part 5: codegen is the one place that holds both the committed
	// schema and the new one, so it is where a breaking change gets caught in
	// the author's PR rather than at runtime on an older client. A warning for
	// now; the hard-fail lands with the migration lenses that would satisfy it.
	if breaking := reportSchemaDiff(dst, schema); len(breaking) > 0 && failOnBreaking {
		log.Fatal().Int("count", len(breaking)).
			Msg("breaking schema changes detected, exiting")
	}

	err = os.WriteFile(dst, []byte(schemaData), 0o644)
	if err != nil {
		log.Fatal().Err(err).Str("dst", dst).Msg("failed to write schema json")
	}

	if dist != "" {
		if err = os.MkdirAll(dist, 0o755); err != nil {
			log.Fatal().Err(err).Msg("failed to create dist folder")
		}
		infoFile := path.Join(dist, base+".resources.json")
		err = os.WriteFile(infoFile, []byte(schemaData), 0o644)
		if err != nil {
			log.Fatal().Err(err).Str("dst", infoFile).Msg("failed to write schema json")
		}
	}
}

// reportSchemaDiff compares the schema about to be written against the one
// already committed and logs what changed. It returns the breaking changes.
//
// A missing or unreadable previous schema is not a finding: that is a new
// provider, or a first generation, and there is nothing to compare against.
func reportSchemaDiff(previousPath string, nu *resources.Schema) []lrcore.Change {
	raw, err := os.ReadFile(previousPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Debug().Err(err).Str("path", previousPath).
				Msg("cannot read the previous schema, skipping the change report")
		}
		return nil
	}

	var previous resources.Schema
	if err := json.Unmarshal(raw, &previous); err != nil {
		log.Warn().Err(err).Str("path", previousPath).
			Msg("cannot parse the previous schema, skipping the change report")
		return nil
	}

	changes := lrcore.DiffSchemas(&previous, nu)
	if len(changes) == 0 {
		return nil
	}

	breaking := lrcore.Breaking(changes)
	log.Info().
		Int("additive", len(changes)-len(breaking)).
		Int("breaking", len(breaking)).
		Msg("schema changed")

	for _, c := range breaking {
		log.Warn().Str("path", c.Path).Msg("breaking schema change: " + c.Detail)
	}
	return breaking
}
