// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shell

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/mql"
	"go.mondoo.com/cnquery/v12/mqlc"
	"go.mondoo.com/cnquery/v12/mqlc/parser"
	"go.mondoo.com/cnquery/v12/utils/stringx"
)

// ErrNotTTY is returned when the shell is run without a terminal
var ErrNotTTY = errors.New("shell requires an interactive terminal (TTY)")

// Message types for async operations
type (
	historyLoadedMsg struct {
		history []string
	}
	queryResultMsg struct {
		code    *llx.CodeBundle
		results map[string]*llx.RawResult
		err     error
	}
	printOutputMsg struct {
		output string
	}
)

// shellModel is the main Bubble Tea model for the interactive shell
type shellModel struct {
	// Runtime and configuration
	runtime  llx.Runtime
	theme    *ShellTheme
	features cnquery.Features
	keyMap   KeyMap

	// Input handling
	input textarea.Model

	// Completion state
	completer   *Completer
	suggestions []Suggestion
	selected    int
	showPopup   bool

	// Query state
	query           string
	isMultiline     bool
	multilineIndent int

	// History
	history      []string
	historyIdx   int
	historyDraft string
	historyPath  string

	// Layout
	width  int
	height int

	// State
	ready     bool
	quitting  bool
	executing bool
	spinner   spinner.Model
}

// newShellModel creates a new shell model
func newShellModel(runtime llx.Runtime, theme *ShellTheme, features cnquery.Features, initialCmd string) *shellModel {
	// Create textarea for input
	ta := textarea.New()
	ta.Placeholder = ""
	ta.Prompt = theme.Prefix
	ta.CharLimit = 0 // No limit
	ta.ShowLineNumbers = false
	ta.SetHeight(1)
	ta.SetWidth(80)
	ta.Focus()

	// Style the textarea
	ta.FocusedStyle.Prompt = theme.Prompt
	ta.FocusedStyle.Text = theme.InputText
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle = ta.FocusedStyle

	// Create completer and set up the schema for the printer
	schema := runtime.Schema()
	theme.PolicyPrinter.SetSchema(schema)
	completer := NewCompleter(schema, features, nil)

	// Create spinner for query execution
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = theme.Spinner

	m := &shellModel{
		runtime:     runtime,
		theme:       theme,
		features:    features,
		keyMap:      DefaultKeyMap(),
		input:       ta,
		completer:   completer,
		suggestions: nil,
		selected:    0,
		showPopup:   false,
		history:     []string{},
		historyIdx:  -1,
		width:       80,
		height:      24,
		spinner:     sp,
	}

	// Set the query prefix callback for completer
	completer.queryPrefix = func() string {
		return m.query
	}

	// Handle initial command
	if initialCmd != "" {
		m.input.SetValue(initialCmd)
	}

	return m
}

// Init implements tea.Model
func (m *shellModel) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.loadHistory(),
		// Print welcome message
		tea.Println(m.theme.Welcome),
	)
}

// loadHistory loads command history from disk
func (m *shellModel) loadHistory() tea.Cmd {
	return func() tea.Msg {
		homeDir, err := homedir.Dir()
		if err != nil {
			return historyLoadedMsg{history: []string{}}
		}

		historyPath := path.Join(homeDir, ".mondoo_history")
		rawHistory, err := os.ReadFile(historyPath)
		if err != nil {
			return historyLoadedMsg{history: []string{}}
		}

		history := strings.Split(string(rawHistory), "\n")
		// Filter empty lines
		filtered := make([]string, 0, len(history))
		for _, h := range history {
			if h != "" {
				filtered = append(filtered, h)
			}
		}

		return historyLoadedMsg{history: filtered}
	}
}

// saveHistory saves command history to disk
func (m *shellModel) saveHistory() {
	if m.historyPath == "" {
		homeDir, _ := homedir.Dir()
		m.historyPath = path.Join(homeDir, ".mondoo_history")
	}

	rawHistory := strings.Join(m.history, "\n")
	if err := os.WriteFile(m.historyPath, []byte(rawHistory), 0o640); err != nil {
		log.Error().Err(err).Msg("failed to save history")
	}
}

