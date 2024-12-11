// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workerpool

import (
	"github.com/cockroachdb/errors"
)

type Task[R any] func() (result R, err error)

type Pool[R any] struct {
	queue        chan Task[R]
	results      chan R
	errors       chan error
	workerCount  int
	requestsSent int
	requestsRead int

	err error
}

func New[R any](count int) *Pool[R] {
	return &Pool[R]{
		queue:       make(chan Task[R]),
		results:     make(chan R),
		errors:      make(chan error),
		workerCount: count,
	}
}

func (p *Pool[R]) Start() {
	for i := 0; i < p.workerCount; i++ {
		w := worker[R]{id: i, queue: p.queue, results: p.results, errors: p.errors}
		w.Start()
	}

	p.errorCollector()
}

func (p *Pool[R]) errorCollector() {
	go func() {
		for e := range p.errors {
			p.err = errors.Join(p.err, e)
		}
	}()
}

func (p *Pool[R]) GetError() error {
	return p.err
}

func (p *Pool[R]) Submit(t Task[R]) {
	p.queue <- t
	p.requestsSent++
}

func (p *Pool[R]) GetResult() R {
	defer func() {
		p.requestsRead++
	}()
	return <-p.results
}

func (p *Pool[R]) HasPendingRequests() bool {
	return p.requestsSent-p.requestsRead > 0
}

func (p *Pool[R]) Close() {
	close(p.queue)
}
