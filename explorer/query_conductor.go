package explorer

import (
	"context"
	"sort"
	"strings"

	"github.com/gogo/status"
	"github.com/rs/zerolog/log"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
)

func (s *LocalServices) Assign(ctx context.Context, assignment *Assignment) (*Empty, error) {
	if len(assignment.PackMrns) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no query pack MRNs were provided")
	}

	// all remote, call upstream
	if s.Upstream != nil && !s.Incognito {
		return s.Upstream.QueryConductor.Assign(ctx, assignment)
	}

	// cache everything from upstream
	if s.Upstream != nil && s.Incognito {
		// NOTE: we request the packs to cache them
		for i := range assignment.PackMrns {
			mrn := assignment.PackMrns[i]
			_, err := s.GetQueryPack(ctx, &Mrn{
				Mrn: mrn,
			})
			if err != nil {
				return nil, err
			}
		}
	}

	// assign a query pack locally
	deltas := map[string]*AssignmentDelta{}
	for i := range assignment.PackMrns {
		packMrn := assignment.PackMrns[i]
		deltas[packMrn] = &AssignmentDelta{
			Mrn:    packMrn,
			Action: AssignmentDelta_ADD,
		}
	}

	s.DataLake.EnsureAsset(ctx, assignment.AssetMrn)

	_, err := s.DataLake.MutateBundle(ctx, &BundleMutationDelta{
		OwnerMrn: assignment.AssetMrn,
		Deltas:   deltas,
	}, true)
	return globalEmpty, err
}

func (s *LocalServices) Unassign(ctx context.Context, assignment *Assignment) (*Empty, error) {
	if len(assignment.PackMrns) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no query pack MRNs were provided")
	}

	// all remote, call upstream
	if s.Upstream != nil && !s.Incognito {
		return s.Upstream.QueryConductor.Unassign(ctx, assignment)
	}

	deltas := map[string]*AssignmentDelta{}
	for i := range assignment.PackMrns {
		packMrn := assignment.PackMrns[i]
		deltas[packMrn] = &AssignmentDelta{
			Mrn:    packMrn,
			Action: AssignmentDelta_DELETE,
		}
	}

	_, err := s.DataLake.MutateBundle(ctx, &BundleMutationDelta{
		OwnerMrn: assignment.AssetMrn,
		Deltas:   deltas,
	}, true)
	return globalEmpty, err
}

// Resolve executable bits for an asset (via asset filters)
func (s *LocalServices) Resolve(ctx context.Context, req *ResolveReq) (*ResolvedPack, error) {
	if s.Upstream != nil && !s.Incognito {
		return s.Upstream.Resolve(ctx, req)
	}

	bundle, err := s.DataLake.GetBundle(ctx, req.EntityMrn)
	if err != nil {
		return nil, err
	}

	filtersChecksum, err := MatchAssetFilters(req.EntityMrn, req.AssetFilters, bundle.Packs)
	if err != nil {
		return nil, err
	}

	supportedFilters := make(map[string]struct{}, len(req.AssetFilters))
	for i := range req.AssetFilters {
		f := req.AssetFilters[i]
		supportedFilters[f.CodeId] = struct{}{}
	}
	applicablePacks := []*QueryPack{}
	for i := range bundle.Packs {
		pack := bundle.Packs[i]
		for k := range pack.AssetFilters {
			if _, ok := supportedFilters[k]; ok {
				applicablePacks = append(applicablePacks, pack)
				break
			}
		}
	}

	job := ExecutionJob{
		Queries:    make(map[string]*ExecutionQuery),
		Datapoints: make(map[string]*DataQueryInfo),
	}
	for i := range applicablePacks {
		pack := applicablePacks[i]
		for i := range pack.Queries {
			query := pack.Queries[i]
			equery, err := query2executionQuery(query)
			if err != nil {
				return nil, err
			}

			code := equery.Code.CodeV2
			refs := append(code.Datapoints(), code.Entrypoints()...)

			job.Queries[query.CodeId] = equery
			for i := range refs {
				ref := refs[i]
				checksum := code.Checksums[ref]
				typ := code.Chunk(ref).DereferencedTypeV2(code)

				job.Datapoints[checksum] = &DataQueryInfo{
					Type: string(typ),
				}
			}
		}
	}

	res := &ResolvedPack{
		ExecutionJob: &job,
	}

	err = s.DataLake.SetResolvedPack(req.EntityMrn, filtersChecksum, res)
	if err != nil {
		return nil, err
	}

	err = s.DataLake.SetAssetResolvedPack(ctx, req.EntityMrn, res, V2Code)
	return res, err
}

