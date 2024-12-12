// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workerpool

import (
	"sync"
	"sync/atomic"
)

type collector[R any] struct {
	resultsCh <-chan R
	results   []R
	read      sync.Mutex

	errorsCh <-chan error
	errors   []error

	requestsRead int64
}

func (c *collector[R]) start() {
	go func() {
		for {
			select {
			case result := <-c.resultsCh:
				c.read.Lock()
				c.results = append(c.results, result)
				c.read.Unlock()

			case err := <-c.errorsCh:
				c.read.Lock()
				c.errors = append(c.errors, err)
				c.read.Unlock()
			}

			atomic.AddInt64(&c.requestsRead, 1)
		}
	}()
}
func (c *collector[R]) GetResults() []R {
	c.read.Lock()
	defer c.read.Unlock()
	return c.results
}

func (c *collector[R]) GetErrors() []error {
	c.read.Lock()
	defer c.read.Unlock()
	return c.errors
}

func (c *collector[R]) RequestsRead() int64 {
	return atomic.LoadInt64(&c.requestsRead)
}
