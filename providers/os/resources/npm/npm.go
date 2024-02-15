// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package npm

import (
	"io"

	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream/mvd"
)

type Parser interface {
	Parse(r io.Reader) ([]*mvd.Package, error)
}
