// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"io"
	"log"

	"github.com/hashicorp/go-hclog"
	"github.com/rs/zerolog"
)

// wrap zerolog to hclogger
type hclogger struct {
	zerolog.Logger
}

var _ hclog.Logger = (*hclogger)(nil)

func (l *hclogger) IsTrace() bool {
	return l.Logger.GetLevel() == zerolog.TraceLevel
}

func (l *hclogger) IsDebug() bool {
	return l.Logger.GetLevel() == zerolog.DebugLevel
}

func (l *hclogger) IsInfo() bool {
	return l.Logger.GetLevel() == zerolog.InfoLevel
}

func (l *hclogger) IsWarn() bool {
	return l.Logger.GetLevel() == zerolog.WarnLevel
}

func (l *hclogger) IsError() bool {
	return l.Logger.GetLevel() == zerolog.ErrorLevel
}

func (l *hclogger) Trace(format string, args ...interface{}) {
	l.Logger.Trace().Fields(args).Msg(format)
}

func (l *hclogger) Debug(format string, args ...interface{}) {
	l.Logger.Debug().Fields(args).Msg(format)
}

func (l *hclogger) Info(format string, args ...interface{}) {
	l.Logger.Info().Fields(args).Msg(format)
}

func (l *hclogger) Warn(format string, args ...interface{}) {
	l.Logger.Warn().Fields(args).Msg(format)
}

func (l *hclogger) Error(format string, args ...interface{}) {
	l.Logger.Error().Fields(args).Msg(format)
}

func (l *hclogger) Log(level hclog.Level, format string, args ...interface{}) {
	switch level {
	case hclog.Trace:
		l.Logger.Trace().Fields(args).Msg(format)
	case hclog.Debug:
		l.Logger.Debug().Fields(args).Msg(format)
	case hclog.Info:
		l.Logger.Info().Fields(args).Msg(format)
	case hclog.Warn:
		l.Logger.Warn().Fields(args).Msg(format)
	case hclog.Error:
		l.Logger.Error().Fields(args).Msg(format)
	default:
		log.Fatalf("unknown level %d", level)
	}
}

func (l *hclogger) SetLevel(level hclog.Level) {
	switch level {
	case hclog.Trace:
		l.Logger = l.Logger.Level(zerolog.TraceLevel)
	case hclog.Debug:
		l.Logger = l.Logger.Level(zerolog.DebugLevel)
	case hclog.Info:
		l.Logger = l.Logger.Level(zerolog.InfoLevel)
	case hclog.Warn:
		l.Logger = l.Logger.Level(zerolog.WarnLevel)
	case hclog.Error:
		l.Logger = l.Logger.Level(zerolog.ErrorLevel)
	default:
		log.Fatalf("unknown level %d", level)
	}
}

// Returns the current level
func (l *hclogger) GetLevel() hclog.Level {
	switch l.Logger.GetLevel() {
	case zerolog.TraceLevel:
		return hclog.Trace
	case zerolog.DebugLevel:
		return hclog.Debug
	case zerolog.InfoLevel:
		return hclog.Info
	case zerolog.WarnLevel:
		return hclog.Warn
	case zerolog.ErrorLevel:
		return hclog.Error
	default:
		log.Fatalf("unknown level %d", l.Logger.GetLevel())
	}
	return hclog.Trace
}

func (l *hclogger) Name() string {
	return ""
}

func (l *hclogger) Named(name string) hclog.Logger {
	return &hclogger{l.Logger.With().Str("name", name).Logger()}
}

func (l *hclogger) ResetNamed(name string) hclog.Logger {
	return &hclogger{l.Logger.With().Str("name", name).Logger()}
}

func (l *hclogger) With(args ...interface{}) hclog.Logger {
	return &hclogger{l.Logger.With().Fields(args).Logger()}
}

func (l *hclogger) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	return log.New(l.Logger, "", 0)
}

func (l *hclogger) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	return l.Logger
}

func (l *hclogger) ImpliedArgs() []interface{} {
	return nil
}
