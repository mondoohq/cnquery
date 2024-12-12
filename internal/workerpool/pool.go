// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workerpool

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors"
)

type Task[R any] func() (result R, err error)

// Pool is a generic pool of workers.
type Pool[R any] struct {
	queueCh   chan Task[R]
	resultsCh chan R
	errorsCh  chan error

	requestsSent int64
	once         sync.Once

	workers     []*worker[R]
	workerCount int

	collector[R]
}

// New initializes a new Pool with the provided number of workers. The pool is generic and can
// accept any type of Task that returns the signature `func() (R, error)`.
//
// For example, a Pool[int] will accept Tasks similar to:
//
//	task := func() (int, error) {
//		return 42, nil
//	}
func New[R any](count int) *Pool[R] {
	resultsCh := make(chan R)
	errorsCh := make(chan error)
	return &Pool[R]{
		queueCh:     make(chan Task[R]),
		resultsCh:   resultsCh,
		errorsCh:    errorsCh,
		workerCount: count,
		collector:   collector[R]{resultsCh: resultsCh, errorsCh: errorsCh},
	}
}

// Start the pool workers and collector. Make sure call `Close()` to clear the pool.
//
//	pool := workerpool.New[int](10)
//	pool.Start()
//	defer pool.Close()
func (p *Pool[R]) Start() {
	p.once.Do(func() {
		for i := 0; i < p.workerCount; i++ {
			w := worker[R]{id: i, queueCh: p.queueCh, resultsCh: p.resultsCh, errorsCh: p.errorsCh}
			w.start()
			p.workers = append(p.workers, &w)
		}

		p.collector.start()
	})
}

// Submit sends a task to the workers
func (p *Pool[R]) Submit(t Task[R]) {
	p.queueCh <- t
	atomic.AddInt64(&p.requestsSent, 1)
}

// GetErrors returns any error from a processed task
func (p *Pool[R]) GetErrors() error {
	return errors.Join(p.collector.GetErrors()...)
}

// GetResults returns the tasks results.
//
// It is recommended to call `Wait()` before reading the results.
func (p *Pool[R]) GetResults() []R {
	return p.collector.GetResults()
}

// Close waits for workers and collector to process all the requests, and then closes
// the task queue channel. After closing the pool, calling `Submit()` will panic.
func (p *Pool[R]) Close() {
	p.Wait()
	close(p.queueCh)
}

// Wait waits until all tasks have been processed.
func (p *Pool[R]) Wait() {
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		if !p.Processing() {
			return
		}
		<-ticker.C
	}
}

// PendingRequests returns the number of pending requests.
func (p *Pool[R]) PendingRequests() int64 {
	return atomic.LoadInt64(&p.requestsSent) - p.collector.RequestsRead()
}

// Processing return true if tasks are being processed.
func (p *Pool[R]) Processing() bool {
	return p.PendingRequests() != 0
}