func query2executionQuery(query *Mquery) (*ExecutionQuery, error) {
	bundle, err := query.Compile(nil)
	if err != nil {
		return nil, err
	}

	return &ExecutionQuery{
		Query:    query.Query,
		Checksum: query.Checksum,
		Code:     bundle,
	}, nil
}

// MatchAssetFilters will take the list of filters and only return the ones
// that are supported by the bundle.
func MatchAssetFilters(entityMrn string, assetFilters []*Mquery, packs []*QueryPack) (string, error) {
	supported := map[string]*Mquery{}
	for i := range packs {
		for k, v := range packs[i].AssetFilters {
			supported[k] = v
		}
	}

	matching := []*Mquery{}
	for i := range assetFilters {
		cur := assetFilters[i]

		if _, ok := supported[cur.CodeId]; ok {
			curCopy := proto.Clone(cur).(*Mquery)
			curCopy.Mrn = entityMrn + "/assetfilter/" + cur.CodeId
			curCopy.Title = curCopy.Query
			matching = append(matching, curCopy)
		}
	}

	if len(matching) == 0 {
		return "", newAssetMatchError(entityMrn, assetFilters, supported)
	}

	sum, err := ChecksumAssetFilters(matching)
	if err != nil {
		return "", err
	}

	return sum, nil
}

func newAssetMatchError(mrn string, assetFilters []*Mquery, supportedFilters map[string]*Mquery) error {
	if len(assetFilters) == 0 {
		// send a proto error with details, so that the agent can render it properly
		msg := "asset does not match any of the activated query packs"
		st := status.New(codes.InvalidArgument, msg)

		std, err := st.WithDetails(&errdetails.ErrorInfo{
			Domain: SERVICE_NAME,
			Reason: "no-matching-packs",
			Metadata: map[string]string{
				"mrn": mrn,
			},
		})
		if err != nil {
			log.Error().Err(err).Msg("could not send status with additional information")
			return st.Err()
		}
		return std.Err()
	}

	supported := make([]string, len(supportedFilters))
	i := 0
	for _, v := range supportedFilters {
		supported[i] = v.Query
		i++
	}

	filters := make([]string, len(assetFilters))
	for i := range assetFilters {
		filters[i] = strings.TrimSpace(assetFilters[i].Query)
	}

	sort.Strings(filters)
	sort.Strings(supported)

	msg := "asset does not support any of these query packs\nfilters supported:\n" + strings.Join(supported, ",\n") + "\n\nasset supports the following filters:\n" + strings.Join(filters, ",\n")
	return status.Error(codes.InvalidArgument, msg)
}

func (s *LocalServices) StoreResults(ctx context.Context, req *StoreResultsReq) (*Empty, error) {
	_, err := s.DataLake.UpdateData(ctx, req.AssetMrn, req.Data)
	if err != nil {
		return globalEmpty, err
	}

	if s.Upstream != nil && !s.Incognito {
		_, err := s.Upstream.QueryConductor.StoreResults(ctx, req)
		if err != nil {
			return globalEmpty, err
		}
	}

	return globalEmpty, nil
}

func (s *LocalServices) GetReport(ctx context.Context, req *EntityDataRequest) (*Report, error) {
	return s.DataLake.GetReport(ctx, req.EntityMrn, req.DataMrn)
}
