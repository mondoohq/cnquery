// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package npm

import (
	"io"
)

type Parser interface {
	Parse(r io.Reader) (*Package, []*Package, error)
}

type Package struct {
	Name        string
	File        string
	License     string
	Description string
	Version     string
	Purl        string
	Cpes        []string
}
