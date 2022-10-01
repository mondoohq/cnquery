package reporter

import (
	"errors"

	"go.mondoo.com/cnquery/cli/theme/colors"
)

type Reporter struct {
	// Pager set to true will use a pager for the output. Only relevant for all
	// non-json/yaml/junit/csv reports (for now)
	UsePager    bool
	Pager       string
	Format      Format
	Colors      *colors.Theme
	IsIncognito bool
	IsVerbose   bool
}

func New(typ string) (*Reporter, error) {
	return nil, errors.New("Reporter NOT YET IMPLEMENTED")
}
