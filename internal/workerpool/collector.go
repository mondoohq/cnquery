// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workerpool

import (
	"sync"
	"sync/atomic"
)

type collector[R any] struct {
	resultsCh <-chan Result[R]
	results   []Result[R]
	read      sync.Mutex

	// The total number of requests read.
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
			}

			atomic.AddInt64(&c.requestsRead, 1)
		}
	}()
}

func (c *collector[R]) RequestsRead() int64 {
	return atomic.LoadInt64(&c.requestsRead)
}

func (c *collector[R]) GetResults() []Result[R] {
	c.read.Lock()
	defer c.read.Unlock()
	return c.results
}

func (c *collector[R]) GetValues() (slice []R) {
	results := c.GetResults()
	for i := range results {
		slice = append(slice, results[i].Value)
	}
	return
}

func (c *collector[R]) GetErrors() (slice []error) {
	results := c.GetResults()
	for i := range results {
		slice = append(slice, results[i].Error)
	}
	return
}
