package logger

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const RequestIDFieldKey = "req-id"

type tagsCtxKey struct{}

type tags struct {
	m map[string]string
}

func AddTag(ctx context.Context, tagName string, value string) {
	if v, ok := ctx.Value(tagsCtxKey{}).(*tags); ok {
		v.m[tagName] = value
	}
}

func GetTags(ctx context.Context) map[string]string {
	if v, ok := ctx.Value(tagsCtxKey{}).(*tags); ok {
		return v.m
	}
	return nil
}

func WithTagsContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, tagsCtxKey{}, &tags{m: map[string]string{}})
}

// RequestScopedContext returns a context that contains a logger which logs the request ID
// Given a context, a logger can be retrieved as follows
//
//	ctx := RequestScopedContext(context.Background(), "req-id")
//	log := FromContext(ctx)
//	log.Debug().Msg("hello")
func RequestScopedContext(ctx context.Context, reqID string) context.Context {
	if reqID == "" {
		// The leading underscore indicates the request id was generated on the
		// server instead of the client. This could be temporary and be useful
		// for debugging which client calls are not passing the request id
		reqID = "_" + uuid.New().String()
	}
	l := log.With().Str(RequestIDFieldKey, reqID).Logger()
	return WithTagsContext(l.WithContext(ctx))
}

// FromContext returns the logger in the context if present, otherwise the it
// returns the default logger
func FromContext(ctx context.Context) *zerolog.Logger {
	l := log.Ctx(ctx)
	if l.GetLevel() == zerolog.Disabled {
		// If a context logger was not set, we'll return a global
		// logger instead of the default noop logger
		l := log.With().Str(RequestIDFieldKey, "global").Logger()
		return &l
	}
	tags := GetTags(ctx)
	if len(tags) > 0 {
		dict := zerolog.Dict()
		for k, v := range tags {
			dict.Str(k, v)
		}
		lv := l.With().Dict("ctags", dict).Logger()
		return &lv
	}
	return l
}
