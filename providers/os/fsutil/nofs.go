// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package fsutil

import (
	"errors"
	"os"
	"time"

	"github.com/spf13/afero"
)

type NoFs struct{}

var errNotImplemented = errors.New("not implemented")

func (NoFs) Create(name string) (afero.File, error) {
	return nil, errNotImplemented
}

func (NoFs) Mkdir(name string, perm os.FileMode) error {
	return errNotImplemented
}

func (NoFs) MkdirAll(path string, perm os.FileMode) error {
	return errNotImplemented
}

func (NoFs) Open(name string) (afero.File, error) {
	return nil, errNotImplemented
}

func (NoFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return nil, errNotImplemented
}

func (NoFs) Remove(name string) error {
	return errNotImplemented
}

func (NoFs) RemoveAll(path string) error {
	return errNotImplemented
}

func (NoFs) Rename(oldname, newname string) error {
	return errNotImplemented
}

func (NoFs) Stat(name string) (os.FileInfo, error) {
	return nil, errNotImplemented
}

func (NoFs) Name() string {
	return "nofs"
}

func (NoFs) Chmod(name string, mode os.FileMode) error {
	return errNotImplemented
}

func (NoFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return errNotImplemented
}

func (NoFs) Chown(name string, uid, gid int) error {
	return errNotImplemented
}