// Update implements tea.Model
func (m *shellModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.SetWidth(msg.Width - 4)
		m.ready = true
		return m, nil

	case historyLoadedMsg:
		m.history = msg.history
		m.historyIdx = len(m.history)
		homeDir, _ := homedir.Dir()
		m.historyPath = path.Join(homeDir, ".mondoo_history")
		return m, nil

	case queryResultMsg:
		// Query finished executing
		m.executing = false
		// Print results directly to terminal (outside of Bubble Tea's view)
		if msg.err != nil {
			output := m.theme.ErrorText("failed to compile: " + msg.err.Error())
			if msg.code != nil && msg.code.Suggestions != nil {
				output += "\n" + m.formatSuggestions(msg.code.Suggestions)
			}
			return m, tea.Println(output)
		}
		output := m.formatResults(msg.code, msg.results)
		return m, tea.Println(output)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case printOutputMsg:
		// Print output directly to terminal
		return m, tea.Println(msg.output)

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	// Update textarea
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleKeyMsg processes keyboard input
func (m *shellModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle completion popup if visible
	if m.showPopup && len(m.suggestions) > 0 {
		switch msg.String() {
		case "down", "tab":
			m.selected = (m.selected + 1) % len(m.suggestions)
			return m, nil
		case "up", "shift+tab":
			m.selected--
			if m.selected < 0 {
				m.selected = len(m.suggestions) - 1
			}
			return m, nil
		case "enter":
			// Accept the selected completion
			return m.acceptCompletion()
		case "esc":
			m.showPopup = false
			m.suggestions = nil
			return m, nil
		}
	}

	switch msg.String() {
	case "ctrl+d":
		m.quitting = true
		return m, tea.Quit

	case "ctrl+c":
		// If there's any input or we're in multiline mode, cancel it
		if m.input.Value() != "" || m.isMultiline {
			m.isMultiline = false
			m.query = ""
			m.input.SetValue("")
			m.input.SetHeight(1)
			m.showPopup = false
			m.suggestions = nil
			m.updatePrompt()
			// Print ^C to show the interrupt
			return m, tea.Println("^C")
		}
		// No input - quit the shell
		m.quitting = true
		return m, tea.Sequence(
			tea.Println("^C"),
			tea.Quit,
		)

	case "ctrl+l":
		// Clear screen using ANSI escape codes
		return m, tea.Println("\033[2J\033[H")

	case "ctrl+j":
		// Insert a newline for manual multiline input
		// Note: ctrl+j is the traditional Unix newline (LF) character
		// alt+enter and shift+enter don't work reliably in most terminals
		// Dismiss completion popup
		m.showPopup = false
		m.suggestions = nil
		// Insert newline at cursor position
		m.input.InsertString("\n")
		// Adjust textarea height to fit content
		lines := strings.Count(m.input.Value(), "\n") + 1
		m.input.SetHeight(lines)
		return m, nil

	case "enter":
		return m.handleSubmit()

	case "up":
		// History navigation (only when input is empty or at start)
		if m.input.Value() == "" || m.input.Line() == 0 {
			return m.navigateHistory(-1)
		}

	case "down":
		// History navigation when browsing history or input is empty
		if m.input.Value() == "" || m.historyIdx < len(m.history) {
			return m.navigateHistory(1)
		}
	}

	// Update input and trigger completion
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	// Trigger completion on text change
	m.updateCompletions()

	return m, cmd
}

// handleSubmit processes the enter key
func (m *shellModel) handleSubmit() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.input.Value())

	// Handle empty input
	if input == "" && !m.isMultiline {
		return m, nil
	}

	// Echo the prompt and input so it stays in terminal history
	echoInput := m.theme.Prompt.Render(m.input.Prompt) + input

	// Check for built-in commands (only when not in multiline mode)
	if !m.isMultiline {
		switch input {
		case "exit", "quit":
			m.quitting = true
			return m, tea.Sequence(
				tea.Println(echoInput),
				tea.Quit,
			)
		case "clear":
			m.input.SetValue("")
			// Clear screen using ANSI escape codes
			return m, tea.Println("\033[2J\033[H")
		case "help":
			output := m.listResources("")
			m.input.SetValue("")
			m.addToHistory(input)
			return m, tea.Sequence(
				tea.Println(echoInput),
				tea.Println(output),
			)
		case "nyanya":
			m.input.SetValue("")
			// Run the nyancat animation
			return m, tea.Sequence(
				tea.Println(echoInput),
				func() tea.Msg {
					nyago(m.width, m.height)
					return nil
				},
			)
		}

		// Check for "help <resource>"
		if strings.HasPrefix(input, "help ") {
			resource := strings.TrimPrefix(input, "help ")
			output := m.listResources(resource)
			m.input.SetValue("")
			m.addToHistory(input)
			return m, tea.Sequence(
				tea.Println(echoInput),
				tea.Println(output),
			)
		}
	}

	// Execute as MQL query
	return m.executeQuery(input)
}

