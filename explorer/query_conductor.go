package explorer

import (
	"context"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	llx "go.mondoo.com/cnquery/llx"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
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

func (s *LocalServices) SetProps(ctx context.Context, req *PropsReq) (*Empty, error) {
	// validate that the queries compile and fill in checksums
	for i := range req.Props {
		prop := req.Props[i]
		code, err := prop.RefreshChecksumAndType()
		if err != nil {
			return nil, err
		}
		prop.CodeId = code.CodeV2.Id
	}

	return globalEmpty, s.DataLake.SetProps(ctx, req)
}

// Resolve executable bits for an asset (via asset filters)
func (s *LocalServices) Resolve(ctx context.Context, req *ResolveReq) (*ResolvedPack, error) {
	if s.Upstream != nil && !s.Incognito {
		res, err := s.Upstream.Resolve(ctx, req)
		if err != nil {
			return nil, err
		}

		err = s.DataLake.SetResolvedPack(req.EntityMrn, res.FiltersChecksum, res)
		if err != nil {
			return nil, err
		}

		err = s.DataLake.SetAssetResolvedPack(ctx, req.EntityMrn, res, V2Code)
		return res, err
	}

	bundle, err := s.DataLake.GetBundle(ctx, req.EntityMrn)
	if err != nil {
		return nil, err
	}

	filtersChecksum, err := MatchFilters(req.EntityMrn, req.AssetFilters, bundle.Packs)
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
		if pack.Filters == nil {
			continue
		}
		for k := range pack.Filters.Items {
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

		props := NewPropsCache()
		props.Add(bundle.Props...)

		for i := range pack.Queries {
			query := pack.Queries[i]

			if query.Filter != nil {
				supported := true
				for codeID := range query.Filter.Items {
					if _, ok := supportedFilters[codeID]; !ok {
						supported = false
						break
					}
				}
				if !supported {
					continue
				}
			}

			err := s.addQuery(ctx, &job, query, props)
			if err != nil {
				return nil, err
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

func (s *LocalServices) addQuery(ctx context.Context, job *ExecutionJob, query *Mquery, propsCache PropsCache) error {
	var props map[string]*llx.Primitive
	var propRefs map[string]string
	if len(query.Props) != 0 {
		props = map[string]*llx.Primitive{}
		propRefs = map[string]string{}

		for i := range query.Props {
			prop, name, err := propsCache.Get(ctx, query.Props[i].Mrn)
			if err != nil {
				return errors.Wrap(err, "failed to get property for query "+query.Mrn)
			}

			props[name] = &llx.Primitive{Type: prop.Type}
			propRefs[name] = prop.CodeId

			if _, ok := job.Queries[prop.CodeId]; ok {
				continue
			}

			code, err := prop.Compile(nil)
			if err != nil {
				return errors.Wrap(err, "failed to compile property for query "+query.Mrn)
			}
			job.Queries[prop.CodeId] = &ExecutionQuery{
				Query:    prop.Mql,
				Checksum: prop.Checksum,
				Code:     code,
			}
		}
	}

	bundle, err := query.Compile(props)
	if err != nil {
		return err
	}

	equery := &ExecutionQuery{
		Query:      query.Mql,
		Checksum:   query.Checksum,
		Code:       bundle,
		Properties: propRefs,
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

	return nil
}

// MatchFilters will take the list of filters and only return the ones
// that are supported by the bundle.
func MatchFilters(entityMrn string, filters []*Mquery, packs []*QueryPack) (string, error) {
	supported := map[string]*Mquery{}
	for i := range packs {
		pack := packs[i]
		if pack.Filters == nil {
			continue
		}

		for k, v := range pack.Filters.Items {
			supported[k] = v
		}
	}

	matching := []*Mquery{}
	for i := range filters {
		cur := filters[i]

		if _, ok := supported[cur.CodeId]; ok {
			curCopy := proto.Clone(cur).(*Mquery)
			curCopy.Mrn = entityMrn + "/assetfilter/" + cur.CodeId
			curCopy.Title = curCopy.Query
			matching = append(matching, curCopy)
		}
	}

	if len(matching) == 0 {
		return "", newAssetMatchError(entityMrn, filters, supported)
	}

	sum, err := ChecksumFilters(matching)
	if err != nil {
		return "", err
	}

	return sum, nil
}

func newAssetMatchError(mrn string, filters []*Mquery, supportedFilters map[string]*Mquery) error {
	if len(filters) == 0 {
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
		supported[i] = v.Mql
		i++
	}

	filtersMql := make([]string, len(filters))
	for i := range filters {
		filtersMql[i] = strings.TrimSpace(filters[i].Mql)
	}

	sort.Strings(filtersMql)
	sort.Strings(supported)

	msg := "asset does not support any of these query packs\nfilters supported:\n" + strings.Join(supported, ",\n") + "\n\nasset supports the following filters:\n" + strings.Join(filtersMql, ",\n")
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

func (s *LocalServices) SynchronizeAssets(context.Context, *SynchronizeAssetsReq) (*SynchronizeAssetsResp, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
