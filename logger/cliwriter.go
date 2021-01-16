package logger

import (
	"fmt"
	"github.com/muesli/termenv"
	"go.mondoo.io/mondoo/cli/theme/colors"
	"io"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func NewConsoleWriter(out io.Writer, compact bool) zerolog.Logger {
	w := zerolog.ConsoleWriter{Out: out}

	if compact {
		w.FormatLevel = consoleFormatLevel()
		w.FormatTimestamp = func(i interface{}) string { return "" }
	}

	return log.Output(w)
}

func consoleFormatLevel() zerolog.Formatter {

	return func(i interface{}) string {
		var l string
		var color termenv.Color

		if ll, ok := i.(string); ok {
			switch ll {
			case "trace":
				l = "TRC"
				color = colors.DefaultColorTheme.Secondary
			case "debug":
				l = "DBG"
				color = colors.DefaultColorTheme.Primary
			case "info":
				l = "→"
				color = colors.DefaultColorTheme.Good
			case "warn":
				l = "!"
				color = colors.DefaultColorTheme.Medium
			case "error":
				l = "x"
				color = colors.DefaultColorTheme.Error
			case "fatal":
				l = "FTL"
				color = colors.DefaultColorTheme.Error
			case "panic":
				l = "PNC"
				color = colors.DefaultColorTheme.Error
			default:
				l = "???"
			}
		} else {
			if i == nil {
				l = "???"
			} else {
				l = strings.ToUpper(fmt.Sprintf("%s", i))[0:3]
			}
		}

		return termenv.String(l).Foreground(color).String()
	}
}
