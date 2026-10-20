// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cat

import (
	"bytes"
	"errors"
	"io"
	"os"
)

func NewFile(name string, buf *bytes.Buffer) *File {
	return &File{path: name, buf: buf}
}

type File struct {
	buf       *bytes.Buffer
	path      string
	info      os.FileInfo
	entries   []os.FileInfo
	dirOffset int
}

func (f *File) Close() error {
	return nil
}

func (f *File) Name() string {
	return f.path
}

func (f *File) Stat() (os.FileInfo, error) {
	if f.info != nil {
		return f.info, nil
	}
	return nil, errors.New("not implemented")
}

func (f *File) Sync() error {
	return nil
}

func (f *File) Truncate(size int64) error {
	return nil
}

func (f *File) Read(b []byte) (n int, err error) {
	if f.info != nil && f.info.IsDir() {
		return 0, &os.PathError{Op: "read", Path: f.path, Err: os.ErrInvalid}
	}
	return f.buf.Read(b)
}

func (f *File) ReadAt(b []byte, off int64) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (f *File) Readdir(count int) (res []os.FileInfo, err error) {
	if f.info == nil || !f.info.IsDir() {
		return nil, &os.PathError{Op: "readdir", Path: f.path, Err: os.ErrInvalid}
	}
	if count > 0 && f.dirOffset == len(f.entries) {
		return nil, io.EOF
	}
	end := len(f.entries)
	if count > 0 && count < end-f.dirOffset {
		end = f.dirOffset + count
	}
	res = f.entries[f.dirOffset:end]
	f.dirOffset = end
	return res, nil
}

func (f *File) Readdirnames(n int) ([]string, error) {
	entries, err := f.Readdir(n)
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names, err
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	return 0, errors.New("not implemented")
}

func (f *File) Write(b []byte) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (f *File) WriteAt(b []byte, off int64) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (f *File) WriteString(s string) (ret int, err error) {
	return 0, errors.New("not implemented")
}
