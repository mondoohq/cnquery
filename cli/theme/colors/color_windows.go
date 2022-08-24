package colors

import (
	"github.com/muesli/termenv"
)

// NOTE: some windows terminals support better colors, but where running into issues where
// different locales behaved differently. If we are going to add a better color scheme, we
// really need to test this very extensively on US, non-US, and putty environments
var Profile termenv.Profile = termenv.Ascii

var DefaultColorTheme = Theme{
	// messages
	Primary:   Profile.Color("#0087ff"),
	Secondary: Profile.Color("#0087d7"),
	Disabled:  Profile.Color("#c0c0c0"),
	Error:     Profile.Color("#800000"),
	Success:   Profile.Color("#008000"),

	// severity
	Critical: Profile.Color("#800000"),
	High:     Profile.Color("#800080"),
	Medium:   Profile.Color("#008080"),
	Low:      Profile.Color("#808000"),
	Good:     Profile.Color("#008000"),
	Unknown:  Profile.Color("#c0c0c0"),
}
