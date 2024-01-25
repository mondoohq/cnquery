// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package executor

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10"
	"go.mondoo.com/cnquery/v10/cli/progress"
	"go.mondoo.com/cnquery/v10/explorer"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/mqlc"
	"go.mondoo.com/cnquery/v10/utils/multierr"
)

func RunExecutionJob(
	runtime llx.Runtime, collectorSvc explorer.QueryConductor, assetMrn string,
	job *explorer.ExecutionJob, features cnquery.Features, progressReporter progress.Progress,
) (*instance, error) {
	// We are setting a sensible default timeout for jobs here. This will need
	// user-configuration.
	timeout := 30 * time.Minute

	res := newInstance(runtime, progressReporter)
	res.assetMrn = assetMrn
	res.collector = collectorSvc
	res.datapoints = job.Datapoints

	return res, res.runCode(job.Queries, timeout)
}

func ExecuteFilterQueries(runtime llx.Runtime, queries []*explorer.Mquery, timeout time.Duration) ([]*explorer.Mquery, []error) {
	equeries := map[string]*explorer.ExecutionQuery{}
	mqueries := map[string]*explorer.Mquery{}
	conf := mqlc.NewConfig(runtime.Schema(), cnquery.DefaultFeatures)
	for i := range queries {
		query := queries[i]
		code, err := query.Compile(nil, conf)
		// Errors for filter queries are common when they reference resources for
		// providers that are not found on the system.
		if err != nil {
			log.Debug().Err(err).Str("mql", query.Mql).Msg("skipping filter query, not supported")
			continue
		}

		equeries[code.CodeV2.Id] = &explorer.ExecutionQuery{
			Query: query.Mql,
			Code:  code,
		}
		mqueries[code.CodeV2.Id] = query
	}

	instance := newInstance(runtime, nil)
	err := instance.runCode(equeries, timeout)
	if err != nil {
		return nil, []error{err}
	}

	instance.WaitUntilDone(timeout)

	var errs []error
	res := []*explorer.Mquery{}
	for _, equery := range equeries {
		bundle := equery.Code
		entrypoints := bundle.EntrypointChecksums()

		allTrue := true
		for j := range entrypoints {
			ep := entrypoints[j]
			res := instance.results[ep]
			if isTrue, _ := res.Data.IsSuccess(); !isTrue {
				allTrue = false
			}
		}

		if allTrue {
			query, ok := mqueries[bundle.CodeV2.Id]
			if ok {
				res = append(res, query)
			} else {
				errs = append(errs, errors.New("cannot find filter-query for result of bundle "+bundle.CodeV2.Id))
			}
		}
	}

	return res, errs
}

func (e *instance) runCode(queries map[string]*explorer.ExecutionQuery, timeout time.Duration) error {
	if len(queries) == 0 {
		e.progressReporter.Completed()
		go func() {
			e.done <- struct{}{}
		}()
		return nil
	}

	e.execs = make(map[string]*llx.MQLExecutorV2, len(queries))

	for i := range queries {
		query := queries[i]
		bundle := query.Code

		e.queries[bundle.CodeV2.Id] = query

		checksums := bundle.DatapointChecksums()
		for j := range checksums {
			e.datapointTracker[checksums[j]] = nil
		}

		checksums = bundle.EntrypointChecksums()
		for j := range checksums {
			e.datapointTracker[checksums[j]] = nil
		}

		for _, codeId := range query.Properties {
			arr := e.notifyQuery[codeId]
			arr = append(arr, query)
			e.notifyQuery[codeId] = arr
		}
	}

	// we need to only retain the checksums that notify other queries
	// to be run later on
	for codeID := range e.notifyQuery {
		query := queries[codeID]
		checksums := query.Code.EntrypointChecksums()

		for k := range checksums {
			checksum := checksums[k]

			arr := e.datapointTracker[checksum]
			arr = append(arr, query)
			e.datapointTracker[checksum] = arr
		}
	}

	var errs multierr.Errors
	for i := range queries {
		query := queries[i]
		if len(query.Properties) != 0 {
			continue
		}

		if err := e.runQuery(query.Code, nil); err != nil {
			errs.Add(err)
		}
	}

	return errs.Deduplicate()
}

// One instance of the executor. May be returned but not instantiated
// from outside this package.
type instance struct {
	runtime llx.Runtime
	// raw list of executino queries mapped via CodeID
	queries map[string]*explorer.ExecutionQuery
	// an optional list of datapoints as an allow-list of data that will be returned
	datapoints map[string]*explorer.DataQueryInfo
	// a tracker for all datapoints, that also references the queries that
	// created them
	datapointTracker map[string][]*explorer.ExecutionQuery
	// all code executors that have been started
	execs map[string]*llx.MQLExecutorV2
	// raw results from CodeID to result
	results map[string]*llx.RawResult
	// identifies which queries (CodeID) trigger other queries
	// this is used for properties, where a prop notifies a query that uses it
	notifyQuery      map[string][]*explorer.ExecutionQuery
	mutex            sync.Mutex
	isAborted        bool
	isDone           bool
	errors           error
	done             chan struct{}
	progressReporter progress.Progress
	collector        explorer.QueryConductor
	assetMrn         string
}

