// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shell

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	prompt "github.com/c-bata/go-prompt"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10"
	"go.mondoo.com/cnquery/v10/cli/theme"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/mql"
	"go.mondoo.com/cnquery/v10/mqlc"
	"go.mondoo.com/cnquery/v10/mqlc/parser"
	"go.mondoo.com/cnquery/v10/providers"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v10/types"
	"go.mondoo.com/cnquery/v10/utils/sortx"
	"go.mondoo.com/cnquery/v10/utils/stringx"
)

type ShellOption func(c *Shell)

func WithOnCloseListener(onCloseHandler func()) ShellOption {
	return func(t *Shell) {
		t.onCloseHandler = onCloseHandler
	}
}

func WithUpstreamConfig(c *upstream.UpstreamConfig) ShellOption {
	return func(t *Shell) {
		if x, ok := t.Runtime.(*providers.Runtime); ok {
			x.UpstreamConfig = c
		}
	}
}

func WithFeatures(features cnquery.Features) ShellOption {
	return func(t *Shell) {
		t.features = features
	}
}

func WithOutput(writer io.Writer) ShellOption {
	return func(t *Shell) {
		t.out = writer
	}
}

func WithTheme(theme *theme.Theme) ShellOption {
	return func(t *Shell) {
		t.Theme = theme
	}
}

// Shell is the interactive explorer
type Shell struct {
	Runtime     llx.Runtime
	Theme       *theme.Theme
	History     []string
	HistoryPath string
	MaxLines    int

	completer       *Completer
	alreadyPrinted  *sync.Map
	out             io.Writer
	features        cnquery.Features
	onCloseHandler  func()
	query           string
	isMultiline     bool
	multilineIndent int
}

// New creates a new Shell
func New(runtime llx.Runtime, opts ...ShellOption) (*Shell, error) {
	res := Shell{
		alreadyPrinted: &sync.Map{},
		out:            os.Stdout,
		features:       cnquery.DefaultFeatures,
		MaxLines:       1024,
		Runtime:        runtime,
	}

	for i := range opts {
		opts[i](&res)
	}

	if res.Theme == nil {
		res.Theme = theme.DefaultTheme
	}

	schema := runtime.Schema()
	res.Theme.PolicyPrinter.SetSchema(schema)

	res.completer = NewCompleter(runtime.Schema(), res.features, func() string {
		return res.query
	})

	return &res, nil
}

func (s *Shell) printWelcome() {
	if s.Theme.Welcome == "" {
		return
	}

	fmt.Fprintln(s.out, s.Theme.Welcome)
}

func (s *Shell) print(msg string) {
	if msg == "" {
		return
	}

	if _, ok := s.alreadyPrinted.Load(msg); !ok {
		s.alreadyPrinted.Store(msg, struct{}{})
		fmt.Fprintln(s.out, msg)
	}
}

// reset the cache that deduplicates messages on the shell
func (s *Shell) resetPrintCache() {
	s.alreadyPrinted = &sync.Map{}
}

