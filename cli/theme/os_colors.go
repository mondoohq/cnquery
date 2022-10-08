//go:build !windows

package theme

import (
	"strings"

	prompt "github.com/c-bata/go-prompt"
	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/cli/printer"
	"go.mondoo.com/cnquery/cli/theme/colors"
)

// OperatingSystemTheme for unix shell
var OperatingSystemTheme = &Theme{
	Colors: colors.DefaultColorTheme,
	PromptColors: PromptColors{
		PrefixTextColor:              prompt.Purple,
		PreviewSuggestionTextColor:   prompt.Blue,
		PreviewSuggestionBGColor:     prompt.DefaultColor,
		SuggestionTextColor:          prompt.DefaultColor,
		SuggestionBGColor:            prompt.DarkGray,
		SelectedSuggestionTextColor:  prompt.White,
		SelectedSuggestionBGColor:    prompt.Purple,
		DescriptionTextColor:         prompt.DefaultColor,
		DescriptionBGColor:           prompt.Purple,
		SelectedDescriptionTextColor: prompt.DefaultColor,
		SelectedDescriptionBGColor:   prompt.Fuchsia,
		ScrollbarBGColor:             prompt.Fuchsia,
		ScrollbarThumbColor:          prompt.DefaultColor,
	},
	List: func(items ...string) string {
		var w strings.Builder
		for i := range items {
			w.WriteString("- " + items[i] + "\n")
		}
		res := w.String()
		return res[0 : len(res)-1]
	},
	Landing:       termenv.String(logo).Foreground(colors.DefaultColorTheme.Primary).String(),
	Welcome:       logo + " interactive shell\n",
	Prefix:        "cnquery> ",
	PolicyPrinter: printer.DefaultPrinter,
}
