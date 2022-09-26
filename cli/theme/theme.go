package theme

import (
	"fmt"

	"github.com/c-bata/go-prompt"
	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/cli/printer"
	"go.mondoo.com/cnquery/cli/theme/colors"
)

type PromptColors struct {
	PrefixTextColor              prompt.Color
	PreviewSuggestionTextColor   prompt.Color
	PreviewSuggestionBGColor     prompt.Color
	SuggestionTextColor          prompt.Color
	SuggestionBGColor            prompt.Color
	SelectedSuggestionTextColor  prompt.Color
	SelectedSuggestionBGColor    prompt.Color
	DescriptionTextColor         prompt.Color
	DescriptionBGColor           prompt.Color
	SelectedDescriptionTextColor prompt.Color
	SelectedDescriptionBGColor   prompt.Color
	ScrollbarBGColor             prompt.Color
	ScrollbarThumbColor          prompt.Color
}

// Theme to configure how the shell will look and feel
type Theme struct {
	Colors       colors.Theme
	PromptColors PromptColors

	List          func(...string) string
	Landing       string
	Welcome       string
	Prefix        string
	PolicyPrinter printer.Printer
}

func (t Theme) Primary(s ...interface{}) string {
	return termenv.String(fmt.Sprint(s...)).Foreground(t.Colors.Primary).String()
}

func (t Theme) Secondary(s ...interface{}) string {
	return termenv.String(fmt.Sprint(s...)).Foreground(t.Colors.Secondary).String()
}

func (t Theme) Disabled(s ...interface{}) string {
	return termenv.String(fmt.Sprint(s...)).Foreground(t.Colors.Disabled).String()
}

func (t Theme) Error(s ...interface{}) string {
	return termenv.String(fmt.Sprint(s...)).Foreground(DefaultTheme.Colors.Error).String()
}

func (t Theme) Success(s ...interface{}) string {
	return termenv.String(fmt.Sprint(s...)).Foreground(DefaultTheme.Colors.Success).String()
}