// executeQuery compiles and runs an MQL query
func (m *shellModel) executeQuery(input string) (tea.Model, tea.Cmd) {
	// Echo the current line input
	echoInput := m.theme.Prompt.Render(m.input.Prompt) + input

	// Accumulate query for multiline
	m.query += " " + input

	// Try to compile
	code, err := mqlc.Compile(m.query, nil, mqlc.NewConfig(m.runtime.Schema(), m.features))
	if err != nil {
		if e, ok := err.(*parser.ErrIncomplete); ok {
			// Incomplete query - enter multiline mode
			m.isMultiline = true
			m.multilineIndent = e.Indent
			m.input.SetValue("")
			m.updatePrompt()
			// Echo the line for multiline continuation
			return m, tea.Println(echoInput)
		}
	}

	// Query is complete (or has error) - execute it
	cleanCommand := m.query
	if code != nil {
		cleanCommand = code.Source
	}

	m.addToHistory(strings.TrimSpace(cleanCommand))

	// Clear input and reset state
	m.input.SetValue("")
	m.input.SetHeight(1)
	m.isMultiline = false
	m.executing = true
	m.updatePrompt()

	// Execute the query
	queryToRun := m.query
	m.query = ""

	// Echo the input, start spinner, then execute and return results
	return m, tea.Batch(
		tea.Println(echoInput),
		m.spinner.Tick,
		func() tea.Msg {
			code, err := mqlc.Compile(queryToRun, nil, mqlc.NewConfig(m.runtime.Schema(), m.features))
			if err != nil {
				return queryResultMsg{code: code, err: err}
			}

			results, err := mql.ExecuteCode(m.runtime, code, nil, m.features)
			return queryResultMsg{code: code, results: results, err: err}
		},
	)
}

// updatePrompt updates the input prompt based on multiline state
func (m *shellModel) updatePrompt() {
	if m.isMultiline {
		indent := strings.Repeat(" ", m.multilineIndent*2)
		m.input.Prompt = "   .. > " + indent
	} else {
		m.input.Prompt = m.theme.Prefix
	}
}

// addToHistory adds a command to history
func (m *shellModel) addToHistory(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return
	}
	// Don't add duplicates
	if len(m.history) > 0 && m.history[len(m.history)-1] == cmd {
		return
	}
	m.history = append(m.history, cmd)
	m.historyIdx = len(m.history)
}

// navigateHistory moves through command history
func (m *shellModel) navigateHistory(direction int) (tea.Model, tea.Cmd) {
	if len(m.history) == 0 {
		return m, nil
	}

	// Save current input when starting to navigate
	if m.historyIdx == len(m.history) {
		m.historyDraft = m.input.Value()
	}

	newIdx := m.historyIdx + direction
	if newIdx < 0 {
		newIdx = 0
	}
	if newIdx > len(m.history) {
		newIdx = len(m.history)
	}

	m.historyIdx = newIdx

	if m.historyIdx == len(m.history) {
		// Restore draft
		m.input.SetValue(m.historyDraft)
	} else {
		m.input.SetValue(m.history[m.historyIdx])
	}

	return m, nil
}

