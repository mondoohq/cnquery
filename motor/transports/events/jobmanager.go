package events

import (
	"errors"
	"sync"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/transports"
)

// the job state
type JobState int32

const (
	// pending is the default state
	Job_PENDING    JobState = 0
	Job_RUNNING    JobState = 1
	Job_TERMINATED JobState = 2
)

type Job struct {
	ID string

	Runnable func(transports.Transport) (transports.Observable, error)
	Callback []func(transports.Observable)

	State        JobState
	ScheduledFor time.Time
	Interval     time.Duration
	// -1 means infinity
	Repeat int32

	Metrics struct {
		RunAt     time.Time
		Duration  time.Duration
		Count     int32
		Errors    int32
		Successes int32
	}
}

func (j *Job) sanitize() error {
	// ensure we have an id
	if len(j.ID) == 0 {
		j.ID = uuid.Must(uuid.NewV4()).String()
	}

	// verify that the interval is set for the job, otherwise overwrite with the default
	if j.Interval == 0 {
		j.Interval = time.Duration(60 * time.Second)
	}

	// verify that we have the required things for a schedule
	if j.ScheduledFor.Before(time.Now().Add(time.Duration(-10 * time.Second))) {
		return errors.New("schedule for the past")
	}

	if j.Runnable == nil {
		return errors.New("no runnable defined")
	}

	if len(j.Callback) == 0 {
		return errors.New("no callback defined")
	}

	return nil
}

func (j *Job) SetInfinity() {
	j.Repeat = -1
}

func (j *Job) isPending() bool {
	return j.State == Job_PENDING
}

func NewJobManager(transport transports.Transport) *JobManager {
	jm := &JobManager{transport: transport, jobs: &Jobs{}}
	jm.jobSelectionMutex = &sync.Mutex{}
	jm.quit = make(chan chan struct{})
	jm.Serve()
	return jm
}

type JobManagerMetrics struct {
	Jobs int
}

// Jobs is a map to store all jobs
type Jobs struct{ sync.Map }

// Store a new job
func (c *Jobs) Store(k string, v *Job) {
	c.Map.Store(k, v)
}

// Load a job
func (c *Jobs) Load(k string) (*Job, bool) {
	res, ok := c.Map.Load(k)
	if !ok {
		return nil, ok
	}
	return res.(*Job), ok
}

func (c *Jobs) Range(f func(string, *Job) bool) {
	c.Map.Range(func(key interface{}, value interface{}) bool {
		return f(key.(string), value.(*Job))
	})
}

func (c *Jobs) Len() int {
	i := 0
	c.Range(func(k string, j *Job) bool {
		i++
		return true
	})
	return i
}

func (c *Jobs) Delete(k string) {
	c.Map.Delete(k)
}

type JobManager struct {
	transport         transports.Transport
	quit              chan chan struct{}
	jobSelectionMutex *sync.Mutex
	jobs              *Jobs
	jobMetrics        JobManagerMetrics
}

// Schedule stores the job in the run list and sanitize the job before execution
func (jm *JobManager) Schedule(job *Job) (string, error) {
	// ensure all defaults are set
	err := job.sanitize()
	if err != nil {
		return "", err
	}

	log.Debug().Str("jobid", job.ID).Msg("motor.job> schedule new job")

	// store job, with a mutex
	jm.jobs.Store(job.ID, job)

	// return job id
	return job.ID, nil
}

func (jm *JobManager) GetJob(jobid string) (*Job, error) {
	job, ok := jm.jobs.Load(jobid)
	if !ok {
		return nil, errors.New("job " + jobid + " does not exist")
	}
	return job, nil
}

func (jm *JobManager) Delete(jobid string) {
	log.Debug().Str("jobid", jobid).Msg("motor.job> delete job")
	jm.jobs.Delete(jobid)
}

func (jm *JobManager) Metrics() *JobManagerMetrics {
	jm.jobMetrics.Jobs = jm.jobs.Len()
	return &jm.jobMetrics
}

// Serve creates a goroutine and runs jobs in the background
func (jm *JobManager) Serve() {
	// create a new channel and starte a new go routine
	go func() {
		for {
			select {
			case doneChan := <-jm.quit:
				close(doneChan)
				return
			default:
				// fetch job
				job, err := jm.nextJob()

				if err == nil {
					// run job
					jm.Run(job)

					// if repeat is 0 and it is not the last iteration of a reoccuring task,
					// we need to remove the job
					if job.Repeat == 0 && job.State == Job_TERMINATED {
						jm.Delete(job.ID)
					}
				}

				// TODO: wake up, when new jobs come in
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

func (jm *JobManager) Run(job *Job) {
	log.Debug().Str("jobid", job.ID).Msg("motor.job> run job")
	job.Metrics.RunAt = time.Now()

	// execute job
	observable, err := job.Runnable(jm.transport)

	// update metrics
	job.Metrics.Count = job.Metrics.Count + 1
	if err != nil {
		job.Metrics.Errors = job.Metrics.Errors + 1
	} else {
		job.Metrics.Successes = job.Metrics.Successes + 1
	}

	// determine the next run or delete the job
	if job.Repeat != 0 {
		job.ScheduledFor = time.Now().Add(job.Interval)
		log.Debug().Str("jobid", job.ID).Time("time", job.ScheduledFor).Msg("motor.job> scheduled job for the future")
		job.State = Job_PENDING
	} else {
		log.Debug().Str("jobid", job.ID).Msg("motor.job> last run for this job, yeah")
		job.State = Job_TERMINATED
	}

	// if we have a positive repeat, we need to decrement
	if job.Repeat > 0 {
		job.Repeat = job.Repeat - 1
	}

	// calc duration
	job.Metrics.Duration = time.Now().Sub(job.Metrics.RunAt)
	log.Debug().Str("jobid", job.ID).Msg("motor.job> completed")

	// send observable to all subscribers
	// since this call is synchronous in the same go routine, we need to do this as the last step, to ensure
	// all job planning is completed before a potential canceling comes in.
	log.Debug().Str("jobid", job.ID).Msg("motor.job> call subscriber")
	for _, subscriber := range job.Callback {
		subscriber(observable)
	}

}

// nextJob looks for the oldest job and does that one first
func (jm *JobManager) nextJob() (*Job, error) {
	// use lock to prevent concurrent access on that list
	var oldestJob *Job
	oldest := time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

	// iterate over list of jobs of pending jobs and find the oldest one
	jm.jobSelectionMutex.Lock()
	now := time.Now()

	jm.jobs.Range(func(k string, job *Job) bool {
		if job.State == Job_PENDING && oldest.After(job.ScheduledFor) && job.ScheduledFor.Before(now) {
			oldest = job.ScheduledFor
			oldestJob = job
		}
		return true
	})

	// set the job to running to ensure other parallel go routines do not fetch the same job
	if oldestJob != nil {
		oldestJob.State = Job_RUNNING
	}
	jm.jobSelectionMutex.Unlock()

	if oldestJob == nil {
		return nil, errors.New("no job available")
	}

	// extrats the next run from the nextruns
	return oldestJob, nil
}

// TeadDown deletes all
func (jm *JobManager) TearDown() {
	log.Debug().Msg("motor.job> tear down")
	// ensures the go routines are canceled
	done := make(chan struct{})
	jm.quit <- done
	<-done
}
