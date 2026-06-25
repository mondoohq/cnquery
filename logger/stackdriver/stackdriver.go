// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackdriver

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/logging"
	"github.com/rs/zerolog"
)

// LogLevelMap maps zerolog.Level to logging.Severity
var logLevelMap = map[zerolog.Level]logging.Severity{
	zerolog.DebugLevel: logging.Debug,
	zerolog.InfoLevel:  logging.Info,
	zerolog.WarnLevel:  logging.Warning,
	zerolog.ErrorLevel: logging.Error,
	zerolog.FatalLevel: logging.Critical,
	zerolog.PanicLevel: logging.Critical,
	zerolog.NoLevel:    logging.Info,
	zerolog.TraceLevel: logging.Debug,
}

// Option configures the stackdriver writer.
type Option func(*options)

type options struct {
	labels map[string]string
}

// WithLabels attaches the given key/value labels to every log entry written by
// this writer (via logging.CommonLabels). Useful to tag logs with metadata such
// as environment, team, or asset identifiers so they can be filtered in the GCP
// Logs Explorer (e.g. `labels.env = "prod"`).
func WithLabels(labels map[string]string) Option {
	return func(o *options) {
		o.labels = labels
	}
}

// https://pkg.go.dev/cloud.google.com/go/logging
// by default, everything is logged async, only zerolog fatal messages are logged synchronously
func NewStackdriverWriter(projectID string, logID string, opts ...Option) (zerolog.LevelWriter, error) {
	o := &options{}
	for _, fn := range opts {
		fn(o)
	}

	client, err := logging.NewClient(context.Background(), projectID)
	if err != nil {
		return nil, err
	}

	var loggerOpts []logging.LoggerOption
	if len(o.labels) > 0 {
		loggerOpts = append(loggerOpts, logging.CommonLabels(o.labels))
	}

	return &stackdriverWriter{
		logger: client.Logger(logID, loggerOpts...),
	}, nil
}

type stackdriverWriter struct {
	logger *logging.Logger
	zerolog.LevelWriter
}

func (c *stackdriverWriter) Write(p []byte) (int, error) {
	c.logger.Log(logging.Entry{
		Severity: logging.Info, // if no level is provided, we assume its info
		Payload:  json.RawMessage(p),
	})
	return len(p), nil
}

func (c *stackdriverWriter) WriteLevel(level zerolog.Level, payload []byte) (int, error) {
	entry := logging.Entry{
		Severity: logLevelMap[level],
		Payload:  json.RawMessage(payload),
	}

	if level == zerolog.FatalLevel {
		// since its fatal, we want to make sure its data is transferred
		err := c.logger.LogSync(context.Background(), entry)
		if err != nil {
			return 0, err
		}
		// prepare the logger to be closed
		err = c.logger.Flush()
		if err != nil {
			return 0, err
		}
	} else {
		c.logger.Log(entry)
	}
	return len(payload), nil
}
