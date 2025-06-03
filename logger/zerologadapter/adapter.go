// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package zerologadapter

import "github.com/rs/zerolog"

// New returns a new adapter for the zerolog logger to the LeveledLogger interface. This struct is
// mainly used in conjunction with a retryable http client to convert all retry logs to debug logs.
//
// NOTE that all messages will go to debug level.
//
// e.g.
// ```go
// retryClient := retryablehttp.NewClient()
// retryClient.Logger = zerologadapter.New(log.Logger)
// ```
func New(logger zerolog.Logger) *Adapter {
	return &Adapter{logger}
}

type Adapter struct {
	logger zerolog.Logger
}

func (z *Adapter) Msg(msg string, keysAndValues ...interface{}) {
	z.logger.Debug().Fields(convertToFields(keysAndValues...)).Msg(msg)
}

func (z *Adapter) Error(msg string, keysAndValues ...interface{}) {
	z.logger.Debug().Fields(convertToFields(keysAndValues...)).Msg(msg)
}

func (z *Adapter) Info(msg string, keysAndValues ...interface{}) {
	z.logger.Debug().Fields(convertToFields(keysAndValues...)).Msg(msg)
}

func (z *Adapter) Debug(msg string, keysAndValues ...interface{}) {
	z.logger.Debug().Fields(convertToFields(keysAndValues...)).Msg(msg)
}

func (z *Adapter) Warn(msg string, keysAndValues ...interface{}) {
	z.logger.Debug().Fields(convertToFields(keysAndValues...)).Msg(msg)
}

func convertToFields(keysAndValues ...interface{}) map[string]interface{} {
	fields := make(map[string]interface{})
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			keyString, ok := keysAndValues[i].(string)
			if ok { // safety first, even though we always expect a string
				fields[keyString] = keysAndValues[i+1]
			}
		}
	}
	return fields
}
