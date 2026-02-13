// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shell

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"go.mondoo.com/mql/v13"
	"go.mondoo.com/mql/v13/cli/theme"
	"go.mondoo.com/mql/v13/exec"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/mqlc"
	"go.mondoo.com/mql/v13/providers"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
	"go.mondoo.com/mql/v13/utils/stringx"
)

// Option configures a ShellProgram
type Option func(*ShellProgram)

// WithTheme sets the shell theme
func WithTheme(theme *ShellTheme) Option {
	return func(s *ShellProgram) {
		s.theme = theme
	}
}

// WithFeatures sets mql features
func WithFeatures(features mql.Features) Option {
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

// WithOutput sets the output writer for non-interactive query execution
func WithOutput(w io.Writer) Option {
	return func(s *ShellProgram) {
		s.out = w
	}
}

// WithMaxLines sets the maximum number of lines to display in output
func WithMaxLines(n int) Option {
	return func(s *ShellProgram) {
		s.maxLines = n
	}
}

// ShellProgram is the main entry point for the shell
// It supports both interactive mode (Run) and non-interactive query execution (RunOnce)
type ShellProgram struct {
	runtime        llx.Runtime
	theme          *ShellTheme
	features       mql.Features
	upstreamConfig *upstream.UpstreamConfig
	onCloseHandler func()
	out            io.Writer
	maxLines       int
	printTheme     *theme.Theme
}

// NewShell creates a new shell program
// It can be used for interactive mode (Run) or non-interactive query execution (RunOnce)
func NewShell(runtime llx.Runtime, opts ...Option) *ShellProgram {
	s := &ShellProgram{
		runtime:    runtime,
		theme:      DefaultShellTheme,
		features:   mql.DefaultFeatures,
		out:        os.Stdout,
		maxLines:   1024,
		printTheme: theme.DefaultTheme,
	}

	for _, opt := range opts {
		opt(s)
	}

	// Set upstream config on runtime if provided
	if s.upstreamConfig != nil {
		if x, ok := s.runtime.(*providers.Runtime); ok {
			x.UpstreamConfig = s.upstreamConfig
		}
	}

	// Initialize the policy printer with the schema
	schema := runtime.Schema()
	s.printTheme.PolicyPrinter.SetSchema(schema)

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

	// Get connected provider IDs to filter autocomplete suggestions
	var connectedProviderIDs []string
	if r, ok := s.runtime.(*providers.Runtime); ok {
		connectedProviderIDs = r.ConnectedProviderIDs()
	}

	// Create the model
	model := newShellModel(s.runtime, s.theme, s.features, initialCmd, connectedProviderIDs)

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

// RunOnce executes a query and returns the results (non-interactive)
func (s *ShellProgram) RunOnce(cmd string) (*llx.CodeBundle, map[string]*llx.RawResult, error) {
	code, err := mqlc.Compile(cmd, nil, mqlc.NewConfig(s.runtime.Schema(), s.features))
	if err != nil {
		fmt.Fprintln(s.out, s.printTheme.Error("failed to compile: "+err.Error()))

		if code != nil && code.Suggestions != nil {
			fmt.Fprintln(s.out, formatSuggestions(code.Suggestions, s.printTheme))
		}
		return nil, nil, err
	}

	res, err := s.RunOnceBundle(code)
	return code, res, err
}

// RunOnceBundle executes a pre-compiled code bundle and returns results (non-interactive)
func (s *ShellProgram) RunOnceBundle(code *llx.CodeBundle) (map[string]*llx.RawResult, error) {
	return exec.ExecuteCode(s.runtime, code, nil, s.features)
}

// PrintResults prints the results of a query execution to the output writer
func (s *ShellProgram) PrintResults(code *llx.CodeBundle, results map[string]*llx.RawResult) {
	printedResult := s.printTheme.PolicyPrinter.Results(code, results)

	if s.maxLines > 0 {
		printedResult = stringx.MaxLines(s.maxLines, printedResult)
	}

	fmt.Fprint(s.out, "\r")
	fmt.Fprintln(s.out, printedResult)
}

func formatSuggestions(suggestions []*llx.Documentation, theme *theme.Theme) string {
	var res strings.Builder
	res.WriteString(theme.Secondary("\nsuggestions: \n"))
	for i := range suggestions {
		s := suggestions[i]
		res.WriteString(theme.List(s.Field+": "+s.Title) + "\n")
	}
	return res.String()
}
