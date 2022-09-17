package printer

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/cli/theme/colors"
)

// Printer turns code into human-readable strings
type Printer struct {
	Primary   func(...interface{}) string
	Secondary func(...interface{}) string
	Yellow    func(...interface{}) string
	Error     func(...interface{}) string
	Warn      func(...interface{}) string
	Disabled  func(...interface{}) string
	Failed    func(...interface{}) string
	Success   func(...interface{}) string
}

// DefaultPrinter that can be used without additional configuration
var DefaultPrinter = Printer{
	Primary: func(args ...interface{}) string {
		return termenv.String(fmt.Sprint(args...)).Foreground(colors.DefaultColorTheme.Primary).String()
	},
	Secondary: func(args ...interface{}) string {
		return termenv.String(fmt.Sprint(args...)).Foreground(colors.DefaultColorTheme.Secondary).String()
	},
	Error: func(args ...interface{}) string {
		return termenv.String("error: " + fmt.Sprint(args...)).Foreground(colors.DefaultColorTheme.Error).String()
	},
	Warn: func(args ...interface{}) string {
		return termenv.String("warning: " + fmt.Sprint(args...)).Foreground(colors.DefaultColorTheme.Low).String()
	},
	Yellow: func(args ...interface{}) string {
		return termenv.String(fmt.Sprint(args...)).Foreground(colors.DefaultColorTheme.Low).String()
	},
	Disabled: func(args ...interface{}) string {
		return termenv.String(fmt.Sprint(args...)).Foreground(colors.DefaultColorTheme.Disabled).String()
	},
	Failed: func(args ...interface{}) string {
		return termenv.String(fmt.Sprint(args...)).Foreground(colors.DefaultColorTheme.Critical).String()
	},
	Success: func(args ...interface{}) string {
		return termenv.String(fmt.Sprint(args...)).Foreground(colors.DefaultColorTheme.Success).String()
	},
}

// PlainNoColorPrinter is a printer without colors
var PlainNoColorPrinter = Printer{
	Primary:   fmt.Sprint,
	Secondary: fmt.Sprint,
	Error:     fmt.Sprint,
	Warn:      fmt.Sprint,
	Disabled:  fmt.Sprint,
	Failed:    fmt.Sprint,
}

// H1 prints a headline
func (print *Printer) H1(headline string) string {
	var res bytes.Buffer
	res.WriteString(print.Primary(headline))
	res.WriteString("\n")
	res.WriteString(print.Primary(strings.Repeat("=", len(headline))))
	res.WriteString("\n\n")
	return res.String()
}

// H2 prints a headline
func (print *Printer) H2(headline string) string {
	var res bytes.Buffer
	res.WriteString(print.Primary(headline))
	res.WriteString("\n")
	res.WriteString(print.Primary(strings.Repeat("-", len(headline))))
	res.WriteString("\n\n")
	return res.String()
}
