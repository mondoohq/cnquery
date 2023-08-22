// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package components

import (
	"errors"
	"os"

	"golang.org/x/term"
)

func TerminalWidth(f *os.File) (int, error) {
	w, _, err := term.GetSize(int(f.Fd()))
	if err != nil {
		return 0, errors.New("can't query terminal size")
	}
	return w, nil
}