func newInstance(runtime llx.Runtime, progressReporter progress.Progress) *instance {
	if progressReporter == nil {
		progressReporter = progress.Noop{}
	}

	return &instance{
		runtime:          runtime,
		datapointTracker: map[string][]*explorer.ExecutionQuery{},
		queries:          map[string]*explorer.ExecutionQuery{},
		results:          map[string]*llx.RawResult{},
		notifyQuery:      map[string][]*explorer.ExecutionQuery{},
		isAborted:        false,
		isDone:           false,
		done:             make(chan struct{}),
		progressReporter: progressReporter,
		assetMrn:         runtime.AssetMRN(),
	}
}

func (e *instance) runQuery(bundle *llx.CodeBundle, props map[string]*llx.Primitive) error {
	exec, err := llx.NewExecutorV2(bundle.CodeV2, e.runtime, props, e.collect)
	if err != nil {
		return err
	}

	err = exec.Run()
	if err != nil {
		return err
	}

	e.execs[bundle.CodeV2.Id] = exec
	return nil
}

func (e *instance) WaitUntilDone(timeout time.Duration) error {
	select {
	case <-e.done:
		return nil

	case <-time.After(timeout):
		e.mutex.Lock()
		e.isAborted = true
		isDone := e.isDone
		e.mutex.Unlock()

		if isDone {
			return nil
		}
		return errors.New("execution timed out after " + timeout.String())
	}
}

func (e *instance) snapshotResults() map[string]*llx.Result {
	if e.datapoints != nil {
		e.mutex.Lock()
		results := make(map[string]*llx.Result, len(e.datapoints))
		for id := range e.datapoints {
			c := e.results[id]
			if c != nil {
				results[id] = c.Result()
			}
		}
		e.mutex.Unlock()
		return results
	}

	e.mutex.Lock()
	results := make(map[string]*llx.Result, len(e.results))
	for id, v := range e.results {
		results[id] = v.Result()
	}
	e.mutex.Unlock()
	return results
}

func (e *instance) StoreQueryData() error {
	if e.collector == nil {
		return errors.New("cannot store data, no collector provided")
	}

	_, err := e.collector.StoreResults(context.Background(), &explorer.StoreResultsReq{
		AssetMrn: e.assetMrn,
		Data:     e.snapshotResults(),
	})

	return err
}

func (e *instance) isCollected(query *llx.CodeBundle) bool {
	checksums := query.EntrypointChecksums()
	for i := range checksums {
		checksum := checksums[i]
		if _, ok := e.results[checksum]; !ok {
			return false
		}
	}

	return true
}

func (e *instance) getProps(query *explorer.ExecutionQuery) (map[string]*llx.Primitive, error) {
	res := map[string]*llx.Primitive{}

	for name, queryID := range query.Properties {
		query, ok := e.queries[queryID]
		if !ok {
			return nil, errors.New("cannot find running process for properties of query " + query.Code.Source)
		}

		eps := query.Code.EntrypointChecksums()
		checksum := eps[0]
		result := e.results[checksum]
		if result == nil {
			return nil, errors.New("cannot find result for property of query " + query.Code.Source)
		}

		res[name] = result.Result().Data
	}

	return res, nil
}

func (e *instance) collect(res *llx.RawResult) {
	var runQueries []*explorer.ExecutionQuery

	e.mutex.Lock()

	e.results[res.CodeID] = res
	cur := len(e.results)
	max := len(e.datapointTracker)
	isDone := cur == max
	e.isDone = isDone
	isAborted := e.isAborted
	e.progressReporter.OnProgress(cur, max)
	if isDone {
		e.progressReporter.Completed()
	}

	// collect all the queries we need to notify + update that list to remove
	// any query that we are about to start (all while inside of mutex lock)
	queries := e.datapointTracker[res.CodeID]
	if len(queries) != 0 {
		remaining := []*explorer.ExecutionQuery{}
		for j := range queries {
			if !e.isCollected(queries[j].Code) {
				remaining = append(remaining, queries[j])
				continue
			}

			codeID := queries[j].Code.CodeV2.Id
			notified := e.notifyQuery[codeID]
			for k := range notified {
				runQueries = append(runQueries, notified[k])
			}
		}
		e.datapointTracker[res.CodeID] = remaining
	}

	e.mutex.Unlock()

	if len(runQueries) != 0 {
		var fatalErr error
		for i := range runQueries {
			query := runQueries[i]
			props, err := e.getProps(query)
			if err != nil {
				fatalErr = err
				break
			}

			err = e.runQuery(query.Code, props)
			if err != nil {
				fatalErr = err
				break
			}
		}

		if fatalErr != nil {
			e.mutex.Lock()
			e.errors = errors.Join(e.errors, fatalErr)
			e.isAborted = true
			isAborted = true
			e.mutex.Unlock()
		}
	}

	if isDone && !isAborted {
		go func() {
			e.done <- struct{}{}
		}()
	}
}
