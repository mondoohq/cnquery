package logger

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func NewConsoleWriter(out io.Writer, nocolor bool, compact bool) zerolog.Logger {
	w := zerolog.ConsoleWriter{Out: out, NoColor: nocolor}

	if compact {
		w.FormatLevel = consoleFormatLevel(w.NoColor)
		w.FormatTimestamp = func(i interface{}) string { return "" }
	}

	return log.Output(w)
}

func consoleFormatLevel(noColor bool) zerolog.Formatter {
	return func(i interface{}) string {
		var l string
		if ll, ok := i.(string); ok {
			switch ll {
			case "trace":
				l = colorize("TRC", color.FgMagenta, noColor)
			case "debug":
				l = colorize("DBG", color.FgHiYellow, noColor)
			case "info":
				l = colorize("→", color.FgHiCyan, noColor)
			case "warn":
				l = colorize("𐄂", color.FgHiYellow, noColor)
			case "error":
				l = colorize(colorize("𐄂", color.FgRed, noColor), color.Bold, noColor)
			case "fatal":
				l = colorize(colorize("FTL", color.FgRed, noColor), color.Bold, noColor)
			case "panic":
				l = colorize(colorize("PNC", color.FgRed, noColor), color.Bold, noColor)
			default:
				l = colorize("???", color.Bold, noColor)
			}
		} else {
			if i == nil {
				l = colorize("???", color.Bold, noColor)
			} else {
				l = strings.ToUpper(fmt.Sprintf("%s", i))[0:3]
			}
		}
		return l
	}
}

// colorize returns the string s wrapped in ANSI code c, unless disabled is true.
func colorize(s interface{}, c color.Attribute, disabled bool) string {
	if disabled {
		return fmt.Sprintf("%s", s)
	}
	return color.New(c).Sprintf("%v", s)
}
