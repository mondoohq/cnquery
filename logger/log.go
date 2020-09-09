// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package logger

import (
	"io"
	"os"

	"github.com/mattn/go-colorable"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Debug is set to true if the application is running in a debug mode
var Debug bool

func init() {
	Set("debug")
	// uses cli logger by default
	CliNoColorLogger()
}

// SetWriter configures a log writer for the global logger
func SetWriter(w io.Writer) {
	log.Logger = log.Output(w)
}

// UseJSONLogging for global logger
func UseJSONLogging() {
	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
}

// CliLogger sets the global logger to the console logger with color
func CliLogger() {
	// we use colerable to support color output on windows
	stderr := colorable.NewColorableStderr()
	log.Logger = NewConsoleWriter(stderr, false, false)
}

func CliCompactLogger() {
	// we use colerable to support color output on windows
	stderr := colorable.NewColorableStderr()
	log.Logger = NewConsoleWriter(stderr, false, true)
}

// CliNoColorLogger sets the global logger to the console logger without color
func CliNoColorLogger() {
	// we use colerable to support color output on windows
	stderr := colorable.NewColorableStderr()
	log.Logger = NewConsoleWriter(stderr, true, false)
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
