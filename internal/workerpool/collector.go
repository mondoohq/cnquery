// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workerpool

type collector[R any] struct {
	resultsCh <-chan R
	results   []R

	errorsCh <-chan error
	errors   []error

	requestsRead int64
}

func (c *collector[R]) Start() {
	go func() {
		for {
			select {
			case result := <-c.resultsCh:
				c.results = append(c.results, result)

			case err := <-c.errorsCh:
				c.errors = append(c.errors, err)
			}

			c.requestsRead++
		}
	}()
}

func (c *collector[R]) RequestsRead() int64 {
	return c.requestsRead
}
