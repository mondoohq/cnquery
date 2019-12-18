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

// Debug is set to true if the application is running in a debug mode
var Debug bool

func init() {
	Set(false, true)
	// uses cli logger by default
	CliNoColorLogger()
}

// SetWriter configures a log writer for the global logger
func SetWriter(w io.Writer) {
	log.Logger = log.Output(w)
}

func UseJsonLogging() {
	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
}

func CliLogger() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func CliNoColorLogger() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true})
}

// Set will set up the logger
func Set(prod bool, debug bool) {
	Debug = debug
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// InitTestEnv will set all log configurations for a test environment
// verbose and colorful
func InitTestEnv() {
	Set(false, true)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}
