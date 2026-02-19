// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Generate a go source file with feature-flags and helper vars.
//
// You configure feature-flags in YAML. Here is an example:
//
// - desc: Allows MQL to use variable references across blocks. Fully changes the compiled code.
//   end: v7.0
//   id: PiperCode
//   start: v5.x
//   status: default
//   idx: 2           # optional, will be generated
//
// Status can be:
// - builtin: features that are completed and have been built into the code, but shouldn't be used (or set) anymore
// - sunset: features that are completed but have not been built into the code, please don't use them anymore
// - new: features that are now available to be used and aren't active by default
// - default: features that are available and turned on by default (you can still turn them off)
// - unknown: try not to have any unknown features, we don't know what's going on with these but don't use them

package main

import (
	"flag"
	"fmt"
	"go/format"
	"os"
	"strconv"
	"strings"

	"sigs.k8s.io/yaml"
)

type (
	features struct {
		Features []*feature
	}
	feature struct {
		Id     string `json:"id"`
		Idx    int    `json:"idx"`
		Start  string `json:"start"`
		End    string `json:"end,omitempty"`
		Desc   string `json:"desc"`
		Status string `json:"status,omitempty"`
	}
)

func load(src string) features {
	raw, err := os.ReadFile(src)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to read file: "+src)
		os.Exit(1)
	}

	var res features
	err = yaml.Unmarshal(raw, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to parse file: "+src)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "🟢 read features from "+src)
	return res
}

func reIndex(dst string, all *features) {
	max := 0
	for i := range all.Features {
		cur := all.Features[i]
		if cur.Idx != 0 && cur.Idx > max {
			max = cur.Idx
		}
	}

	hasChanged := false
	for i := range all.Features {
		cur := all.Features[i]
		if cur.Idx == 0 {
			cur.Idx = max + 1
			max++
			hasChanged = true
		}

		if cur.Status == "" {
			cur.Status = "new"
		}
	}

	if hasChanged {
		raw, err := yaml.Marshal(all)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to marshal yaml: "+err.Error())
			os.Exit(1)
		}

		err = os.WriteFile(dst, raw, 0o644)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to write yaml to "+dst+": "+err.Error())
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "🟢 updated index of features in "+dst+" => please remember to commit it to git")
	}
}

func main() {
	var featureType string
	flag.StringVar(&featureType, "type", "Feature", "specify a go-type that will be used for each feature")
	var outPath string
	flag.StringVar(&outPath, "out", "features.go", "specify the output path for the generated go code")
	flag.Parse()
	src := flag.Arg(0)

	res := load(src)
	packageName := "mql"

	reIndex(src, &res)

	var out strings.Builder
	out.WriteString(header)
	out.WriteString("\n\npackage " + packageName + "\n\n")
	out.WriteString("const (\n")

	for i := range res.Features {
		cur := res.Features[i]

		out.WriteString("\t// " + strings.ReplaceAll(cur.Desc, "\n", "") + "\n")
		out.WriteString("\t// start:  " + cur.Start + "\n")
		if cur.End != "" {
			out.WriteString("\t// end: " + cur.End + "\n")
		}
		out.WriteString("\t// status: " + cur.Status + "\n")

		out.WriteString("\t" + cur.Id + " " + featureType + " = " + strconv.Itoa(cur.Idx) + "\n\n")
	}
	out.WriteString("\t// Placeholder to indicate how many feature flags exist. This number\n")
	out.WriteString("\t// is changing with every new feature and cannot be used as a featureflag itself.\n")
	out.WriteString("\tMAX_FEATURES byte = " + strconv.Itoa(len(res.Features)+1) + "\n")
	out.WriteString(")\n\n")

	// Feature string to ID mappings
	fmt.Fprintf(&out, "var %ssValue = map[string]%s{\n", featureType, featureType)
	for i := range res.Features {
		cur := res.Features[i]
		out.WriteString("\t\"" + cur.Id + "\": " + cur.Id + ",\n")
	}
	out.WriteString("}\n\n")

	// Default features
	fmt.Fprintf(&out, `// DefaultFeatures are a set of default flags that are active
var DefaultFeatures = %ss{
`, featureType)
	for i := range res.Features {
		cur := res.Features[i]
		if cur.Status == "default" {
			out.WriteString("\tbyte(" + cur.Id + "),\n")
		}
	}
	out.WriteString("}\n\n")

	// Available features
	fmt.Fprintf(&out, `// AvailableFeatures are a set of flags that can be activated
var AvailableFeatures = %ss{
`, featureType)
	for i := range res.Features {
		cur := res.Features[i]
		if cur.Status == "new" {
			out.WriteString("\tbyte(" + cur.Id + "),\n")
		}
	}
	out.WriteString("}\n")

	if outPath != "" {
		fmtGoData, err := format.Source([]byte(out.String()))
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to format go code: "+err.Error())
			os.Exit(1)
		}
		err = os.WriteFile(outPath, fmtGoData, 0o644)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to write go code to "+outPath+": "+err.Error())
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "🟢 wrote go code to "+outPath)
	}
}

const header = "// Code generated by \"featureflags -type=Feature\"; DO NOT EDIT."
