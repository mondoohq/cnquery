// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shell

import (
	"github.com/charmbracelet/lipgloss"
	"go.mondoo.com/cnquery/v12/cli/printer"
)

// ShellTheme defines the visual appearance of the shell
type ShellTheme struct {
	// Input styling
	Prompt          lipgloss.Style
	MultilinePrompt lipgloss.Style
	InputText       lipgloss.Style

	// Completion popup
	PopupBorder           lipgloss.Style
	SuggestionNormal      lipgloss.Style
	SuggestionSelected    lipgloss.Style
	DescriptionNormal     lipgloss.Style
	DescriptionSelected   lipgloss.Style
	ScrollIndicator       lipgloss.Style

	// Output
	OutputArea lipgloss.Style
	Error      lipgloss.Style
	Success    lipgloss.Style
	Secondary  lipgloss.Style
	Disabled   lipgloss.Style

	// Status
	Spinner  lipgloss.Style
	HelpBar  lipgloss.Style
	HelpKey  lipgloss.Style
	HelpText lipgloss.Style

	// Text content
	Welcome string
	Prefix  string

	// Printer for results
	PolicyPrinter printer.Printer
}

// Color constants matching the original theme
var (
	colorPurple   = lipgloss.Color("133") // Purple for prefix and selected items
	colorFuchsia  = lipgloss.Color("201") // Fuchsia for accents
	colorBlue     = lipgloss.Color("69")  // Blue for preview
	colorGray     = lipgloss.Color("240") // Dark gray for unselected suggestions
	colorWhite    = lipgloss.Color("15")  // White for selected text
	colorRed      = lipgloss.Color("196") // Red for errors
	colorGreen    = lipgloss.Color("82")  // Green for success
	colorDisabled = lipgloss.Color("245") // Gray for disabled text
)

// DefaultShellTheme is the default theme for the shell
var DefaultShellTheme = &ShellTheme{
	// Input
	Prompt: lipgloss.NewStyle().
		Foreground(colorPurple).
		Bold(true),
	MultilinePrompt: lipgloss.NewStyle().
		Foreground(colorPurple),
	InputText: lipgloss.NewStyle(),

	// Completion popup
	PopupBorder: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPurple),
	SuggestionNormal: lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(colorGray).
		Padding(0, 1),
	SuggestionSelected: lipgloss.NewStyle().
		Foreground(colorWhite).
		Background(colorPurple).
		Bold(true).
		Padding(0, 1),
	DescriptionNormal: lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(colorPurple).
		Padding(0, 1),
	DescriptionSelected: lipgloss.NewStyle().
		Foreground(colorWhite).
		Background(colorFuchsia).
		Padding(0, 1),
	ScrollIndicator: lipgloss.NewStyle().
		Foreground(colorFuchsia),

	// Output
	OutputArea: lipgloss.NewStyle().
		MarginTop(1),
	Error: lipgloss.NewStyle().
		Foreground(colorRed),
	Success: lipgloss.NewStyle().
		Foreground(colorGreen),
	Secondary: lipgloss.NewStyle().
		Foreground(colorPurple),
	Disabled: lipgloss.NewStyle().
		Foreground(colorDisabled),

	// Status
	Spinner: lipgloss.NewStyle().
		Foreground(colorFuchsia),
	HelpBar: lipgloss.NewStyle().
		Foreground(colorDisabled),
	HelpKey: lipgloss.NewStyle().
		Foreground(colorPurple).
		Bold(true),
	HelpText: lipgloss.NewStyle().
		Foreground(colorDisabled),

	// Text
	Welcome: logo + " interactive shell\n",
	Prefix:  "cnquery> ",

	// Printer
	PolicyPrinter: printer.DefaultPrinter,
}

// logo for the shell
const logo = `
  ___ _ __   __ _ _   _  ___ _ __ _   _
 / __| '_ \ / _` + "`" + ` | | | |/ _ \ '__| | | |
| (__| | | | (_| | |_| |  __/ |  | |_| |
 \___|_| |_|\__, |\__,_|\___|_|   \__, |
  mondoo™      |_|                |___/
`

// Error formats a string as an error message
func (t *ShellTheme) ErrorText(s string) string {
	return t.Error.Render(s)
}

// SuccessText formats a string as a success message
func (t *ShellTheme) SuccessText(s string) string {
	return t.Success.Render(s)
}

// SecondaryText formats a string as secondary text
func (t *ShellTheme) SecondaryText(s string) string {
	return t.Secondary.Render(s)
}

// DisabledText formats a string as disabled/dimmed text
func (t *ShellTheme) DisabledText(s string) string {
	return t.Disabled.Render(s)
}