// updateCompletions fetches new completions based on current input
func (m *shellModel) updateCompletions() {
	input := m.input.Value()
	if input == "" {
		m.showPopup = false
		m.suggestions = nil
		return
	}

	suggestions := m.completer.Complete(input)
	if len(suggestions) > 0 {
		m.suggestions = suggestions
		m.selected = 0
		m.showPopup = true
	} else {
		m.showPopup = false
		m.suggestions = nil
	}
}

// acceptCompletion inserts the selected completion
func (m *shellModel) acceptCompletion() (tea.Model, tea.Cmd) {
	if m.selected >= 0 && m.selected < len(m.suggestions) {
		suggestion := m.suggestions[m.selected]

		// Get current input and find the word to replace
		input := m.input.Value()

		// Find the start of the current word (after last separator)
		lastDot := strings.LastIndex(input, ".")
		lastSpace := strings.LastIndex(input, " ")
		wordStart := lastDot
		if lastSpace > lastDot {
			wordStart = lastSpace
		}

		var newValue string
		if wordStart >= 0 {
			newValue = input[:wordStart+1] + suggestion.Text
		} else {
			newValue = suggestion.Text
		}

		m.input.SetValue(newValue)
	}

	m.showPopup = false
	m.suggestions = nil
	return m, nil
}

// View implements tea.Model
func (m *shellModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	var b strings.Builder

	// Show spinner when executing, otherwise show input
	if m.executing {
		b.WriteString(m.spinner.View())
		b.WriteString(" Executing query...")
	} else {
		// Input area - output is printed directly to terminal via tea.Println
		b.WriteString(m.input.View())

		// Completion popup
		if m.showPopup && len(m.suggestions) > 0 {
			b.WriteString("\n")
			b.WriteString(m.renderCompletionPopup())
		}
	}

	// Help bar at the bottom
	b.WriteString("\n")
	b.WriteString(m.renderHelpBar())

	return b.String()
}

// renderHelpBar renders the help bar with available key bindings
func (m *shellModel) renderHelpBar() string {
	var items []string

	if m.showPopup {
		items = []string{
			m.theme.HelpKey.Render("↑↓") + m.theme.HelpText.Render(" navigate"),
			m.theme.HelpKey.Render("tab") + m.theme.HelpText.Render(" select"),
			m.theme.HelpKey.Render("esc") + m.theme.HelpText.Render(" dismiss"),
		}
	} else if m.executing {
		items = []string{
			m.theme.HelpText.Render("query running..."),
		}
	} else {
		items = []string{
			m.theme.HelpKey.Render("enter") + m.theme.HelpText.Render(" execute"),
			m.theme.HelpKey.Render("ctrl+j") + m.theme.HelpText.Render(" newline"),
			m.theme.HelpKey.Render("↑↓") + m.theme.HelpText.Render(" history"),
			m.theme.HelpKey.Render("ctrl+d") + m.theme.HelpText.Render(" exit"),
		}
	}

	return strings.Join(items, m.theme.HelpText.Render(" • "))
}

