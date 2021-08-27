package logger

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const RequestIDFieldKey = "req-id"

var GlobalLogger = log.With().Str(RequestIDFieldKey, "global").Logger()

// RequestScopedContext returns a context that contains a logger which logs the request ID
// Given a context, a logger can be retrieved as follows
//  ctx := RequestScopedContext(context.Background(), "req-id")
//  log := FromContext(ctx)
//  log.Debug().Msg("hello")
func RequestScopedContext(ctx context.Context, reqID string) context.Context {
	if reqID == "" {
		// The leading underscore indicates the request id was generated on the
		// server instead of the client. This could be temporary and be useful
		// for debugging which client calls are not passing the request id
		reqID = "_" + uuid.New().String()
	}
	l := log.With().Str(RequestIDFieldKey, reqID).Logger()
	return l.WithContext(ctx)
}

// FromContext returns the logger in the context if present, otherwise the it
// returns the default logger
func FromContext(ctx context.Context) *zerolog.Logger {
	l := log.Ctx(ctx)
	if l.GetLevel() == zerolog.Disabled {
		// If a context logger was not set, we'll return our global
		// logger instead of the default noop logger
		return &GlobalLogger
	}
	return l
}
