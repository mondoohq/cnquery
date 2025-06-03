// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workerpool

type worker[R any] struct {
	id        int
	queueCh   <-chan Task[R]
	resultsCh chan<- Result[R]
}

func (w *worker[R]) start() {
	go func() {
		for task := range w.queueCh {
			data, err := task()
			w.resultsCh <- Result[R]{data, err}
		}
	}()
}
