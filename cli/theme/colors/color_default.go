//go:build !windows
// +build !windows

package colors

import (
	"github.com/muesli/termenv"
)

var Profile termenv.Profile = termenv.EnvColorProfile()

var DefaultColorTheme = Theme{
	// messages
	Primary:   Profile.Color("75"),
	Secondary: Profile.Color("140"),
	Disabled:  Profile.Color("248"),
	Error:     Profile.Color("210"),
	Success:   Profile.Color("78"),

	// severity
	Critical: Profile.Color("204"),
	High:     Profile.Color("212"),
	Medium:   Profile.Color("75"),
	Low:      Profile.Color("117"),
	Good:     Profile.Color("78"),
	Unknown:  Profile.Color("231"),
}
