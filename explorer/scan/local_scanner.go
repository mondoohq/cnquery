package scan

import (
	"context"
	"errors"

	"github.com/gogo/status"
	"go.mondoo.com/cnquery/explorer"
	"google.golang.org/grpc/codes"
)

type LocalScanner struct {
	ctx context.Context
}

func NewLocalScanner() *LocalScanner {
	return &LocalScanner{}
}

func (s *LocalScanner) RunIncognito(ctx context.Context, job *Job) (*explorer.ReportCollection, error) {
	if job == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing scan job")
	}

	if job.Inventory == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing inventory")
	}

	if ctx == nil {
		return nil, errors.New("no context provided to run job with local scanner")
	}

	return nil, errors.New("runincognito scan NOT YET IMPLEMENTED")
}
