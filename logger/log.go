// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package logger

import (
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// we use colorable to support color output on windows
// we buffer it by default, so that tui components can interrupt cli logger
var LogOutputWriter = NewBufferedWriter(os.Stderr)

// Debug is set to true if the application is running in a debug mode
var Debug bool

// SetWriter configures a log writer for the global logger
func SetWriter(w io.Writer) {
	log.Logger = log.Output(w)
}

// UseJSONLogging for global logger
func UseJSONLogging(out io.Writer) {
	log.Logger = zerolog.New(out).With().Timestamp().Logger()
}

// CliLogger sets the global logger to the console logger with color
func CliLogger() {
	log.Logger = NewConsoleWriter(LogOutputWriter, false)
}

func CliCompactLogger(out io.Writer) {
	log.Logger = NewConsoleWriter(out, true)
}

// Set will set up the logger
func Set(level string) {
	switch level {
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	default:
		log.Error().Msg("unknown log level: " + level)
	}
}

// InitTestEnv will set all log configurations for a test environment
// verbose and colorful
func InitTestEnv() {
	Set("debug")
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}
