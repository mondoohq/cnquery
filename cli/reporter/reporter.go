package reporter

import (
	"errors"
	"io"
	"strings"

	"go.mondoo.com/cnquery/cli/printer"
	"go.mondoo.com/cnquery/cli/theme/colors"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/shared"
)

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
	switch r.Format {
	case Compact:
		rr := &defaultReporter{
			Reporter:  r,
			isCompact: true,
			out:       out,
			data:      data,
		}
		return rr.print()
	case Summary:
		rr := &defaultReporter{
			Reporter:  r,
			isCompact: true,
			isSummary: true,
			out:       out,
			data:      data,
		}
		return rr.print()
	case Full:
		rr := &defaultReporter{
			Reporter:  r,
			isCompact: false,
			out:       out,
			data:      data,
		}
		return rr.print()
	case JSON:
		w := shared.IOWriter{Writer: out}
		return ReportCollectionToJSON(data, &w)
	// case CSV:
	// 	res, err = data.ToCsv()
	default:
		return errors.New("unknown reporter type, don't recognize this Format")
	}
}
