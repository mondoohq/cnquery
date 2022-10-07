package executor

import (
	"errors"
	"time"

	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/cli/progress"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/resources"
)

// One instance of the executor. May be returned but not instantiated
// from outside this package.
type instance struct{}

func (i *instance) WaitUntilDone(timeout time.Duration) error {
	return errors.New("Executor is NOT YET IMPLEMENTED (wait until done)")
}

func ExecuteResolvedPack(
	schema *resources.Schema, runtime *resources.Runtime, collectorSvc explorer.QueryConductor, assetMrn string,
	job *explorer.ExecutionJob, features cnquery.Features, progressFn progress.Progress,
) (*instance, error) {
	return nil, errors.New("Executing a resolved pack NOT YET IMPLEMENTED")
}

func ExecuteFilterQueries(
	schema *resources.Schema, runtime *resources.Runtime,
	queries []*explorer.Mquery, timeout time.Duration,
) ([]*explorer.Mquery, []error) {
	return nil, []error{errors.New("Execute filter queries is NOT YET IMPLEMENTED")}
}
