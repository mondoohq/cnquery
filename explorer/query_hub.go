package explorer

import (
	"context"
	"errors"

	"github.com/gogo/status"
	"go.mondoo.com/cnquery/logger"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc/codes"
)

var tracer = otel.Tracer("go.mondoo.com/cnquery/explorer")

// ValidateBundle and check queries, relationships, MRNs, and versions
func (s *LocalServices) ValidateBundle(ctx context.Context, bundle *Bundle) (*Empty, error) {
	_, err := bundle.Compile(ctx)
	return globalEmpty, err
}

// SetBundle stores a bundle of policies and queries in this marketplace
func (s *LocalServices) SetBundle(ctx context.Context, bundle *Bundle) (*Empty, error) {
	if len(bundle.OwnerMrn) == 0 {
		return globalEmpty, status.Error(codes.InvalidArgument, "owner MRN is required")
	}

	bundlemap, err := bundle.Compile(ctx)
	if err != nil {
		return globalEmpty, err
	}

	if err := s.setAllQueryPacks(ctx, bundlemap); err != nil {
		return nil, err
	}

	return globalEmpty, nil
}

// PreparePack takes a query pack and an optional bundle and gets it
// ready to be saved in the DB, including asset filters.
// Note1: The bundle must have been pre-compiled and validated!
func (s *LocalServices) PreparePack(ctx context.Context, querypack *QueryPack) (*QueryPack, []*Mquery, error) {
	logCtx := logger.FromContext(ctx)

	if querypack == nil || len(querypack.Mrn) == 0 {
		return nil, nil, status.Error(codes.InvalidArgument, "mrn is required")
	}

	if len(querypack.OwnerMrn) == 0 {
		return nil, nil, status.Error(codes.InvalidArgument, "owner mrn is required")
	}

	// store all queries
	for i := range querypack.Queries {
		q := querypack.Queries[i]
		if err := s.setQuery(ctx, q.Mrn, q, false); err != nil {
			return nil, nil, err
		}
	}

	if querypack.LocalExecutionChecksum == "" || querypack.LocalContentChecksum == "" {
		logCtx.Trace().Str("querypack", querypack.Mrn).Msg("hub> update checksum")
		if err := querypack.UpdateChecksums(); err != nil {
			return nil, nil, err
		}
	}

	filters, err := querypack.ComputeAssetFilters(ctx)
	if err != nil {
		return nil, nil, err
	}

	return querypack, filters, nil
}

func (s *LocalServices) setPack(ctx context.Context, querypack *QueryPack) error {
	querypack, filters, err := s.PreparePack(ctx, querypack)
	if err != nil {
		return err
	}

	err = s.DataLake.SetQueryPack(ctx, querypack, filters)
	if err != nil {
		return err
	}

	return nil
}

func (s *LocalServices) setAllQueryPacks(ctx context.Context, bundle *BundleMap) error {
	logCtx := logger.FromContext(ctx)

	var err error
	for i := range bundle.Packs {
		querypack := bundle.Packs[i]
		logCtx.Debug().Str("owner", querypack.OwnerMrn).Str("uid", querypack.Uid).Str("mrn", querypack.Mrn).Msg("store query pack")
		querypack.OwnerMrn = bundle.OwnerMrn

		// If this is user generated, it must be non-public
		if bundle.OwnerMrn != "//querypack.api.mondoo.app" {
			querypack.IsPublic = false
		}

		if err = s.setPack(ctx, querypack); err != nil {
			return err
		}
	}

	return nil
}

func (s *LocalServices) setQuery(ctx context.Context, mrn string, query *Mquery, isScored bool) error {
	if query == nil {
		return errors.New("cannot set query '" + mrn + "' as it is not defined")
	}

	if query.Title == "" {
		query.Title = query.Query
	}

	return s.DataLake.SetQuery(ctx, mrn, query)
}

