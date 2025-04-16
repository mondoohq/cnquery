package jobpool_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.uber.org/goleak"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// verify that we are not leaking goroutines
	goleak.VerifyTestMain(m)
}

func TestNewJob(t *testing.T) {
	f := func() (jobpool.JobResult, error) {
		return 42, nil
	}
	job := jobpool.NewJob(f)
	assert.NotNil(t, job)
	assert.Nil(t, job.Result)
	assert.Nil(t, job.Err)
}

func TestJobRunWithFunction(t *testing.T) {
	f := func() (jobpool.JobResult, error) {
		return "done", nil
	}
	job := jobpool.NewJob(f)

	var wg sync.WaitGroup
	wg.Add(1)
	job.Run(&wg)
	wg.Wait()

	assert.Equal(t, "done", job.Result)
	assert.Nil(t, job.Err)
}

func TestJobRunWithNilFunction(t *testing.T) {
	job := jobpool.NewJob(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	job.Run(&wg)
	wg.Wait()

	assert.Error(t, job.Err)
	assert.Nil(t, job.Result)
	assert.Contains(t, job.Err.Error(), "no function to run")
}

func TestPoolHasErrors(t *testing.T) {
	jobs := []*jobpool.Job{
		jobpool.NewJob(func() (jobpool.JobResult, error) { return nil, nil }),
		jobpool.NewJob(func() (jobpool.JobResult, error) { return nil, errors.New("fail") }),
	}

	pool := jobpool.CreatePool(jobs, 2)
	pool.Run()

	assert.True(t, pool.HasErrors())
}

func TestPoolGetErrors(t *testing.T) {
	jobs := []*jobpool.Job{
		jobpool.NewJob(func() (jobpool.JobResult, error) { return nil, errors.New("one") }),
		jobpool.NewJob(func() (jobpool.JobResult, error) { return nil, errors.New("two") }),
	}

	pool := jobpool.CreatePool(jobs, 2)
	pool.Run()

	err := pool.GetErrors()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "one")
	assert.Contains(t, err.Error(), "two")
}

func TestPoolRunExecutesAllJobs(t *testing.T) {
	count := int32(0)
	jobs := make([]*jobpool.Job, 10)

	for i := range jobs {
		jobs[i] = jobpool.NewJob(func() (jobpool.JobResult, error) {
			atomic.AddInt32(&count, 1)
			return nil, nil
		})
	}

	pool := jobpool.CreatePool(jobs, 5)
	pool.Run()

	assert.Equal(t, int32(len(jobs)), count)
}

func TestPoolConcurrencySafety(t *testing.T) {
	var maxConcurrent int32
	var current int32
	jobs := make([]*jobpool.Job, 50)

	for i := range jobs {
		jobs[i] = jobpool.NewJob(func() (jobpool.JobResult, error) {
			v := atomic.AddInt32(&current, 1)
			atomic.CompareAndSwapInt32(&maxConcurrent, maxConcurrent, max(maxConcurrent, v))

			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&current, -1)

			return nil, nil
		})
	}

	pool := jobpool.CreatePool(jobs, 10)
	pool.Run()

	assert.LessOrEqual(t, maxConcurrent, int32(10))
}

func BenchmarkPoolRun_10(b *testing.B)  { benchmarkPoolRun(b, 10) }
func BenchmarkPoolRun_100(b *testing.B) { benchmarkPoolRun(b, 100) }
func BenchmarkPoolRun_500(b *testing.B) { benchmarkPoolRun(b, 500) }

func benchmarkPoolRun(b *testing.B, n int) {
	jobs := make([]*jobpool.Job, n)
	for i := 0; i < n; i++ {
		jobs[i] = jobpool.NewJob(func() (jobpool.JobResult, error) {
			return nil, nil
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool := jobpool.CreatePool(jobs, 10)
		pool.Run()
	}
}

// Utility for Go 1.21+ without importing unsafe
func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
