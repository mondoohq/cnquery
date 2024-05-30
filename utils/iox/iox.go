// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package iox

import "io"

type OutputHelper interface {
	WriteString(string) error
	Write([]byte) (int, error)
}

type IOWriter struct {
	io.Writer
}

func (i *IOWriter) Write(x []byte) (int, error) {
	return i.Writer.Write(x)
}

func (i *IOWriter) WriteString(x string) error {
	_, err := i.Writer.Write([]byte(x))
	return err
}
