package jobpool

import (
	"sync"

	"github.com/pkg/errors"
)

/*
HOW TO USE THIS

in your primary resource function:

func (s *lumiAwsEc2) PRIMARY_RESOURCE_FUNCTION() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(FUNCTION THAT ACUTALLY CALLS AWS, 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
	}
	return res, nil
}

in the function that actually calls aws, the return object must be a slice of jobs:

func (s *lumiAwsEc2) getThingsFromAWS() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}} // return the error
	}
	regions := at.GetRegions()

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := atWithRegion.Ec2()

			AWS CALL AND HANDLING THE RESPONSE

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}
*/

type JobResult interface{}

// Job encapsulates a work item that should go in a work pool.
type Job struct {
	Err    error
	Result JobResult
	f      func() (JobResult, error)
}

// NewJob initializes a new job based on given params.
func NewJob(f func() (JobResult, error)) *Job {
	return &Job{f: f}
}

// Run runs a job and does appropriate accounting via a given sync.WorkGroup
func (t *Job) Run(wg *sync.WaitGroup) {
	t.Result, t.Err = t.f()
	wg.Done()
}

// Pool is a worker group that runs a number of jobs
type Pool struct {
	Jobs []*Job

	concurrency int // the amount of jobs to run concurrently
	jobsChan    chan *Job
	wg          sync.WaitGroup
}

// CreatePool takes a slice of jobs and a concurrency int, creating a channel to handle the jobs
func CreatePool(jobs []*Job, concurrency int) *Pool {
	return &Pool{
		Jobs:        jobs,
		concurrency: concurrency,
		jobsChan:    make(chan *Job),
	}
}

// HasErrors returns a bool base on the existence of errors in the job.
func (p *Pool) HasErrors() bool {
	for _, job := range p.Jobs {
		if job.Err != nil {
			return true
		}
	}
	return false
}

// GetErrors returns all errors from jobs run.
func (p *Pool) GetErrors() error {
	var err error
	for _, job := range p.Jobs {
		if job.Err != nil {
			err = errors.Wrap(job.Err, "job err: ")
		}
	}
	return err
}

// Run runs all work within the pool and blocks until it's finished.
func (p *Pool) Run() {

	for i := 0; i < p.concurrency; i++ {
		go p.work()
	}

	p.wg.Add(len(p.Jobs))
	for i := range p.Jobs {
		p.jobsChan <- p.Jobs[i]
	}

	// all workers return
	close(p.jobsChan)

	p.wg.Wait()
}

// The work loop for any single goroutine.
func (p *Pool) work() {
	for job := range p.jobsChan {
		job.Run(&p.wg)
	}
}