// RunInteractive starts a REPL loop
func (s *Shell) RunInteractive(cmd string) {
	s.backupTerminalSettings()
	s.printWelcome()

	s.History = []string{}
	homeDir, _ := homedir.Dir()
	s.HistoryPath = path.Join(homeDir, ".mondoo_history")
	if rawHistory, err := os.ReadFile(s.HistoryPath); err == nil {
		s.History = strings.Split(string(rawHistory), "\n")
	}

	if cmd != "" {
		s.ExecCmd(cmd)
		s.History = append(s.History, cmd)
	}

	completer := s.completer.CompletePrompt
	// NOTE: this is an issue with windows cmd and powershell prompt, since this is not reliable we deactivate the
	// autocompletion, see https://github.com/c-bata/go-prompt/issues/209
	if runtime.GOOS == "windows" {
		completer = func(doc prompt.Document) []prompt.Suggest {
			return nil
		}
	}

	p := prompt.New(
		s.ExecCmd,
		completer,
		prompt.OptionPrefix(s.Theme.Prefix),
		prompt.OptionPrefixTextColor(s.Theme.PromptColors.PrefixTextColor),
		prompt.OptionLivePrefix(s.changeLivePrefix),
		prompt.OptionPreviewSuggestionTextColor(s.Theme.PromptColors.PreviewSuggestionTextColor),
		prompt.OptionPreviewSuggestionBGColor(s.Theme.PromptColors.PreviewSuggestionBGColor),
		prompt.OptionSelectedSuggestionTextColor(s.Theme.PromptColors.SelectedSuggestionTextColor),
		prompt.OptionSelectedSuggestionBGColor(s.Theme.PromptColors.SelectedSuggestionBGColor),
		prompt.OptionSuggestionTextColor(s.Theme.PromptColors.SuggestionTextColor),
		prompt.OptionSuggestionBGColor(s.Theme.PromptColors.SuggestionBGColor),
		prompt.OptionDescriptionTextColor(s.Theme.PromptColors.DescriptionTextColor),
		prompt.OptionDescriptionBGColor(s.Theme.PromptColors.DescriptionBGColor),
		prompt.OptionSelectedDescriptionTextColor(s.Theme.PromptColors.SelectedDescriptionTextColor),
		prompt.OptionSelectedDescriptionBGColor(s.Theme.PromptColors.SelectedDescriptionBGColor),
		prompt.OptionScrollbarBGColor(s.Theme.PromptColors.ScrollbarBGColor),
		prompt.OptionScrollbarThumbColor(s.Theme.PromptColors.ScrollbarThumbColor),
		prompt.OptionAddKeyBind(
			prompt.KeyBind{
				Key: prompt.ControlC,
				Fn: func(buf *prompt.Buffer) {
					s.print("")
				},
			},
			prompt.KeyBind{
				Key: prompt.ControlD,
				Fn: func(buf *prompt.Buffer) {
					s.handleExit()
				},
			},
			prompt.KeyBind{
				Key: prompt.ControlZ,
				Fn: func(buf *prompt.Buffer) {
					s.suspend()
				},
			},
		),
		prompt.OptionHistory(s.History),
		prompt.OptionCompletionWordSeparator(completerSeparator),
	)

	p.Run()

	s.handleExit()
}

var helpResource = regexp.MustCompile(`help\s(.*)`)

func (s *Shell) ExecCmd(cmd string) {
	switch {
	case s.isMultiline:
		s.execQuery(cmd)
	case cmd == "":
		return
	case cmd == "quit":
		fallthrough
	case cmd == "exit":
		s.handleExit()
		return
	case cmd == "clear":
		// clear screen
		s.out.Write([]byte{0x1b, '[', '2', 'J'})
		// move cursor to home
		s.out.Write([]byte{0x1b, '[', 'H'})
		return
	case cmd == "help":
		s.listAvailableResources()
		return
	case cmd == "nyanya":
		size := prompt.NewStandardInputParser().GetWinSize()
		nyago(int(size.Col), int(size.Row))
		return
	case helpResource.MatchString(cmd):
		s.listFilteredResources(cmd)
		return
	default:
		s.execQuery(cmd)
	}
}

func (s *Shell) execQuery(cmd string) {
	s.query += " " + cmd

	// Note: we could optimize the call structure here, since compile
	// will end up being called twice. However, since we are talking about
	// the shell and we only deal with one query at a time, with the
	// compiler being rather fast, the additional time is negligible
	// and may not be worth coding around.
	code, err := mqlc.Compile(s.query, nil, mqlc.NewConfig(s.Runtime.Schema(), s.features))
	if err != nil {
		if e, ok := err.(*parser.ErrIncomplete); ok {
			s.isMultiline = true
			s.multilineIndent = e.Indent
			return
		}
	}

	// at this point we know this is not a multi-line call anymore

	cleanCommand := s.query
	if code != nil {
		cleanCommand = code.Source
	}

	if len(s.History) == 0 || s.History[len(s.History)-1] != cleanCommand {
		s.History = append(s.History, cleanCommand)
	}

	code, res, err := s.RunOnce(s.query)
	// we can safely ignore err != nil, since RunOnce handles most of the printing we need
	if err == nil {
		s.PrintResults(code, res)
	}

	s.isMultiline = false
	s.query = ""
}

