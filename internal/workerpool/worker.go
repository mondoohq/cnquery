// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workerpool

type worker[R any] struct {
	id      int
	queue   <-chan Task[R]
	results chan<- R
	errors  chan<- error
}

func (w *worker[R]) Start() {
	go func() {
		for task := range w.queue {
			if task == nil {
				continue
			}

			data, err := task()
			if err != nil {
				w.errors <- err
			}

			w.results <- data
		}
	}()
}
