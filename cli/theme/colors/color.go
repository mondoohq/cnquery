package colors

// NOTE: this package is used by various packages and should really have NO external dependency

import (
	"github.com/muesli/termenv"
)

// Color Theme
type Theme struct {
	// messages
	Primary   termenv.Color
	Secondary termenv.Color
	Disabled  termenv.Color
	Error     termenv.Color
	Success   termenv.Color

	// severity
	Critical termenv.Color
	High     termenv.Color
	Medium   termenv.Color
	Low      termenv.Color
	Good     termenv.Color
	Unknown  termenv.Color
}

func ProfileName(profile termenv.Profile) string {
	switch profile {
	case termenv.Ascii:
		return "Ascii"
	case termenv.ANSI:
		return "ANSI"
	case termenv.ANSI256:
		return "ANSI256"
	case termenv.TrueColor:
		return "TrueColor"
	default:
		return "unknown"
	}
}