func (s *Shell) changeLivePrefix() (string, bool) {
	if s.isMultiline {
		indent := strings.Repeat(" ", s.multilineIndent*2)
		return "   .. > " + indent, true
	}
	return "", false
}

// handleExit is called when the user wants to exit the shell, it restores the terminal
// when the interactive prompt has been used and writes the history to disk. Once that
// is completed it calls Close() to call the optional close handler for the provider
func (s *Shell) handleExit() {
	rawHistory := strings.Join(s.History, "\n")
	err := os.WriteFile(s.HistoryPath, []byte(rawHistory), 0o640)
	if err != nil {
		log.Error().Err(err).Msg("failed to save history")
	}

	s.restoreTerminalSettings()

	// run onClose handler if set
	s.Close()

	os.Exit(0)
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
	s.resetPrintCache()

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

func (s *Shell) PrintResults(code *llx.CodeBundle, results map[string]*llx.RawResult) {
	printedResult := s.Theme.PolicyPrinter.Results(code, results)

	if s.MaxLines > 0 {
		printedResult = stringx.MaxLines(s.MaxLines, printedResult)
	}

	fmt.Fprint(s.out, "\r")
	fmt.Fprintln(s.out, printedResult)
}

func indent(indent int) string {
	indentTxt := ""
	for i := 0; i < indent; i++ {
		indentTxt += " "
	}
	return indentTxt
}

// listAvailableResources lists resource names and their title
func (s *Shell) listAvailableResources() {
	resources := s.Runtime.Schema().AllResources()
	keys := sortx.Keys(resources)
	s.renderResources(resources, keys)
}

// listFilteredResources displays the schema of one or many resources that start with the provided prefix
func (s *Shell) listFilteredResources(cmd string) {
	m := helpResource.FindStringSubmatch(cmd)
	if len(m) == 0 {
		return
	}

	search := m[1]
	resources := s.Runtime.Schema().AllResources()

	// if we find the requested resource, just return it
	if _, ok := resources[search]; ok {
		s.renderResources(resources, []string{search})
		return
	}

	// otherwise we will look for anything that matches
	keys := []string{}
	for k := range resources {
		if strings.HasPrefix(k, search) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	s.renderResources(resources, keys)
}

// renderResources renders a set of resources from a given schema
func (s *Shell) renderResources(resources map[string]*resources.ResourceInfo, keys []string) {
	// list resources and field
	type rowEntry struct {
		key       string
		keylength int
		value     string
	}

	rows := []rowEntry{}
	maxk := 0
	const separator = ":"

	for i := range keys {
		k := keys[i]
		resource := resources[k]

		keyLength := len(resource.Name) + len(separator)
		rows = append(rows, rowEntry{
			s.Theme.PolicyPrinter.Secondary(resource.Name) + separator,
			keyLength,
			resource.Title,
		})
		if maxk < keyLength {
			maxk = keyLength
		}

		fields := sortx.Keys(resource.Fields)
		for i := range fields {
			field := resource.Fields[fields[i]]
			if field.IsPrivate {
				continue
			}

			fieldName := "  " + field.Name
			fieldType := types.Type(field.Type).Label()
			displayType := ""
			fieldComment := field.Title
			if fieldComment == "" && types.Type(field.Type).IsResource() {
				r, ok := resources[fieldType]
				if ok {
					fieldComment = r.Title
				}
			}
			if len(fieldType) > 0 {
				fieldType = " " + fieldType
				displayType = s.Theme.PolicyPrinter.Disabled(fieldType)
			}

			keyLength = len(fieldName) + len(fieldType) + len(separator)
			rows = append(rows, rowEntry{
				s.Theme.PolicyPrinter.Secondary(fieldName) + displayType + separator,
				keyLength,
				fieldComment,
			})
			if maxk < keyLength {
				maxk = keyLength
			}
		}
	}

	for i := range rows {
		entry := rows[i]
		fmt.Fprintln(s.out, entry.key+indent(maxk-entry.keylength+1)+entry.value)
	}
}

// capture the interrupt signal (SIGINT) once and notify a given channel
func captureSIGINTonce(sig chan<- struct{}) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		signal.Stop(c)
		sig <- struct{}{}
	}()
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
