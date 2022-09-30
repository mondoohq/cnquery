package reporter

import (
	"errors"
	"io"
	"strings"

	"go.mondoo.com/cnquery/explorer"
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
	return strings.Join(res, ", ")
}

func (r *Reporter) Print(data *explorer.ReportCollection, out io.Writer) error {
	return errors.New("Printing reports is NOT YET IMPLEMENTED")
}
