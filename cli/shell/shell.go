// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shell

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"go.mondoo.com/cnquery/v12"
	"go.mondoo.com/cnquery/v12/cli/theme"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/mql"
	"go.mondoo.com/cnquery/v12/mqlc"
	"go.mondoo.com/cnquery/v12/providers"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v12/utils/stringx"
)

// ShellOption configures the legacy Shell (for non-interactive use)
// Deprecated: Use Option with NewShell for interactive shell
type ShellOption func(c *Shell)

// LegacyWithOnCloseListener sets a close handler for the legacy Shell
// Deprecated: Use WithOnClose with NewShell for interactive shell
func LegacyWithOnCloseListener(onCloseHandler func()) ShellOption {
	return func(t *Shell) {
		t.onCloseHandler = onCloseHandler
	}
}

// LegacyWithUpstreamConfig sets the upstream config for the legacy Shell
// Deprecated: Use WithUpstreamConfig with NewShell for interactive shell
func LegacyWithUpstreamConfig(c *upstream.UpstreamConfig) ShellOption {
	return func(t *Shell) {
		if x, ok := t.Runtime.(*providers.Runtime); ok {
			x.UpstreamConfig = c
		}
	}
}

// LegacyWithFeatures sets features for the legacy Shell
// Deprecated: Use WithFeatures with NewShell for interactive shell
func LegacyWithFeatures(features cnquery.Features) ShellOption {
	return func(t *Shell) {
		t.features = features
	}
}

// LegacyWithOutput sets the output writer for the legacy Shell
func LegacyWithOutput(writer io.Writer) ShellOption {
	return func(t *Shell) {
		t.out = writer
	}
}

// LegacyWithTheme sets the theme for the legacy Shell
// Deprecated: Use WithTheme with NewShell for interactive shell
func LegacyWithTheme(theme *theme.Theme) ShellOption {
	return func(t *Shell) {
		t.Theme = theme
	}
}

// Shell provides non-interactive query execution
// For interactive use, use NewShell() which returns a ShellProgram
type Shell struct {
	Runtime  llx.Runtime
	Theme    *theme.Theme
	MaxLines int

	out            io.Writer
	features       cnquery.Features
	onCloseHandler func()
}

// New creates a new Shell for non-interactive query execution
// For interactive shell, use NewShell() instead
func New(runtime llx.Runtime, opts ...ShellOption) (*Shell, error) {
	res := &Shell{
		out:      os.Stdout,
		features: cnquery.DefaultFeatures,
		MaxLines: 1024,
		Runtime:  runtime,
	}

	for _, opt := range opts {
		opt(res)
	}

	if res.Theme == nil {
		res.Theme = theme.DefaultTheme
	}

	schema := runtime.Schema()
	res.Theme.PolicyPrinter.SetSchema(schema)

	return res, nil
}

// Close is called when the shell is closed and calls the onCloseHandler
func (s *Shell) Close() {
	s.Runtime.Close()
	// run onClose handler if set
	if s.onCloseHandler != nil {
		s.onCloseHandler()
	}
}

// RunOnce executes the query and returns results
func (s *Shell) RunOnce(cmd string) (*llx.CodeBundle, map[string]*llx.RawResult, error) {
	code, err := mqlc.Compile(cmd, nil, mqlc.NewConfig(s.Runtime.Schema(), s.features))
	if err != nil {
		fmt.Fprintln(s.out, s.Theme.Error("failed to compile: "+err.Error()))

		if code != nil && code.Suggestions != nil {
			fmt.Fprintln(s.out, formatSuggestions(code.Suggestions, s.Theme))
		}
		return nil, nil, err
	}

	res, err := s.RunOnceBundle(code)
	return code, res, err
}

// RunOnceBundle executes the given code bundle and returns results
func (s *Shell) RunOnceBundle(code *llx.CodeBundle) (map[string]*llx.RawResult, error) {
	return mql.ExecuteCode(s.Runtime, code, nil, s.features)
}

// PrintResults prints the results of a query execution
func (s *Shell) PrintResults(code *llx.CodeBundle, results map[string]*llx.RawResult) {
	printedResult := s.Theme.PolicyPrinter.Results(code, results)

	if s.MaxLines > 0 {
		printedResult = stringx.MaxLines(s.MaxLines, printedResult)
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

// captureSIGINTonce captures the interrupt signal (SIGINT) once and notifies a given channel
// Used by the nyago easter egg
func captureSIGINTonce(sig chan<- struct{}) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		signal.Stop(c)
		sig <- struct{}{}
	}()
}
