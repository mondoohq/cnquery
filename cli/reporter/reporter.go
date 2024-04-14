// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:generate protoc --proto_path=../:. --go_out=. --go_opt=paths=source_relative cnquery_report.proto

package reporter

import (
	"bytes"
	"errors"
	"io"
	"sort"
	"strings"

	"go.mondoo.com/cnquery/v10/logger"
	"sigs.k8s.io/yaml"

	"go.mondoo.com/cnquery/v10/cli/printer"
	"go.mondoo.com/cnquery/v10/cli/theme/colors"
	"go.mondoo.com/cnquery/v10/explorer"
	"go.mondoo.com/cnquery/v10/shared"
)

type Format byte

const (
	Compact Format = iota + 1
	Summary
	Full
	YAML
	JSONv1
	JUnit
	CSV
	JSONv2
)

// Formats that are supported by the reporter
var Formats = map[string]Format{
	"compact": Compact,
	"summary": Summary,
	"full":    Full,
	"":        Compact,
	"yaml":    YAML,
	"yml":     YAML,
	"json-v1": JSONv1,
	"json-v2": JSONv2,
	"json":    JSONv1,
	"csv":     CSV,
}

func AllFormats() string {
	var res []string
	for k := range Formats {
		if k != "" && // default if nothing is provided, ignore
			k != "yml" { // don't show both yaml and yml
			res = append(res, k)
		}
	}

	// ensure the order is always the same
	sort.Strings(res)
	return strings.Join(res, ", ")
}

type Reporter struct {
	Format      Format
	Printer     *printer.Printer
	Colors      *colors.Theme
	IsIncognito bool
	IsVerbose   bool
}

func New(typ string) (*Reporter, error) {
	format, ok := Formats[strings.ToLower(typ)]
	if !ok {
		return nil, errors.New("unknown output format '" + typ + "'. Available: " + AllFormats())
	}

	return &Reporter{
		Format:  format,
		Printer: &printer.DefaultPrinter,
		Colors:  &colors.DefaultColorTheme,
	}, nil
}

func (r *Reporter) Print(data *explorer.ReportCollection, out io.Writer) error {
	logger.DebugDumpYAML("report_collection", data)
	switch r.Format {
	case Compact:
		rr := &cliReporter{
			Reporter:  r,
			isCompact: true,
			out:       out,
			data:      data,
		}
		return rr.print()
	case Summary:
		rr := &cliReporter{
			Reporter:  r,
			isCompact: true,
			isSummary: true,
			out:       out,
			data:      data,
		}
		return rr.print()
	case Full:
		rr := &cliReporter{
			Reporter:  r,
			isCompact: false,
			out:       out,
			data:      data,
		}
		return rr.print()
	case JSONv1:
		w := shared.IOWriter{Writer: out}
		return ConvertToJSON(data, &w)
	case JSONv2:
		r, err := ConvertToProto(data)
		if err != nil {
			return err
		}

		data, err := r.ToJSON()
		if err != nil {
			return err
		}
		_, err = out.Write(data)
		return err
	case CSV:
		w := shared.IOWriter{Writer: out}
		return ConvertToCSV(data, &w)
	case YAML:
		raw := bytes.Buffer{}
		writer := shared.IOWriter{Writer: &raw}
		err := ConvertToJSON(data, &writer)
		if err != nil {
			return err
		}

		data, err := yaml.JSONToYAML(raw.Bytes())
		if err != nil {
			return err
		}
		_, err = out.Write(data)
		return err
	default:
		return errors.New("unknown reporter type, don't recognize this Format")
	}
}
