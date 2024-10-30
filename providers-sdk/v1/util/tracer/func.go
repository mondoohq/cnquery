// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tracer

import (
	"time"

	"github.com/rs/zerolog/log"
)

// FuncDur must be used in a `defer` statement. It receives the function name and the time
// when a function started and logs out the time it took to run.
//
// ```go
//
//	func MyFunction() {
//		defer tracer.FuncDur(time.Now(), "mypackage.MyFunction")
//
//		...
//	}
//
// ```
func FuncDur(start time.Time, name string) {
	log.Trace().Str("func", name).
		TimeDiff("took", time.Now(), start).
		Msgf("tracer.FuncDur>")
}
