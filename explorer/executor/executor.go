package executor

import (
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/cli/progress"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/resources"
)

func RunExecutionJob(
	schema *resources.Schema, runtime *resources.Runtime, collectorSvc explorer.QueryConductor, assetMrn string,
	job *explorer.ExecutionJob, features cnquery.Features, progressFn progress.Progress,
) (*instance, error) {
	// We are setting a sensible default timeout for jobs here. This will need
	// user-configuration.
	timeout := 30 * time.Minute

	bundles := make([]*llx.CodeBundle, len(job.Queries))
	i := 0
	for _, query := range job.Queries {
		bundles[i] = query.Code
		i++
	}

	return runCode(schema, runtime, bundles, timeout, progressFn)
}

func RunFilterQueries(
	schema *resources.Schema, runtime *resources.Runtime,
	queries []*explorer.Mquery, timeout time.Duration,
) ([]*explorer.Mquery, []error) {
	errs := []error{}
	bundles := []*llx.CodeBundle{}
	mqueries := map[string]*explorer.Mquery{}
	for i := range queries {
		query := queries[i]
		bundle, err := query.Compile(nil)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		bundles = append(bundles, bundle)
		mqueries[bundle.CodeV2.Id] = query
	}
	if len(errs) != 0 {
		return nil, errs
	}

	instance, err := runCode(schema, runtime, bundles, timeout, progress.Noop{})
	if err != nil {
		return nil, []error{err}
	}

	instance.WaitUntilDone(timeout)

	res := []*explorer.Mquery{}
	for i := range bundles {
		bundle := bundles[i]
		entrypoints := bundle.EntrypointChecksums(true)

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

func runCode(
	schema *resources.Schema, runtime *resources.Runtime,
	bundles []*llx.CodeBundle, timeout time.Duration,
	progress progress.Progress,
) (*instance, error) {
	instance := &instance{
		schema:           schema,
		runtime:          runtime,
		execs:            make(map[string]*llx.MQLExecutorV2, len(bundles)),
		datapointTracker: map[string]struct{}{},
		results:          map[string]*llx.RawResult{},
		isAborted:        false,
		isDone:           false,
		done:             make(chan struct{}),
		progress:         progress,
	}

	for i := range bundles {
		bundle := bundles[i]

		checksums := bundle.DatapointChecksums(true)
		for i := range checksums {
			instance.datapointTracker[checksums[i]] = struct{}{}
		}
		checksums = bundle.EntrypointChecksums(true)
		for i := range checksums {
			instance.datapointTracker[checksums[i]] = struct{}{}
		}
	}

	var errs error
	for i := range bundles {
		bundle := bundles[i]

		exec, err := llx.NewExecutorV2(bundle.CodeV2, runtime, nil, instance.collect)
		if err != nil {
			multierror.Append(errs, err)
			continue
		}

		err = exec.Run()
		if err != nil {
			multierror.Append(errs, err)
			continue
		}

		instance.execs[bundle.CodeV2.Id] = exec
	}

	return instance, errs
}

// One instance of the executor. May be returned but not instantiated
// from outside this package.
type instance struct {
	schema           *resources.Schema
	runtime          *resources.Runtime
	datapointTracker map[string]struct{}
	execs            map[string]*llx.MQLExecutorV2
	results          map[string]*llx.RawResult
	mutex            sync.Mutex
	isAborted        bool
	isDone           bool
	done             chan struct{}
	progress         progress.Progress
}

func (i *instance) WaitUntilDone(timeout time.Duration) error {
	select {
	case <-i.done:
		return nil

	case <-time.After(timeout):
		i.mutex.Lock()
		i.isAborted = true
		isDone := i.isDone
		i.mutex.Unlock()

		if isDone {
			return nil
		}
		return errors.New("execution timed out after " + timeout.String())
	}
}

func (i *instance) collect(res *llx.RawResult) {
	i.mutex.Lock()
	i.results[res.CodeID] = res
	cur := len(i.results)
	max := len(i.datapointTracker)
	isDone := cur == max
	i.isDone = isDone
	i.progress.OnProgress(cur, max)
	isAborted := i.isAborted
	i.mutex.Unlock()

	if isDone && !isAborted {
		go func() {
			i.done <- struct{}{}
		}()
	}
}
