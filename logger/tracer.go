// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package logger

import (
	"time"

	"github.com/rs/zerolog/log"
)

// FuncDur must be used in a `defer` statement. It receives the function name and the time
// when the function started and logs out the time it took to execute.
//
// ```go
//
//	func MyFunction() {
//		defer logger.FuncDur(time.Now(), "mypackage.MyFunction")
//
//		...
//	}
//
// ```
func FuncDur(start time.Time, name string) {
	log.Trace().Str("func", name).
		TimeDiff("took", time.Now(), start).
		Msgf("logger.FuncDur>")
}
