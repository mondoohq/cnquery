// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package events

import (
	"io/ioutil"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go.mondoo.com/cnquery/motor/providers/os"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/mock"
)

func SetupTest() *JobManager {
	filepath, _ := filepath.Abs("testdata/watcher_test.toml")
	trans, _ := mock.NewFromTomlFile(filepath)
	return NewJobManager(trans)
}

func TeardownTest(jm *JobManager) {
	jm.TearDown()
}

func TestJobCreation(t *testing.T) {
	jm := SetupTest()
	jobid, err := jm.Schedule(&Job{
		ID:           "command",
		ScheduledFor: time.Now(),
		Interval:     time.Duration(10 * time.Second),
		Repeat:       5,
		Runnable: func(m os.OperatingSystemProvider) (providers.Observable, error) {
			cmd, _ := m.RunCommand("hostname")
			return &CommandObservable{Result: cmd}, nil
		},
		Callback: []func(o providers.Observable){
			func(o providers.Observable) {
				// noop
			},
		},
	})

	assert.NotNil(t, jobid, "job is scheduled")
	assert.Nil(t, err, "job could be scheduled without any error")

	job, err := jm.GetJob(jobid)
	assert.NotNil(t, job, "able to retrieve the job")
	assert.Nil(t, err, "job could be retrieved without any error")

	TeardownTest(jm)
}

func TestJobDeletion(t *testing.T) {
	jm := SetupTest()

	assert.Equal(t, 0, jm.Metrics().Jobs, "no job is scheduled")

	// schedule a new job
	jobid, err := jm.Schedule(&Job{
		ID:           "command",
		ScheduledFor: time.Now(),
		Interval:     time.Duration(10 * time.Second),
		Repeat:       5,
		Runnable: func(m os.OperatingSystemProvider) (providers.Observable, error) {
			cmd, _ := m.RunCommand("hostname")
			return &CommandObservable{Result: cmd}, nil
		},
		Callback: []func(o providers.Observable){
			func(o providers.Observable) {
				// noop
			},
		},
	})
	assert.Nil(t, err, "job was scheduled without any error")

	// verify that the job is stored with the ID
	job, err := jm.GetJob(jobid)
	assert.Nil(t, err, "job was retrieved without any error")
	assert.NotNil(t, job, "job could be retrieved")

	// cancel the job
	jm.Delete(jobid)
	assert.Nil(t, err, "job could be deleted without any error")

	// verify that the job is not there anymore
	job, err = jm.GetJob(jobid)
	assert.NotNil(t, err, "job could not be retrieved")
	assert.Nil(t, job)

	assert.Equal(t, 0, jm.Metrics().Jobs, "no job is scheduled")
	TeardownTest(jm)
}

func TestRejectEmptyJob(t *testing.T) {
	jm := SetupTest()

	assert.Equal(t, 0, jm.Metrics().Jobs, "no job is scheduled")

	// schedule a new job
	id, err := jm.Schedule(&Job{})
	assert.Equal(t, 0, len(id), "job is not scheduled")
	assert.NotNil(t, err, "job schedule returns an error")

	assert.Equal(t, 0, jm.Metrics().Jobs, "no job is scheduled")
	TeardownTest(jm)
}

func TestCommandJob(t *testing.T) {
	var wg sync.WaitGroup
	jm := SetupTest()

	var res *CommandObservable
	wg.Add(1)
	_, err := jm.Schedule(&Job{
		ID:           "command-abc",
		ScheduledFor: time.Now(),
		Interval:     time.Duration(10 * time.Second),
		Repeat:       5,
		Runnable: func(m os.OperatingSystemProvider) (providers.Observable, error) {
			cmd, _ := m.RunCommand("hostname")
			return &CommandObservable{Result: cmd}, nil
		},
		Callback: []func(o providers.Observable){
			func(o providers.Observable) {
				defer wg.Done()
				switch x := o.(type) {
				case *CommandObservable:
					res = x
				}
			},
		},
	})
	require.NoError(t, err)

	wg.Wait()

	stdout, err := ioutil.ReadAll(res.Result.Stdout)
	assert.Nil(t, err, "could extract stdout")
	assert.Equal(t, "mockland.local", string(stdout), "get the expected command output")
	TeardownTest(jm)
}

func TestFileJob(t *testing.T) {
	var wg sync.WaitGroup
	jm := SetupTest()
	path := "/tmp/test"
	var res *FileObservable
	wg.Add(1)
	_, err := jm.Schedule(&Job{
		ID:           "file-abc",
		ScheduledFor: time.Now(),
		Interval:     time.Duration(10 * time.Second),
		Runnable: func(m os.OperatingSystemProvider) (providers.Observable, error) {
			file, _ := m.FS().Open(path)
			return &FileObservable{File: file, FileOp: Modify}, nil
		},
		Callback: []func(o providers.Observable){
			func(o providers.Observable) {
				defer wg.Done()
				switch x := o.(type) {
				case *FileObservable:
					res = x
				}
			},
		},
	})
	require.NoError(t, err)
	wg.Wait()
	assert.Equal(t, path, res.File.Name(), "get the expected file")
	assert.Equal(t, Modify, res.FileOp, "get the expected file event")
	TeardownTest(jm)
}

func TestScheduleRepeating(t *testing.T) {
	var wg sync.WaitGroup
	jm := SetupTest()

	var res *CommandObservable

	wg.Add(2)
	// one call is executed at the scheduled time
	_, err := jm.Schedule(&Job{
		ID:           "command-abc",
		ScheduledFor: time.Now(),
		Repeat:       1,
		Interval:     time.Duration(1),
		Runnable: func(m os.OperatingSystemProvider) (providers.Observable, error) {
			cmd, _ := m.RunCommand("hostname")
			return &CommandObservable{Result: cmd}, nil
		},
		Callback: []func(o providers.Observable){
			func(o providers.Observable) {
				defer wg.Done()

				switch x := o.(type) {
				case *CommandObservable:
					res = x
				}
			},
		},
	})
	assert.Nil(t, err, "job was scheduled without any error")
	wg.Wait()

	// check that the result expects the outcome
	stdout, err := ioutil.ReadAll(res.Result.Stdout)
	assert.Nil(t, err, "could extract stdout")
	assert.Equal(t, "mockland.local", string(stdout), "get the expected command output")
	TeardownTest(jm)
}
