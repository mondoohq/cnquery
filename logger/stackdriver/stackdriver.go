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

// https://pkg.go.dev/cloud.google.com/go/logging
// by default, everything is logged async, only zerolog fatal messages are logged synchronously
func NewStackdriverWriter(projectID string, logID string) (zerolog.LevelWriter, error) {
	client, err := logging.NewClient(context.Background(), projectID)
	if err != nil {
		return nil, err
	}

	return &stackdriverWriter{
		logger: client.Logger(logID),
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
