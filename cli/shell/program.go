// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shell

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"go.mondoo.com/cnquery/v12"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/upstream"
)

// Option configures a ShellProgram
type Option func(*ShellProgram)

// WithTheme sets the shell theme
func WithTheme(theme *ShellTheme) Option {
	return func(s *ShellProgram) {
		s.theme = theme
	}
}

// WithFeatures sets the cnquery features
func WithFeatures(features cnquery.Features) Option {
	return func(s *ShellProgram) {
		s.features = features
	}
}

// WithUpstreamConfig sets the upstream configuration
func WithUpstreamConfig(c *upstream.UpstreamConfig) Option {
	return func(s *ShellProgram) {
		s.upstreamConfig = c
	}
}

// WithOnClose sets a callback to run when the shell closes
func WithOnClose(handler func()) Option {
	return func(s *ShellProgram) {
		s.onCloseHandler = handler
	}
}

// ShellProgram is the main entry point for the interactive shell
type ShellProgram struct {
	runtime        llx.Runtime
	theme          *ShellTheme
	features       cnquery.Features
	upstreamConfig *upstream.UpstreamConfig
	onCloseHandler func()
}

// NewShell creates a new interactive shell program
func NewShell(runtime llx.Runtime, opts ...Option) *ShellProgram {
	s := &ShellProgram{
		runtime:  runtime,
		theme:    DefaultShellTheme,
		features: cnquery.DefaultFeatures,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Run starts the interactive shell
func (s *ShellProgram) Run() error {
	return s.RunWithCommand("")
}

// RunWithCommand starts the interactive shell and optionally executes an initial command
func (s *ShellProgram) RunWithCommand(initialCmd string) error {
	// Check if we're running in a terminal
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		return ErrNotTTY
	}

	// Create the model
	model := newShellModel(s.runtime, s.theme, s.features, initialCmd)

	// Create and run the Bubble Tea program
	// Note: We don't use WithAltScreen() so output stays in terminal scrollback
	// Note: We don't use WithMouseCellMotion() so terminal handles text selection natively
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// Handle cleanup
	if m, ok := finalModel.(*shellModel); ok {
		m.saveHistory()
	}

	// Close runtime
	s.runtime.Close()

	// Run close handler if set
	if s.onCloseHandler != nil {
		s.onCloseHandler()
	}

	return nil
}

// Close cleans up the shell resources
func (s *ShellProgram) Close() {
	s.runtime.Close()
	if s.onCloseHandler != nil {
		s.onCloseHandler()
	}
}
