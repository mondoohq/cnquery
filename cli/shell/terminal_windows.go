// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package shell

func (s *Shell) backupTerminalSettings() {}

func (s *Shell) restoreTerminalSettings() {}

func (s *Shell) suspend() {}