// GetQueryPack for a given MRN
func (s *LocalServices) GetQueryPack(ctx context.Context, in *Mrn) (*QueryPack, error) {
	logCtx := logger.FromContext(ctx)

	if in == nil || len(in.Mrn) == 0 {
		return nil, status.Error(codes.InvalidArgument, "mrn is required")
	}

	b, err := s.DataLake.GetQueryPack(ctx, in.Mrn)
	if err == nil {
		logCtx.Debug().Str("querypack", in.Mrn).Err(err).Msg("query.hub> get query pack from db")
		return b, nil
	}
	if s.Upstream == nil {
		return nil, err
	}

	// try upstream; once it's cached, try again
	_, err = s.cacheUpstreamQueryPack(ctx, in.Mrn)
	if err != nil {
		return nil, err
	}
	return s.DataLake.GetQueryPack(ctx, in.Mrn)
}

// GetBundle retrieves the given bundle and all its dependencies (policies/queries)
func (s *LocalServices) GetBundle(ctx context.Context, in *Mrn) (*Bundle, error) {
	if in == nil || len(in.Mrn) == 0 {
		return nil, status.Error(codes.InvalidArgument, "mrn is required")
	}

	b, err := s.DataLake.GetBundle(ctx, in.Mrn)
	if err == nil {
		return b, nil
	}
	if s.Upstream == nil {
		return nil, err
	}

	// try upstream
	return s.cacheUpstreamQueryPack(ctx, in.Mrn)
}

// GetFilters retrieves the asset filter queries for a given query pack
func (s *LocalServices) GetFilters(ctx context.Context, mrn *Mrn) (*Mqueries, error) {
	if mrn == nil || len(mrn.Mrn) == 0 {
		return nil, status.Error(codes.InvalidArgument, "mrn is required")
	}

	filters, err := s.DataLake.GetQueryPackFilters(ctx, mrn.Mrn)
	if err != nil {
		return nil, errors.New("failed to get filters: " + err.Error())
	}

	return &Mqueries{Items: filters}, nil
}

// List all policies for a given owner
func (s *LocalServices) List(ctx context.Context, filter *ListReq) (*QueryPacks, error) {
	if filter == nil {
		return nil, status.Error(codes.InvalidArgument, "need to provide a filter object for list")
	}

	if len(filter.OwnerMrn) == 0 {
		return nil, status.Error(codes.InvalidArgument, "a MRN for the owner is required")
	}

	res, err := s.DataLake.ListQueryPacks(ctx, filter.OwnerMrn, filter.Name)
	if err != nil {
		return nil, err
	}
	if res == nil {
		res = []*QueryPack{}
	}

	return &QueryPacks{
		Items: res,
	}, nil
}

// DeleteQueryPack removes a query pack via its given MRN
func (s *LocalServices) DeleteQueryPack(ctx context.Context, in *Mrn) (*Empty, error) {
	if in == nil || len(in.Mrn) == 0 {
		return nil, status.Error(codes.InvalidArgument, "mrn is required")
	}

	return globalEmpty, s.DataLake.DeleteQueryPack(ctx, in.Mrn)
}

// HELPER METHODS
// =================

// cacheUpstreamQueryPack by storing a copy of the upstream bundle in this db
// Note: upstream has to be defined
func (s *LocalServices) cacheUpstreamQueryPack(ctx context.Context, mrn string) (*Bundle, error) {
	logCtx := logger.FromContext(ctx)
	if s.Upstream == nil {
		return nil, errors.New("failed to retrieve upstream query pack " + mrn + " since upstream is not defined")
	}

	logCtx.Debug().Str("querypack", mrn).Msg("query.hub> fetch bundle from upstream")
	bundle, err := s.Upstream.GetBundle(ctx, &Mrn{Mrn: mrn})
	if err != nil {
		logCtx.Error().Err(err).Str("querypack", mrn).Msg("query.hub> failed to retrieve bundle from upstream")
		return nil, errors.New("failed to retrieve upstream query pack " + mrn + ": " + err.Error())
	}

	_, err = s.SetBundle(ctx, bundle)
	if err != nil {
		logCtx.Error().Err(err).Str("querypack", mrn).Msg("query.hub> failed to set bundle retrieved from upstream")
		return nil, errors.New("failed to cache upstream query pack " + mrn + ": " + err.Error())
	}

	logCtx.Debug().Str("querypack", mrn).Msg("query.hub> fetched bundle from upstream")
	return bundle, nil
}
