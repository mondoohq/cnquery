package reporter

import (
	"errors"
	"io"
	"sort"
	"strings"

	"go.mondoo.com/cnquery/logger"

	"go.mondoo.com/cnquery/cli/printer"
	"go.mondoo.com/cnquery/cli/theme/colors"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/shared"
)

type Format byte

const (
	Compact Format = iota + 1
	Summary
	Full
	YAML
	JSON
	JUnit
	CSV
)

// Formats that are supported by the reporter
var Formats = map[string]Format{
	"compact": Compact,
	"summary": Summary,
	"full":    Full,
	"":        Compact,
	"yaml":    YAML,
	"yml":     YAML,
	"json":    JSON,
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
	// Pager set to true will use a pager for the output. Only relevant for all
	// non-json/yaml/junit/csv reports (for now)
	UsePager    bool
	Pager       string
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
	case JSON:
		w := shared.IOWriter{Writer: out}
		return ReportCollectionToJSON(data, &w)
	case CSV:
		w := shared.IOWriter{Writer: out}
		return ReportCollectionToCSV(data, &w)
	default:
		return errors.New("unknown reporter type, don't recognize this Format")
	}
}
