// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"regexp"

	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/fsutil"
)

var _ shared.FileSearch = (*FS)(nil)

func NewFs() *FS {
	return &FS{}
}

type FS struct {
	afero.OsFs
}

// Find searches for files and returns the file info, regex can be nil
func (fs *FS) Find(from string, r *regexp.Regexp, typ string, perm *uint32, depth *int) ([]string, error) {
	iofs := afero.NewIOFS(fs)
	return fsutil.FindFiles(iofs, from, r, typ, perm, depth)
}
