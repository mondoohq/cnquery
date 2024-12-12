// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workerpool

type worker[R any] struct {
	id        int
	queueCh   <-chan Task[R]
	resultsCh chan<- R
	errorsCh  chan<- error
}

func (w *worker[R]) start() {
	go func() {
		for task := range w.queueCh {
			if task == nil {
				// let the collector know we processed the request
				w.errorsCh <- nil
				continue
			}

			data, err := task()
			if err != nil {
				w.errorsCh <- err
			} else {
				w.resultsCh <- data
			}
		}
	}()
}
