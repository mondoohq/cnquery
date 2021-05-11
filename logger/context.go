package logger

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const RequestIDFieldKey = "req-id"

// RequestScopedContext returns a context that contains a logger which logs the request ID
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
	return log.Ctx(ctx)
}
