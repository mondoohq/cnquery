package explorer

import (
	"context"
	"errors"
	"os"

	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/ranger-rpc"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
	"go.opentelemetry.io/otel"
)

const defaultQueryHubUrl = "https://hub.api.mondoo.com"

var tracer = otel.Tracer("go.mondoo.com/cnquery/explorer")

// ValidateBundle and check queries, relationships, MRNs, and versions
func (s *LocalServices) ValidateBundle(ctx context.Context, bundle *Bundle) (*Empty, error) {
	_, err := bundle.Compile(ctx)
	return globalEmpty, err
}

// SetBundle stores a bundle of query packs and queries in this marketplace
func (s *LocalServices) SetBundle(ctx context.Context, bundle *Bundle) (*Empty, error) {
	bundlemap, err := bundle.Compile(ctx)
	if err != nil {
		return globalEmpty, err
	}

	if err := s.setBundleFromMap(ctx, bundlemap); err != nil {
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

	filters, err := querypack.ComputeAssetFilters(ctx, querypack.Mrn)
	if err != nil {
		return nil, nil, err
	}
	querypack.AssetFilters = map[string]*Mquery{}
	for i := range filters {
		cur := filters[i]
		querypack.AssetFilters[cur.CodeId] = cur
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

func (s *LocalServices) setBundleFromMap(ctx context.Context, bundle *BundleMap) error {
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
	_, err = s.cacheUpstreamQueryPackBundle(ctx, in.Mrn)
	if err != nil {
		return nil, err
	}
	return s.DataLake.GetQueryPack(ctx, in.Mrn)
}

// GetQueryPack for a given MRN
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
	return s.cacheUpstreamQueryPackBundle(ctx, in.Mrn)
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

// List all query packs for a given owner
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

// DefaultPacks retrieves a list of default packs for a given asset
func (s *LocalServices) DefaultPacks(ctx context.Context, req *DefaultPacksReq) (*URLs, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "no filters provided")
	}

	if s.Upstream != nil {
		return s.Upstream.DefaultPacks(ctx, req)
	}

	queryHubURL := os.Getenv("QUERYHUB_URL")
	if queryHubURL == "" {
		queryHubURL = defaultQueryHubUrl
	}

	client, err := NewQueryHubClient(queryHubURL, ranger.DefaultHttpClient())
	if err != nil {
		return nil, err
	}
	return client.DefaultPacks(ctx, req)
}

// HELPER METHODS
// =================

// cacheUpstreamQueryPackBundle by storing a copy of the upstream pack in this db
// Note: upstream has to be defined
func (s *LocalServices) cacheUpstreamQueryPackBundle(ctx context.Context, mrn string) (*Bundle, error) {
	logCtx := logger.FromContext(ctx)
	if s.Upstream == nil {
		return nil, errors.New("failed to retrieve upstream query pack " + mrn + " since upstream is not defined")
	}

	logCtx.Debug().Str("querypack", mrn).Msg("query.hub> fetch query pack from upstream")
	bundle, err := s.Upstream.GetBundle(ctx, &Mrn{Mrn: mrn})
	if err != nil {
		logCtx.Error().Err(err).Str("querypack", mrn).Msg("query.hub> failed to retrieve query pack from upstream")
		return nil, errors.New("failed to retrieve upstream query pack " + mrn + ": " + err.Error())
	}

	bundleMap := bundle.ToMap()
	if err = s.setBundleFromMap(ctx, bundleMap); err != nil {
		logCtx.Error().Err(err).Str("querypack", mrn).Msg("query.hub> failed to set query pack retrieved from upstream")
		return nil, err
	}

	// we need to assign the bundles to the asset
	querypackMrns := []string{}
	for k := range bundleMap.Packs {
		querypackMrns = append(querypackMrns, k)
	}

	// assign a query pack locally
	deltas := map[string]*AssignmentDelta{}
	for i := range querypackMrns {
		packMrn := querypackMrns[i]
		deltas[packMrn] = &AssignmentDelta{
			Mrn:    packMrn,
			Action: AssignmentDelta_ADD,
		}
	}

	s.DataLake.EnsureAsset(ctx, mrn)
	_, err = s.DataLake.MutateBundle(ctx, &BundleMutationDelta{
		OwnerMrn: mrn,
		Deltas:   deltas,
	}, true)

	logCtx.Debug().Str("querypack", mrn).Msg("query.hub> fetched bundle from upstream")
	return bundle, nil
}
