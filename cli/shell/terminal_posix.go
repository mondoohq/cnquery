// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package shell

import (
	"os/signal"
	"syscall"

	"github.com/pkg/term/termios"
	"go.mondoo.com/cnquery/v11/utils/piped"
	"golang.org/x/sys/unix"
)

var terminalIos *unix.Termios

func (s *Shell) backupTerminalSettings() {
	// we only backup if we have no input pipe
	if piped.IsPipe() {
		return
	}

	var err error
	terminalIos, err = termios.Tcgetattr(uintptr(syscall.Stdin))
	if err != nil {
		panic(err)
	}
}

func (s *Shell) restoreTerminalSettings() {
	signal.Reset(syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGWINCH)
	syscall.SetNonblock(syscall.Stdin, false)
	termios.Tcsetattr(uintptr(syscall.Stdin), termios.TCSANOW, terminalIos)
}

func (s *Shell) suspend() {
	syscall.Kill(syscall.Getppid(), syscall.SIGTSTP)
}
