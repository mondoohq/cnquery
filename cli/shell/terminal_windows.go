//go:build windows

package shell

func (s *Shell) backupTerminalSettings() {}

func (s *Shell) restoreTerminalSettings() {}

func (s *Shell) suspend() {}
