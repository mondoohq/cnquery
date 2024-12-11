// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workerpool_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/internal/workerpool"
)

func TestPoolSubmitAndRetrieveResult(t *testing.T) {
	pool := workerpool.New[int](2)
	pool.Start()
	defer pool.Close()

	task := func() (int, error) {
		return 42, nil
	}

	// no requests
	assert.False(t, pool.HasPendingRequests())

	// submit a request
	pool.Submit(task)

	// should have pending requests
	assert.True(t, pool.HasPendingRequests())

	// assert results comes back
	result := pool.GetResult()
	assert.Equal(t, 42, result)

	// no more requests pending
	assert.False(t, pool.HasPendingRequests())

	// no errors
	assert.Nil(t, pool.GetError())
}

func TestPoolHandleErrors(t *testing.T) {
	pool := workerpool.New[int](5)
	pool.Start()
	defer pool.Close()

	// submit a task that will return an error
	task := func() (int, error) {
		return 0, errors.New("task error")
	}
	pool.Submit(task)

	// Wait for error collector to process
	time.Sleep(100 * time.Millisecond)

	err := pool.GetError()
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "task error")
	}
}

func TestPoolMultipleTasksWithErrors(t *testing.T) {
	type test struct {
		data int
	}
	pool := workerpool.New[*test](5)
	pool.Start()
	defer pool.Close()

	tasks := []workerpool.Task[*test]{
		func() (*test, error) { return &test{1}, nil },
		func() (*test, error) { return &test{2}, nil },
		func() (*test, error) {
			return nil, errors.New("task error")
		},
		func() (*test, error) { return &test{3}, nil },
	}

	for _, task := range tasks {
		pool.Submit(task)
	}

	var results []*test
	for range tasks {
		results = append(results, pool.GetResult())
	}

	assert.ElementsMatch(t, []*test{nil, &test{1}, &test{2}, &test{3}}, results)
	assert.False(t, pool.HasPendingRequests())

}

func TestPoolHandlesNilTasks(t *testing.T) {
	pool := workerpool.New[int](2)
	pool.Start()
	defer pool.Close()

	var nilTask workerpool.Task[int]
	pool.Submit(nilTask)

	// Wait for worker to process the nil task
	time.Sleep(100 * time.Millisecond)

	err := pool.GetError()
	assert.NoError(t, err)
}

func TestPoolHasPendingRequests(t *testing.T) {
	pool := workerpool.New[int](2)
	pool.Start()
	defer pool.Close()

	task := func() (int, error) {
		time.Sleep(50 * time.Millisecond)
		return 10, nil
	}

	pool.Submit(task)
	assert.True(t, pool.HasPendingRequests())

	result := pool.GetResult()
	assert.Equal(t, 10, result)
	assert.False(t, pool.HasPendingRequests())
}

func TestPoolClosesGracefully(t *testing.T) {
	pool := workerpool.New[int](1)
	pool.Start()

	task := func() (int, error) {
		time.Sleep(100 * time.Millisecond)
		return 42, nil
	}

	pool.Submit(task)

	pool.Close()

	// Ensure no panic occurs and channels are closed
	assert.PanicsWithError(t, "send on closed channel", func() {
		pool.Submit(task)
	})
}