// renderCompletionPopup renders the completion suggestions
func (m *shellModel) renderCompletionPopup() string {
	if len(m.suggestions) == 0 {
		return ""
	}

	maxItems := 10
	if len(m.suggestions) < maxItems {
		maxItems = len(m.suggestions)
	}

	// Calculate scroll offset
	startIdx := 0
	if m.selected >= maxItems {
		startIdx = m.selected - maxItems + 1
	}

	// Calculate available width and column sizes
	availableWidth := m.width
	if availableWidth < 40 {
		availableWidth = 80 // fallback
	}

	// Reserve space for: padding (4), separator (3), description (min 20)
	minDescWidth := 20
	maxNameWidth := availableWidth - minDescWidth - 7

	// Cap name column width
	if maxNameWidth > 40 {
		maxNameWidth = 40
	}
	if maxNameWidth < 15 {
		maxNameWidth = 15
	}

	// Find the longest name in visible items (for alignment)
	nameWidth := 0
	for i := startIdx; i < startIdx+maxItems && i < len(m.suggestions); i++ {
		nameLen := len(m.suggestions[i].Text)
		if nameLen > nameWidth {
			nameWidth = nameLen
		}
	}
	// Clamp to maxNameWidth
	if nameWidth > maxNameWidth {
		nameWidth = maxNameWidth
	}
	if nameWidth < 10 {
		nameWidth = 10
	}

	// Calculate description width
	descWidth := availableWidth - nameWidth - 7
	if descWidth < minDescWidth {
		descWidth = minDescWidth
	}
	if descWidth > 50 {
		descWidth = 50
	}

	var rows []string
	for i := startIdx; i < startIdx+maxItems && i < len(m.suggestions); i++ {
		s := m.suggestions[i]

		var suggStyle, descStyle lipgloss.Style
		if i == m.selected {
			suggStyle = m.theme.SuggestionSelected
			descStyle = m.theme.DescriptionSelected
		} else {
			suggStyle = m.theme.SuggestionNormal
			descStyle = m.theme.DescriptionNormal
		}

		// Truncate name if needed
		name := s.Text
		if len(name) > nameWidth {
			name = name[:nameWidth-1] + "…"
		}

		// Truncate description if needed
		desc := s.Description
		if len(desc) > descWidth {
			desc = desc[:descWidth-1] + "…"
		}

		// Format with proper alignment
		nameFormatted := fmt.Sprintf("%-*s", nameWidth, name)
		descFormatted := fmt.Sprintf("%-*s", descWidth, desc)

		row := suggStyle.Render(nameFormatted) + " " + descStyle.Render(descFormatted)
		rows = append(rows, row)
	}

	// Add scroll indicator if needed
	if len(m.suggestions) > maxItems {
		indicator := fmt.Sprintf(" ↑↓ %d/%d", m.selected+1, len(m.suggestions))
		rows = append(rows, m.theme.ScrollIndicator.Render(indicator))
	}

	return strings.Join(rows, "\n")
}

// formatResults formats query results for display
func (m *shellModel) formatResults(code *llx.CodeBundle, results map[string]*llx.RawResult) string {
	result := m.theme.PolicyPrinter.Results(code, results)

	// Apply max lines limit (1024 by default)
	result = stringx.MaxLines(1024, result)

	return result
}

// formatSuggestions formats compiler suggestions for display
func (m *shellModel) formatSuggestions(suggestions []*llx.Documentation) string {
	var b strings.Builder
	b.WriteString(m.theme.SecondaryText("\nsuggestions:\n"))
	for _, s := range suggestions {
		b.WriteString("- " + s.Field + ": " + s.Title + "\n")
	}
	return b.String()
}

// listResources lists available resources
func (m *shellModel) listResources(filter string) string {
	resources := m.runtime.Schema().AllResources()

	var keys []string
	for k := range resources {
		if filter == "" || strings.HasPrefix(k, filter) {
			keys = append(keys, k)
		}
	}

	if len(keys) == 0 {
		return "No resources found"
	}

	// Sort keys
	sortedKeys := make([]string, len(keys))
	copy(sortedKeys, keys)
	for i := 0; i < len(sortedKeys); i++ {
		for j := i + 1; j < len(sortedKeys); j++ {
			if sortedKeys[i] > sortedKeys[j] {
				sortedKeys[i], sortedKeys[j] = sortedKeys[j], sortedKeys[i]
			}
		}
	}

	var b strings.Builder
	for _, k := range sortedKeys {
		resource := resources[k]
		b.WriteString(m.theme.SecondaryText(resource.Name))
		b.WriteString(": ")
		b.WriteString(resource.Title)
		b.WriteString("\n")

		// If filtering to a specific resource, show its fields
		if filter != "" && k == filter {
			for _, field := range resource.Fields {
				if field.IsPrivate {
					continue
				}
				b.WriteString("  ")
				b.WriteString(m.theme.SecondaryText(field.Name))
				b.WriteString(": ")
				b.WriteString(field.Title)
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}
